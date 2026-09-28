package main

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "otele2e: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	config := flag.String("config", "test/e2e/static_resources/otel-collector.yaml", "path to collector config")
	logFile := flag.String("logfile", "_out/otel-e2e/logs.json", "path for received OTLP logs (OTEL_E2E_LOG_FILE)")
	binary := flag.String("binary", "telemetry/collector/build/collector/telemetrycollector", "path to telemetrycollector binary")
	healthURL := flag.String("health", "http://127.0.0.1:13133/", "collector health check URL")
	readyTimeout := flag.Duration("ready-timeout", 30*time.Second, "how long to wait for health check")
	flag.Parse()

	if err := ensureBinary(*binary); err != nil {
		return err
	}
	if _, err := os.Stat(*config); err != nil {
		return fmt.Errorf("config %q not found: %w", *config, err)
	}
	if err := os.MkdirAll(filepath.Dir(*logFile), 0o755); err != nil {
		return err
	}

	absLogFile, err := filepath.Abs(*logFile)
	if err != nil {
		return err
	}
	absConfig, err := filepath.Abs(*config)
	if err != nil {
		return err
	}

	cmd := exec.Command(*binary, "--config="+absConfig)
	cmd.Env = append(os.Environ(), "OTEL_E2E_LOG_FILE="+absLogFile)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return err
	}

	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	if err := waitHealthy(*healthURL, *readyTimeout, done); err != nil {
		_ = cmd.Process.Kill()
		<-done
		return err
	}
	fmt.Println("otele2e: ready")

	select {
	case <-sigCh:
		if err := cmd.Process.Signal(syscall.SIGTERM); err != nil {
			_ = cmd.Process.Kill()
		}
		<-done
		return nil
	case err := <-done:
		if err != nil {
			return fmt.Errorf("collector exited: %w", err)
		}
		return fmt.Errorf("collector exited unexpectedly")
	}
}

func ensureBinary(path string) error {
	if _, err := os.Stat(path); err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("stat binary %q: %w", path, err)
	}

	fmt.Fprintf(os.Stderr, "otele2e: %s not found; running make build-telemetrycollector\n", path)
	build := exec.Command("make", "build-telemetrycollector")
	build.Stdout = os.Stdout
	build.Stderr = os.Stderr
	if err := build.Run(); err != nil {
		return fmt.Errorf("make build-telemetrycollector: %w", err)
	}
	if _, err := os.Stat(path); err != nil {
		return fmt.Errorf("binary %q still missing after build: %w", path, err)
	}
	return nil
}

func waitHealthy(url string, timeout time.Duration, done <-chan error) error {
	deadline := time.Now().Add(timeout)
	client := &http.Client{Timeout: 2 * time.Second}
	for time.Now().Before(deadline) {
		select {
		case err := <-done:
			if err != nil {
				return fmt.Errorf("collector exited before becoming healthy: %w", err)
			}
			return fmt.Errorf("collector exited before becoming healthy")
		default:
		}
		resp, err := client.Get(url)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode >= 200 && resp.StatusCode < 300 {
				return nil
			}
		}
		time.Sleep(200 * time.Millisecond)
	}
	return fmt.Errorf("collector not healthy at %s within %s", url, timeout)
}
