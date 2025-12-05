package metrics

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
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

		collector = NewResourceUsageCollector(log, mockEmitter)
	})

	AfterEach(func() {
		controller.Finish()
	})

	It("emits all resource metrics with correct dimensions", func() {
		usage := &ResourceUsage{
			CPUPercent:    5.5,
			MemoryBytes:   1024 * 1024 * 100,
			MemoryPercent: 2.5,
			IOReadBytes:   1024 * 1024,
			IOWriteBytes:  1024 * 512,
		}

		expectedDimensions := map[string]string{
			"service": "mimo-actuator",
			"pid":     strconv.Itoa(collector.pid),
		}

		mockEmitter.EXPECT().EmitFloat("mimo.resource.cpu.percent", usage.CPUPercent, expectedDimensions)
		mockEmitter.EXPECT().EmitGauge("mimo.resource.memory.bytes", usage.MemoryBytes, expectedDimensions)
		mockEmitter.EXPECT().EmitFloat("mimo.resource.memory.percent", usage.MemoryPercent, expectedDimensions)
		mockEmitter.EXPECT().EmitGauge("mimo.resource.io.read_bytes", usage.IOReadBytes, expectedDimensions)
		mockEmitter.EXPECT().EmitGauge("mimo.resource.io.write_bytes", usage.IOWriteBytes, expectedDimensions)

		collector.emitMetrics(usage)
	})

	It("stops when context is cancelled", func() {
		collector.interval = 100 * time.Millisecond
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

func TestResourceUsageCollector(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "ResourceUsageCollector Suite")
}
