// Copyright (c) 2023 Contributors to the Eclipse Foundation
//
// See the NOTICE file(s) distributed with this work for additional
// information regarding copyright ownership.
//
// This program and the accompanying materials are made available under the
// terms of the Eclipse Public License 2.0 which is available at
// https://www.eclipse.org/legal/epl-2.0, or the Apache License, Version 2.0
// which is available at https://www.apache.org/licenses/LICENSE-2.0.
//
// SPDX-License-Identifier: EPL-2.0 OR Apache-2.0

package config

import (
	"encoding/json"
	"os"

	"github.com/eclipse-kanto/update-manager/logger"
	"github.com/eclipse-kanto/update-manager/mqtt"
)

// BaseConfig represents the common, reusable configuration that holds logger options and MQTT connection parameters.
type BaseConfig struct {
	Log           *logger.LogConfig      `json:"log,omitempty"`
	MQTT          *mqtt.ConnectionConfig `json:"connection,omitempty"`
	Domain        string                 `json:"domain,omitempty"`
	ThingsEnabled bool                   `json:"thingsEnabled,omitempty"`
}

// DefaultDomainConfig creates a new configuration filled with default values for all config properties and domain name set to the given parameter.
func DefaultDomainConfig(domain string) *BaseConfig {
	return &BaseConfig{
		Log: &logger.LogConfig{
			LogFile:       logFileDefault,
			LogLevel:      logLevelDefault,
			LogFileSize:   logFileSizeDefault,
			LogFileCount:  logFileCountDefault,
			LogFileMaxAge: logFileMaxAgeDefault,
		},
		MQTT:          mqtt.NewDefaultConfig(),
		Domain:        domain,
		ThingsEnabled: true,
	}
}

// LoadConfigFromFile reads the file contents and unmarshal them into the given config structure.
func LoadConfigFromFile(filePath string, config interface{}) error {
	file, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}
	if err = json.Unmarshal(file, config); err != nil {
		return err
	}
	return nil
}
