// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

package kubeserializer

import (
	"bytes"

	kruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	kjson "k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/client-go/kubernetes/scheme"
)

type Serializer interface {
	ToYAML(kruntime.Object) ([]byte, error)
}

type serializer struct {
	codec kruntime.Codec
}

func NewYamlSerializer() *serializer {
	kserializer := kjson.NewSerializerWithOptions(
		kjson.DefaultMetaFactory, scheme.Scheme, scheme.Scheme,
		kjson.SerializerOptions{Yaml: true},
	)
	yaml := scheme.Codecs.CodecForVersions(kserializer, nil, schema.GroupVersions(scheme.Scheme.PrioritizedVersionsAllGroups()), nil)

	return &serializer{codec: yaml}
}

func (s *serializer) ToYAML(obj kruntime.Object) ([]byte, error) {
	o := new(bytes.Buffer)
	err := s.codec.Encode(obj, o)
	if err != nil {
		return nil, err
	}

	return o.Bytes(), nil
}
