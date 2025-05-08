package log

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"reflect"
	"testing"

	"github.com/sirupsen/logrus"
)

func TestLogrWrapperWithKeysAndValues(t *testing.T) {
	for _, tt := range []struct {
		name     string
		kv       []interface{}
		expected logrus.Fields
	}{
		{
			kv: []interface{}{"", "value1", "key2", "value2"},
			expected: logrus.Fields{
				"":     "value1",
				"key2": "value2",
			},
		},
		{
			kv: []interface{}{"key1", "value1", "key2"},
			expected: logrus.Fields{
				"key1": "value1",
				"key2": nil,
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			tt.expected["existingkey"] = "existingvalue"

			lw := &logrWrapper{
				entry: &logrus.Entry{
					Data: logrus.Fields{"existingkey": "existingvalue"},
				},
			}

			log := lw.withKeysAndValues(tt.kv)
			if !reflect.DeepEqual(log.Data, tt.expected) {
				t.Error(log.Data)
			}
		})
	}
}
