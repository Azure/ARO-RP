package main

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	commonv1 "go.opentelemetry.io/proto/otlp/common/v1"
	collectorlogsv1 "go.opentelemetry.io/proto/otlp/collector/logs/v1"
	logsv1 "go.opentelemetry.io/proto/otlp/logs/v1"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

type receivedLogEntry struct {
	ReceivedAt        time.Time `json:"receivedAt"`
	ClusterResourceID string    `json:"clusterResourceID"`
	ContentType       string    `json:"contentType,omitempty"`
	PayloadSize       int       `json:"payloadSize"`
	PayloadBase64     string    `json:"payloadBase64"`
}

type kustoReadyLogEntry struct {
	ReceivedAt        time.Time         `json:"receivedAt"`
	ClusterResourceID string            `json:"clusterResourceID"`
	ContentType       string            `json:"contentType,omitempty"`
	ResourceAttrs     map[string]any    `json:"resourceAttributes,omitempty"`
	ScopeName         string            `json:"scopeName,omitempty"`
	ScopeVersion      string            `json:"scopeVersion,omitempty"`
	ScopeAttrs        map[string]any    `json:"scopeAttributes,omitempty"`
	LogAttrs          map[string]any    `json:"logAttributes,omitempty"`
	Body              any               `json:"body,omitempty"`
	SeverityText      string            `json:"severityText,omitempty"`
	SeverityNumber    logsv1.SeverityNumber `json:"severityNumber,omitempty"`
	Timestamp         string            `json:"timestamp,omitempty"`
	ObservedTimestamp string            `json:"observedTimestamp,omitempty"`
	TraceID           string            `json:"traceID,omitempty"`
	SpanID            string            `json:"spanID,omitempty"`
}

type importResult struct {
	ImportedRequests int `json:"importedRequests"`
	TranslatedLogs   int `json:"translatedLogs"`
	DroppedRequests  int `json:"droppedRequests"`
}

var outputPath string
var outputMu sync.Mutex
var kustoIngestURL string
var kustoAuthHeader string

func main() {
	var err error
	outputPath, err = initializeStoragePath(getEnv("KUSTO_READY_LOGS_PATH", "/tmp/otlp-fake-kusto-bridge/kusto-ready-logs.ndjson"))
	if err != nil {
		log.Fatalf("failed to initialize output path: %v", err)
	}
	kustoIngestURL = getEnv("KUSTO_INGEST_URL", "")
	kustoAuthHeader = getEnv("KUSTO_AUTH_HEADER", "")

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz/ready", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	mux.HandleFunc("/import/received-logs", handleImport)

	addr := getEnv("LISTEN_ADDR", ":8081")
	if kustoIngestURL == "" {
		log.Printf("starting fake OTLP->Kusto bridge on %s writing rows to %s (forwarding disabled)", addr, outputPath)
	} else {
		log.Printf("starting fake OTLP->Kusto bridge on %s writing rows to %s and forwarding to %s", addr, outputPath, kustoIngestURL)
	}
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("bridge exited: %v", err)
	}
}

func handleImport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "failed to read body", http.StatusBadRequest)
		return
	}

	rows, result, err := translateImportedPayload(body)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to translate input: %v", err), http.StatusBadRequest)
		return
	}
	if err := persistTranslatedRows(rows); err != nil {
		http.Error(w, fmt.Sprintf("failed to persist translated rows: %v", err), http.StatusInternalServerError)
		return
	}

	forwarded := false
	if kustoIngestURL != "" && len(rows) > 0 {
		if err := forwardToKusto(rows); err != nil {
			http.Error(w, fmt.Sprintf("failed to forward translated rows to kusto: %v", err), http.StatusBadGateway)
			return
		}
		forwarded = true
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"message": "imported",
		"result":  result,
		"output":  outputPath,
		"forwardedToKusto": forwarded,
	})
}

func translateImportedPayload(raw []byte) ([]kustoReadyLogEntry, importResult, error) {
	scanner := bufio.NewScanner(bytes.NewReader(raw))
	scanner.Buffer(make([]byte, 64*1024), 16*1024*1024)

	var rows []kustoReadyLogEntry
	result := importResult{}
	for scanner.Scan() {
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 {
			continue
		}

		var entry receivedLogEntry
		if err := json.Unmarshal(line, &entry); err != nil {
			result.DroppedRequests++
			log.Printf("dropping malformed imported line: %v", err)
			continue
		}

		req, err := decodeRequestPayload(entry.ContentType, entry.PayloadBase64)
		if err != nil {
			result.DroppedRequests++
			log.Printf("dropping request for cluster=%q: %v", entry.ClusterResourceID, err)
			continue
		}

		result.ImportedRequests++
		requestRows := flattenRequest(entry, req)
		result.TranslatedLogs += len(requestRows)
		rows = append(rows, requestRows...)
	}
	if err := scanner.Err(); err != nil {
		return nil, importResult{}, fmt.Errorf("scanning imported payload: %w", err)
	}

	return rows, result, nil
}

func decodeRequestPayload(contentType, payloadBase64 string) (*collectorlogsv1.ExportLogsServiceRequest, error) {
	rawPayload, err := base64.StdEncoding.DecodeString(payloadBase64)
	if err != nil {
		return nil, fmt.Errorf("decoding payloadBase64: %w", err)
	}

	req := &collectorlogsv1.ExportLogsServiceRequest{}
	contentType = strings.ToLower(contentType)
	if strings.Contains(contentType, "json") {
		if err := protojson.Unmarshal(rawPayload, req); err != nil {
			return nil, fmt.Errorf("decoding json payload: %w", err)
		}
		return req, nil
	}

	if err := proto.Unmarshal(rawPayload, req); err == nil {
		return req, nil
	}
	if err := protojson.Unmarshal(rawPayload, req); err == nil {
		return req, nil
	}

	return nil, fmt.Errorf("unsupported OTLP payload for content type %q", contentType)
}

func flattenRequest(imported receivedLogEntry, req *collectorlogsv1.ExportLogsServiceRequest) []kustoReadyLogEntry {
	var out []kustoReadyLogEntry

	for _, resourceLogs := range req.ResourceLogs {
		resourceAttrs := keyValuesToMap(resourceLogs.Resource.GetAttributes())

		for _, scopeLogs := range resourceLogs.ScopeLogs {
			scopeAttrs := keyValuesToMap(scopeLogs.Scope.GetAttributes())
			scopeName := scopeLogs.Scope.GetName()
			scopeVersion := scopeLogs.Scope.GetVersion()

			for _, record := range scopeLogs.LogRecords {
				out = append(out, kustoReadyLogEntry{
					ReceivedAt:        imported.ReceivedAt,
					ClusterResourceID: imported.ClusterResourceID,
					ContentType:       imported.ContentType,
					ResourceAttrs:     resourceAttrs,
					ScopeName:         scopeName,
					ScopeVersion:      scopeVersion,
					ScopeAttrs:        scopeAttrs,
					LogAttrs:          keyValuesToMap(record.Attributes),
					Body:              anyValueToInterface(record.Body),
					SeverityText:      record.SeverityText,
					SeverityNumber:    record.SeverityNumber,
					Timestamp:         unixNanoToString(record.TimeUnixNano),
					ObservedTimestamp: unixNanoToString(record.ObservedTimeUnixNano),
					TraceID:           hex.EncodeToString(record.TraceId),
					SpanID:            hex.EncodeToString(record.SpanId),
				})
			}
		}
	}

	return out
}

func anyValueToInterface(value *commonv1.AnyValue) any {
	if value == nil {
		return nil
	}

	switch typed := value.Value.(type) {
	case *commonv1.AnyValue_StringValue:
		return typed.StringValue
	case *commonv1.AnyValue_BoolValue:
		return typed.BoolValue
	case *commonv1.AnyValue_IntValue:
		return typed.IntValue
	case *commonv1.AnyValue_DoubleValue:
		return typed.DoubleValue
	case *commonv1.AnyValue_ArrayValue:
		result := make([]any, 0, len(typed.ArrayValue.Values))
		for _, element := range typed.ArrayValue.Values {
			result = append(result, anyValueToInterface(element))
		}
		return result
	case *commonv1.AnyValue_KvlistValue:
		return keyValuesToMap(typed.KvlistValue.Values)
	case *commonv1.AnyValue_BytesValue:
		return base64.StdEncoding.EncodeToString(typed.BytesValue)
	default:
		return nil
	}
}

func keyValuesToMap(values []*commonv1.KeyValue) map[string]any {
	if len(values) == 0 {
		return nil
	}

	out := make(map[string]any, len(values))
	for _, kv := range values {
		if kv == nil {
			continue
		}
		out[kv.Key] = anyValueToInterface(kv.Value)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func unixNanoToString(v uint64) string {
	if v == 0 {
		return ""
	}
	return time.Unix(0, int64(v)).UTC().Format(time.RFC3339Nano)
}

func persistTranslatedRows(rows []kustoReadyLogEntry) error {
	if len(rows) == 0 {
		return nil
	}

	outputMu.Lock()
	defer outputMu.Unlock()

	file, err := os.OpenFile(outputPath, os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("opening output file %q: %w", outputPath, err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	for _, row := range rows {
		if err := encoder.Encode(row); err != nil {
			return fmt.Errorf("writing output file %q: %w", outputPath, err)
		}
	}

	return nil
}

func forwardToKusto(rows []kustoReadyLogEntry) error {
	var payload bytes.Buffer
	encoder := json.NewEncoder(&payload)
	for _, row := range rows {
		if err := encoder.Encode(row); err != nil {
			return fmt.Errorf("encoding forward payload: %w", err)
		}
	}

	req, err := http.NewRequest(http.MethodPost, kustoIngestURL, bytes.NewReader(payload.Bytes()))
	if err != nil {
		return fmt.Errorf("creating forward request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-ndjson")
	if kustoAuthHeader != "" {
		req.Header.Set("Authorization", kustoAuthHeader)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("sending forward request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("kusto ingest responded %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return nil
}

func initializeStoragePath(path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("path is empty")
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("creating directory %q: %w", dir, err)
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return "", fmt.Errorf("opening file %q: %w", path, err)
	}
	if err := file.Close(); err != nil {
		return "", fmt.Errorf("closing file %q: %w", path, err)
	}
	return path, nil
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
