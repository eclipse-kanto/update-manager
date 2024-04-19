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
	"reflect"
	"testing"

	"github.com/eclipse-kanto/update-manager/api"

	"github.com/eclipse-kanto/update-manager/logger"
	"github.com/eclipse-kanto/update-manager/mqtt"

	"github.com/stretchr/testify/assert"
)

func TestNewDefaultConfig(t *testing.T) {

	agentsDefault := map[string]*api.UpdateManagerConfig{
		"containers": {
			Name:           "containers",
			RebootRequired: false,
			ReadTimeout:    "1m",
		},
	}

	defaultConfigValues := Config{
		BaseConfig: &BaseConfig{
			Log: &logger.LogConfig{
				LogFile:       "",
				LogLevel:      "INFO",
				LogFileSize:   2,
				LogFileCount:  5,
				LogFileMaxAge: 28,
			},
			MQTT: &mqtt.ConnectionConfig{
				Broker:             "tcp://localhost:1883",
				KeepAlive:          "20s",
				DisconnectTimeout:  "250ms",
				Username:           "",
				Password:           "",
				ConnectTimeout:     "30s",
				AcknowledgeTimeout: "15s",
				SubscribeTimeout:   "15s",
				UnsubscribeTimeout: "5s",
			},
			Domain:        "device",
			ThingsEnabled: true,
		},
		Agents:                 agentsDefault,
		RebootEnabled:          true,
		RebootAfter:            "30s",
		ReportFeedbackInterval: "1m",
		CurrentStateDelay:      "30s",
		PhaseTimeout:           "10m",
	}

	cfg := newDefaultConfig()
	cfg.Agents = newDefaultAgentsConfig()
	assert.True(t, reflect.DeepEqual(*cfg, defaultConfigValues))
}

func TestLoadConfigFromFile(t *testing.T) {
	cfg := newDefaultConfig()
	t.Run("test_not_existing", func(t *testing.T) {
		err := LoadConfigFromFile("../config/testdata/not-existing.json", cfg)
		assert.Error(t, err, "error expected for non existing file")
	})
	t.Run("test_is_dir", func(t *testing.T) {
		err := LoadConfigFromFile("../config/testdata/", cfg)
		assert.Error(t, err, "provided configuration path %s is a directory", "../config/testdata/")
	})
	t.Run("test_file_empty", func(t *testing.T) {
		err := LoadConfigFromFile("../config/testdata/empty.json", cfg)
		assert.Error(t, err, "error expected for empty.json")
	})
	t.Run("test_json_invalid", func(t *testing.T) {
		err := LoadConfigFromFile("../config/testdata/invalid.json", cfg)
		assert.Error(t, err, "unexpected end of JSON input")
	})
	t.Run("test_json_valid", func(t *testing.T) {
		err := LoadConfigFromFile("../config/testdata/config.json", cfg)
		assert.NoError(t, err)

		expectedAgentValues := map[string]*api.UpdateManagerConfig{
			"self-update": {
				RebootRequired: false,
				ReadTimeout:    "20s",
			},
			"containers": {
				RebootRequired: true,
				ReadTimeout:    "30s",
			},
			"test-domain": {
				RebootRequired: true,
				ReadTimeout:    "50s",
			},
		}

		expectedConfigValues := Config{
			BaseConfig: &BaseConfig{
				Log: &logger.LogConfig{
					LogFile:       "log/update-manager.log",
					LogLevel:      "ERROR",
					LogFileSize:   3,
					LogFileCount:  6,
					LogFileMaxAge: 29,
				},
				MQTT: &mqtt.ConnectionConfig{
					Broker:             "www",
					KeepAlive:          "500ms",
					DisconnectTimeout:  "500ms",
					Username:           "username",
					Password:           "pass",
					ConnectTimeout:     "500ms",
					AcknowledgeTimeout: "500ms",
					SubscribeTimeout:   "500ms",
					UnsubscribeTimeout: "500ms",
				},
				Domain:        "mydomain",
				ThingsEnabled: false,
			},
			Agents:                 expectedAgentValues,
			RebootEnabled:          false,
			RebootAfter:            "1m",
			ReportFeedbackInterval: "2m",
			CurrentStateDelay:      "1m",
			PhaseTimeout:           "2m",
			OwnerConsentPhases:     []string{"download"},
		}
		assert.True(t, reflect.DeepEqual(*cfg, expectedConfigValues))
	})
	t.Run("test_json_valid_but_cfg_nil", func(t *testing.T) {
		err := LoadConfigFromFile("../config/testdata/config.json", nil)
		assert.Error(t, err)
	})

}
