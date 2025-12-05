package metrics

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/metrics"
	"github.com/Azure/ARO-RP/pkg/util/recover"
)

// ResourceUsageCollector collects CPU, memory, and I/O metrics for the MIMO actuator process
// Simplified implementation: uses os.Getpid() since collector runs in same process as MIMO actuator
type ResourceUsageCollector struct {
	log          *logrus.Entry
	m            metrics.Emitter
	pid          int
	interval     time.Duration
	lastCPUUsage time.Time
	lastCPUTime  uint64
}

// ResourceUsage contains the collected resource metrics
type ResourceUsage struct {
	CPUPercent    float64
	MemoryBytes   int64
	MemoryPercent float64
	IOReadBytes   int64
	IOWriteBytes  int64
}

// NewResourceUsageCollector creates a new resource usage collector for MIMO
// Simplified: uses os.Getpid() since collector runs in same process as MIMO actuator
func NewResourceUsageCollector(log *logrus.Entry, m metrics.Emitter) *ResourceUsageCollector {
	return &ResourceUsageCollector{
		log:      log,
		m:        m,
		pid:      os.Getpid(), // Simple: use current process PID
		interval: time.Minute, // Collect metrics every minute
	}
}

// Run starts the resource usage collector goroutine
func (r *ResourceUsageCollector) Run(ctx context.Context, stop <-chan struct{}) {
	defer recover.Panic(r.log)

	r.log.Infof("starting resource usage collector for PID: %d", r.pid)

	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			usage, err := r.collectResourceUsage()
			if err != nil {
				r.log.Errorf("failed to collect resource usage: %v", err)
				continue
			}
			r.emitMetrics(usage)
		case <-stop:
			r.log.Info("resource usage collector stopped")
			return
		case <-ctx.Done():
			r.log.Info("resource usage collector stopped")
			return
		}
	}
}

// collectResourceUsage collects CPU, memory, and I/O metrics from /proc
func (r *ResourceUsageCollector) collectResourceUsage() (*ResourceUsage, error) {
	usage := &ResourceUsage{}

	// Read /proc/[pid]/stat for CPU and basic info
	statPath := fmt.Sprintf("/proc/%d/stat", r.pid)
	statData, err := os.ReadFile(statPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read stat: %w", err)
	}

	// Parse CPU time from /proc/[pid]/stat
	// Format: pid comm state ppid pgrp session tty_nr tty_pgrp flags minflt cminflt majflt cmajflt utime stime cutime cstime ...
	fields := strings.Fields(string(statData))
	if len(fields) < 15 {
		return nil, fmt.Errorf("invalid stat format")
	}

	utime, _ := strconv.ParseUint(fields[13], 10, 64) // utime (user time)
	stime, _ := strconv.ParseUint(fields[14], 10, 64) // stime (system time)
	totalCPUTime := utime + stime

	// Calculate CPU percentage
	now := time.Now()
	if !r.lastCPUUsage.IsZero() {
		timeDelta := now.Sub(r.lastCPUUsage).Seconds()
		if timeDelta > 0 && r.lastCPUTime > 0 {
			cpuDelta := float64(totalCPUTime - r.lastCPUTime)
			// CPU time is in jiffies (clock ticks), typically 100 per second
			clockTicks := float64(100)
			usage.CPUPercent = (cpuDelta / clockTicks / timeDelta) * 100.0
			if usage.CPUPercent < 0 {
				usage.CPUPercent = 0
			}
			// Cap at 100% per core
			if usage.CPUPercent > 100.0 {
				usage.CPUPercent = 100.0
			}
		}
	}
	r.lastCPUUsage = now
	r.lastCPUTime = totalCPUTime

	// Read /proc/[pid]/status for memory info
	statusPath := fmt.Sprintf("/proc/%d/status", r.pid)
	statusData, err := os.ReadFile(statusPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read status: %w", err)
	}

	// Parse memory from status file
	lines := strings.Split(string(statusData), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "VmRSS:") {
			// VmRSS is resident set size (physical memory)
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				value, _ := strconv.ParseInt(fields[1], 10, 64)
				usage.MemoryBytes = value * 1024 // Convert kB to bytes
			}
		}
	}

	// Calculate memory percentage (need total system memory)
	totalMem, err := r.getTotalMemory()
	if err == nil && totalMem > 0 {
		usage.MemoryPercent = (float64(usage.MemoryBytes) / float64(totalMem)) * 100.0
	}

	// Read /proc/[pid]/io for I/O statistics
	ioPath := fmt.Sprintf("/proc/%d/io", r.pid)
	ioData, err := os.ReadFile(ioPath)
	if err == nil {
		// Parse I/O stats
		lines := strings.Split(string(ioData), "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "read_bytes:") {
				fields := strings.Fields(line)
				if len(fields) >= 2 {
					usage.IOReadBytes, _ = strconv.ParseInt(fields[1], 10, 64)
				}
			}
			if strings.HasPrefix(line, "write_bytes:") {
				fields := strings.Fields(line)
				if len(fields) >= 2 {
					usage.IOWriteBytes, _ = strconv.ParseInt(fields[1], 10, 64)
				}
			}
		}
	}

	return usage, nil
}

// getTotalMemory gets total system memory in bytes from /proc/meminfo
func (r *ResourceUsageCollector) getTotalMemory() (int64, error) {
	meminfoPath := "/proc/meminfo"
	data, err := os.ReadFile(meminfoPath)
	if err != nil {
		return 0, err
	}

	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "MemTotal:") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				value, err := strconv.ParseInt(fields[1], 10, 64)
				if err != nil {
					return 0, err
				}
				return value * 1024, nil // Convert kB to bytes
			}
		}
	}

	return 0, fmt.Errorf("MemTotal not found in /proc/meminfo")
}

// emitMetrics emits resource usage metrics via statsd
func (r *ResourceUsageCollector) emitMetrics(usage *ResourceUsage) {
	dimensions := map[string]string{
		"service": "mimo-actuator",
		"pid":     strconv.Itoa(r.pid),
	}

	// Emit CPU percentage
	r.m.EmitFloat("mimo.resource.cpu.percent", usage.CPUPercent, dimensions)

	// Emit memory usage in bytes
	r.m.EmitGauge("mimo.resource.memory.bytes", usage.MemoryBytes, dimensions)

	// Emit memory percentage
	r.m.EmitFloat("mimo.resource.memory.percent", usage.MemoryPercent, dimensions)

	// Emit I/O read bytes
	r.m.EmitGauge("mimo.resource.io.read_bytes", usage.IOReadBytes, dimensions)

	// Emit I/O write bytes
	r.m.EmitGauge("mimo.resource.io.write_bytes", usage.IOWriteBytes, dimensions)

	r.log.Debugf("emitted MIMO resource metrics: CPU=%.2f%%, Memory=%d bytes (%.2f%%), IO Read=%d bytes, IO Write=%d bytes",
		usage.CPUPercent, usage.MemoryBytes, usage.MemoryPercent, usage.IOReadBytes, usage.IOWriteBytes)
}
