package metrics

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"os"
	"strconv"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/sirupsen/logrus"
	"go.uber.org/mock/gomock"

	mock_metrics "github.com/Azure/ARO-RP/pkg/util/mocks/metrics"
)

var _ = Describe("ResourceUsageCollector", func() {
	var controller *gomock.Controller
	var mockEmitter *mock_metrics.MockEmitter
	var log *logrus.Entry
	var collector *ResourceUsageCollector

	BeforeEach(func() {
		controller = gomock.NewController(GinkgoT())
		mockEmitter = mock_metrics.NewMockEmitter(controller)

		log = logrus.NewEntry(&logrus.Logger{
			Out:       GinkgoWriter,
			Formatter: new(logrus.TextFormatter),
			Hooks:     make(logrus.LevelHooks),
			Level:     logrus.DebugLevel,
		})
	})

	AfterEach(func() {
		controller.Finish()
	})

	Describe("NewResourceUsageCollector", func() {
		It("creates a collector with the current process PID", func() {
			collector = NewResourceUsageCollector(log, mockEmitter)

			Expect(collector).NotTo(BeNil())
			Expect(collector.pid).To(Equal(os.Getpid()))
			Expect(collector.interval).To(Equal(time.Minute))
			Expect(collector.log).To(Equal(log))
			Expect(collector.m).To(Equal(mockEmitter))
		})
	})

	Describe("emitMetrics", func() {
		BeforeEach(func() {
			collector = NewResourceUsageCollector(log, mockEmitter)
		})

		It("emits all resource metrics with correct dimensions", func() {
			usage := &ResourceUsage{
				CPUPercent:    5.5,
				MemoryBytes:   1024 * 1024 * 100, // 100 MB
				MemoryPercent: 2.5,
				IOReadBytes:   1024 * 1024, // 1 MB
				IOWriteBytes:  1024 * 512,  // 512 KB
			}

			expectedDimensions := map[string]string{
				"service": "mimo-actuator",
				"pid":     strconv.Itoa(collector.pid),
			}

			// Expect all metrics to be emitted
			mockEmitter.EXPECT().EmitFloat("mimo.resource.cpu.percent", usage.CPUPercent, expectedDimensions)
			mockEmitter.EXPECT().EmitGauge("mimo.resource.memory.bytes", usage.MemoryBytes, expectedDimensions)
			mockEmitter.EXPECT().EmitFloat("mimo.resource.memory.percent", usage.MemoryPercent, expectedDimensions)
			mockEmitter.EXPECT().EmitGauge("mimo.resource.io.read_bytes", usage.IOReadBytes, expectedDimensions)
			mockEmitter.EXPECT().EmitGauge("mimo.resource.io.write_bytes", usage.IOWriteBytes, expectedDimensions)

			collector.emitMetrics(usage)
		})

		It("emits zero values correctly", func() {
			usage := &ResourceUsage{
				CPUPercent:    0,
				MemoryBytes:   0,
				MemoryPercent: 0,
				IOReadBytes:   0,
				IOWriteBytes:  0,
			}

			expectedDimensions := map[string]string{
				"service": "mimo-actuator",
				"pid":     strconv.Itoa(collector.pid),
			}

			mockEmitter.EXPECT().EmitFloat("mimo.resource.cpu.percent", float64(0), expectedDimensions)
			mockEmitter.EXPECT().EmitGauge("mimo.resource.memory.bytes", int64(0), expectedDimensions)
			mockEmitter.EXPECT().EmitFloat("mimo.resource.memory.percent", float64(0), expectedDimensions)
			mockEmitter.EXPECT().EmitGauge("mimo.resource.io.read_bytes", int64(0), expectedDimensions)
			mockEmitter.EXPECT().EmitGauge("mimo.resource.io.write_bytes", int64(0), expectedDimensions)

			collector.emitMetrics(usage)
		})
	})

	Describe("Run", func() {
		BeforeEach(func() {
			collector = NewResourceUsageCollector(log, mockEmitter)
			collector.interval = 100 * time.Millisecond
		})

		It("stops when stop channel is closed", func() {
			stop := make(chan struct{})
			done := make(chan struct{})

			go func() {
				collector.Run(context.Background(), stop)
				close(done)
			}()

			close(stop)
			Eventually(done, time.Second).Should(BeClosed())
		})

		It("stops when context is cancelled", func() {
			ctx, cancel := context.WithCancel(context.Background())
			done := make(chan struct{})

			go func() {
				collector.Run(ctx, make(chan struct{}))
				close(done)
			}()

			cancel()
			Eventually(done, time.Second).Should(BeClosed())
		})
	})

	Describe("collectResourceUsage", func() {
		BeforeEach(func() {
			collector = NewResourceUsageCollector(log, mockEmitter)
		})

		// This test runs on the actual /proc filesystem, so it will only work on Linux
		It("collects resource usage for the current process", func() {
			// Skip if not on Linux (no /proc filesystem)
			if _, err := os.Stat("/proc/self/stat"); os.IsNotExist(err) {
				Skip("Skipping test - /proc filesystem not available (not Linux)")
			}

			usage, err := collector.collectResourceUsage()

			Expect(err).ToNot(HaveOccurred())
			Expect(usage).NotTo(BeNil())

			// Memory should be > 0 for a running process
			Expect(usage.MemoryBytes).To(BeNumerically(">", 0))

			// CPU percent on first call should be 0 (no previous baseline)
			Expect(usage.CPUPercent).To(BeNumerically(">=", 0))
			Expect(usage.CPUPercent).To(BeNumerically("<=", 100))

			// Memory percent should be reasonable (between 0 and 100)
			Expect(usage.MemoryPercent).To(BeNumerically(">=", 0))
			Expect(usage.MemoryPercent).To(BeNumerically("<=", 100))

			// I/O bytes should be >= 0
			Expect(usage.IOReadBytes).To(BeNumerically(">=", 0))
			Expect(usage.IOWriteBytes).To(BeNumerically(">=", 0))
		})

		It("calculates CPU percentage on subsequent calls", func() {
			// Skip if not on Linux (no /proc filesystem)
			if _, err := os.Stat("/proc/self/stat"); os.IsNotExist(err) {
				Skip("Skipping test - /proc filesystem not available (not Linux)")
			}

			// First call establishes baseline
			_, err := collector.collectResourceUsage()
			Expect(err).ToNot(HaveOccurred())

			// Second call should be able to calculate CPU percentage
			usage, err := collector.collectResourceUsage()
			Expect(err).ToNot(HaveOccurred())
			Expect(usage).NotTo(BeNil())

			// CPU percent should be valid (between 0 and 100)
			Expect(usage.CPUPercent).To(BeNumerically(">=", 0))
			Expect(usage.CPUPercent).To(BeNumerically("<=", 100))
		})
	})

	Describe("getTotalMemory", func() {
		BeforeEach(func() {
			collector = NewResourceUsageCollector(log, mockEmitter)
		})

		It("returns total system memory", func() {
			// Skip if not on Linux (no /proc filesystem)
			if _, err := os.Stat("/proc/meminfo"); os.IsNotExist(err) {
				Skip("Skipping test - /proc filesystem not available (not Linux)")
			}

			totalMem, err := collector.getTotalMemory()

			Expect(err).ToNot(HaveOccurred())
			// Total memory should be at least 1 GB on most systems
			Expect(totalMem).To(BeNumerically(">", 1024*1024*1024))
		})
	})
})

func TestResourceUsageCollector(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "ResourceUsageCollector Suite")
}
