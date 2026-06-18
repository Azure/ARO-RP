package main

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const clusterResourceIDHeader = "X-ARO-Cluster-Resource-ID"
const maxLogAge = 10 * time.Minute

var receivedLogsPath string
var storageMu sync.Mutex

type receivedLogEntry struct {
	ReceivedAt        time.Time `json:"receivedAt"`
	ClusterResourceID string    `json:"clusterResourceID"`
	ContentType       string    `json:"contentType,omitempty"`
	PayloadSize       int       `json:"payloadSize"`
	PayloadBase64     string    `json:"payloadBase64"`
}

func main() {
	var err error
	receivedLogsPath, err = initializeStoragePath(getEnv("RECEIVED_LOGS_PATH", "/tmp/otlp-fake-receiver/received-logs.ndjson"))
	if err != nil {
		log.Fatalf("failed to initialize storage path: %v", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz/ready", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	mux.HandleFunc("/v1/logs", handleLogs)

	addr := getEnv("LISTEN_ADDR", ":4318")
	log.Printf("starting fake OTLP receiver on %s storing payloads in %s", addr, receivedLogsPath)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("receiver exited: %v", err)
	}
}

func handleLogs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	clusterID := r.Header.Get(clusterResourceIDHeader)
	if clusterID == "" {
		http.Error(w, clusterResourceIDHeader+" header is required", http.StatusUnauthorized)
		return
	}

	payload, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "failed to read body", http.StatusBadRequest)
		return
	}

	contentType := r.Header.Get("Content-Type")
	log.Printf("received /v1/logs size=%d contentType=%q clusterResourceID=%q", len(payload), contentType, clusterID)
	if os.Getenv("DUMP_PAYLOADS") == "true" {
		log.Printf("payload=%s", payload)
	}
	if err := persistReceivedLog(contentType, clusterID, payload); err != nil {
		http.Error(w, fmt.Sprintf("failed to persist payload: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"message": "accepted",
	})
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func initializeStoragePath(path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("storage path is empty")
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("creating directory %q: %w", dir, err)
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return "", fmt.Errorf("opening storage file %q: %w", path, err)
	}
	if err := file.Close(); err != nil {
		return "", fmt.Errorf("closing storage file %q: %w", path, err)
	}
	return path, nil
}

func persistReceivedLog(contentType, clusterID string, payload []byte) error {
	storageMu.Lock()
	defer storageMu.Unlock()

	now := time.Now().UTC()
	if err := purgeExpiredLogsLocked(now); err != nil {
		return err
	}

	entry := receivedLogEntry{
		ReceivedAt:        now,
		ClusterResourceID: clusterID,
		ContentType:       contentType,
		PayloadSize:       len(payload),
		PayloadBase64:     base64.StdEncoding.EncodeToString(payload),
	}
	raw, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("marshaling log entry: %w", err)
	}

	file, err := os.OpenFile(receivedLogsPath, os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("opening storage file %q: %w", receivedLogsPath, err)
	}
	defer file.Close()

	if _, err := file.Write(append(raw, '\n')); err != nil {
		return fmt.Errorf("writing storage file %q: %w", receivedLogsPath, err)
	}
	return nil
}

func purgeExpiredLogsLocked(now time.Time) error {
	in, err := os.Open(receivedLogsPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("opening storage file %q for purge: %w", receivedLogsPath, err)
	}
	defer in.Close()

	tmpPath := receivedLogsPath + ".tmp"
	out, err := os.OpenFile(tmpPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("creating temporary storage file %q: %w", tmpPath, err)
	}

	cutoff := now.Add(-maxLogAge)
	reader := bufio.NewReader(in)
	for {
		line, readErr := reader.ReadBytes('\n')
		if len(line) > 0 {
			keep, keepErr := shouldKeepLogLine(line, cutoff)
			if keepErr != nil {
				_ = out.Close()
				_ = os.Remove(tmpPath)
				return keepErr
			}
			if keep {
				if _, err := out.Write(append(bytes.TrimSpace(line), '\n')); err != nil {
					_ = out.Close()
					_ = os.Remove(tmpPath)
					return fmt.Errorf("writing temporary storage file %q: %w", tmpPath, err)
				}
			}
		}

		if readErr != nil {
			if errors.Is(readErr, io.EOF) {
				break
			}
			_ = out.Close()
			_ = os.Remove(tmpPath)
			return fmt.Errorf("reading storage file %q for purge: %w", receivedLogsPath, readErr)
		}
	}

	if err := out.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("closing temporary storage file %q: %w", tmpPath, err)
	}
	if err := os.Rename(tmpPath, receivedLogsPath); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("replacing storage file %q after purge: %w", receivedLogsPath, err)
	}
	return nil
}

func shouldKeepLogLine(line []byte, cutoff time.Time) (bool, error) {
	trimmed := bytes.TrimSpace(line)
	if len(trimmed) == 0 {
		return false, nil
	}

	var metadata struct {
		ReceivedAt time.Time `json:"receivedAt"`
	}
	if err := json.Unmarshal(trimmed, &metadata); err != nil {
		log.Printf("dropping malformed stored log entry: %v", err)
		return false, nil
	}

	return !metadata.ReceivedAt.Before(cutoff), nil
}
