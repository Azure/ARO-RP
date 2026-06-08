package gatewayauth

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configoptional"
	"go.opentelemetry.io/collector/config/configtls"
	"go.opentelemetry.io/collector/confmap/xconfmap"
)

type Config struct {
	// Configures the protocol to use TLS.
	// The default value is nil, which will cause the protocol to not use TLS.
	TLS configoptional.Optional[configtls.ServerConfig] `mapstructure:"tls,omitempty"`

	ChangefeedBatchSize configoptional.Optional[int] `mapstructure:"changefeedBatchSize,omitempty"`
	ChangefeedInterval  configoptional.Optional[int] `mapstructure:"changefeedInterval,omitempty"`
}

func (sc *Config) Validate() error {
	return nil
}

var _ xconfmap.Validator = (*Config)(nil)

func createAuthConfig() component.Config {
	return &Config{}
}
