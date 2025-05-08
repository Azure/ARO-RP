package log

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"fmt"
	"strings"

	"github.com/coreos/go-systemd/v22/journal"
	"github.com/sirupsen/logrus"
)

type journaldHook struct{}

var _ logrus.Hook = (*journaldHook)(nil)

func (h *journaldHook) Levels() []logrus.Level {
	return logrus.AllLevels
}

func (h *journaldHook) Fire(e *logrus.Entry) error {
	var priority journal.Priority

	switch e.Level {
	case logrus.PanicLevel:
		priority = journal.PriEmerg
	case logrus.FatalLevel:
		priority = journal.PriCrit
	case logrus.ErrorLevel:
		priority = journal.PriErr
	case logrus.WarnLevel:
		priority = journal.PriWarning
	case logrus.InfoLevel:
		priority = journal.PriInfo
	default:
		priority = journal.PriDebug
	}

	vars := make(map[string]string, len(e.Data))
	for k, v := range e.Data {
		vars[key(k)] = fmt.Sprint(v)
	}

	if e.Caller != nil {
		vars["FUNCTION"], vars["FILE"] = relativeFilePathPrettier(e.Caller)
	}

	return journal.Send(e.Message, priority, vars)
}

func key(k string) string {
	return strings.TrimPrefix(strings.Map(func(r rune) rune {
		switch {
		case r >= '0' && r <= '9',
			r >= 'A' && r <= 'Z':
			return r
		case r >= 'a' && r <= 'z':
			return r - 32
		default:
			return '_'
		}
	}, k), "_")
}
