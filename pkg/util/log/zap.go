package log

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"github.com/sirupsen/logrus"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type zapWrapper struct {
	zapLogger *zap.Logger
}

var _levels = map[logrus.Level]zapcore.Level{
	logrus.TraceLevel: zap.DebugLevel,
	logrus.DebugLevel: zap.DebugLevel,
	logrus.InfoLevel:  zap.InfoLevel,
	logrus.WarnLevel:  zap.WarnLevel,
	logrus.ErrorLevel: zap.ErrorLevel,
	logrus.PanicLevel: zap.PanicLevel,
	logrus.FatalLevel: zap.FatalLevel,
}

var _ logrus.Hook = &zapWrapper{}

func (f *zapWrapper) Levels() []logrus.Level {
	return logrus.AllLevels
}

func (f *zapWrapper) Fire(entry *logrus.Entry) error {
	fields := []zap.Field{}

	for k, v := range entry.Data {
		fields = append(fields, zap.Any(k, v))
	}

	f.zapLogger.Log(_levels[entry.Level], entry.Message, fields...)

	// zap's logger doesn't error
	return nil
}

// Return a *logrus.Entry which forwards logs to zap. Don't set the level at the logrus level
func NewLogrusToZapLogger(tgt *zap.Logger) *logrus.Entry {
	logger := logrus.New()
	logger.AddHook(&zapWrapper{
		zapLogger: tgt.WithOptions(
			// TODO: logrus info/infof are different depths so this will
			// sometimes emit the outer function, we can't pass the caller to
			// zap directly
			zap.AddCallerSkip(7),
		),
	})

	return logrus.NewEntry(logger)
}
