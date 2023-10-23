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
	"flag"
	"fmt"
	"os"
	"reflect"
	"testing"

	"github.com/eclipse-kanto/update-manager/api"

	"github.com/stretchr/testify/assert"
)

func TestSetupFlags(t *testing.T) {
	cfg := newDefaultConfig()
	flagSet := flag.CommandLine
	SetupAllUpdateManagerFlags(flagSet, cfg)

	tests := map[string]struct {
		flag         string
		expectedType string
	}{
		"test_flags_log_level": {
			flag:         "log-level",
			expectedType: reflect.String.String(),
		},
		"test_flags_log_file": {
			flag:         "log-file",
			expectedType: reflect.String.String(),
		},
		"test_flags_log_file_size": {
			flag:         "log-file-size",
			expectedType: reflect.Int.String(),
		},
		"test_flags_log_file_count": {
			flag:         "log-file-count",
			expectedType: reflect.Int.String(),
		},
		"test_flags_log_file_max_age": {
			flag:         "log-file-max-age",
			expectedType: reflect.Int.String(),
		},
		"test_flags_mqtt_conn_broker": {
			flag:         "mqtt-conn-broker",
			expectedType: reflect.String.String(),
		},
		"test_flags_mqtt_conn_keep_alive": {
			flag:         "mqtt-conn-keep-alive",
			expectedType: reflect.String.String(),
		},
		"test_flags_mqtt_conn_disconnect_timeout": {
			flag:         "mqtt-conn-disconnect-timeout",
			expectedType: reflect.String.String(),
		},
		"test_flags_mqtt_conn_username": {
			flag:         "mqtt-conn-username",
			expectedType: reflect.String.String(),
		},
		"test_flags_mqtt-conn-password": {
			flag:         "mqtt-conn-password",
			expectedType: reflect.String.String(),
		},
		"test_flags_mqtt-conn-connect-timeout": {
			flag:         "mqtt-conn-connect-timeout",
			expectedType: reflect.String.String(),
		},
		"test_flags_mqtt-conn-ack-timeout": {
			flag:         "mqtt-conn-ack-timeout",
			expectedType: reflect.String.String(),
		},
		"test_flags_mqtt-conn-sub-timeout": {
			flag:         "mqtt-conn-sub-timeout",
			expectedType: reflect.String.String(),
		},
		"test_flags_mqtt-conn-unsub-timeout": {
			flag:         "mqtt-conn-unsub-timeout",
			expectedType: reflect.String.String(),
		},
		"test_flags_mqtt-conn-root-ca": {
			flag:         "mqtt-conn-root-ca",
			expectedType: reflect.String.String(),
		},
		"test_flags_mqtt-conn-client-cert": {
			flag:         "mqtt-conn-client-cert",
			expectedType: reflect.String.String(),
		},
		"test_flags_mqtt-conn-client-key": {
			flag:         "mqtt-conn-client-key",
			expectedType: reflect.String.String(),
		},
		"test_flags_domain": {
			flag:         "domain",
			expectedType: reflect.String.String(),
		},
		"test_flags_reboot_enabled": {
			flag:         "reboot-enabled",
			expectedType: reflect.Bool.String(),
		},
		"test_flags_reboot_after": {
			flag:         "reboot-after",
			expectedType: reflect.String.String(),
		},
		"test_flags_domains": {
			flag:         "domains",
			expectedType: reflect.String.String(),
		},
		"test_flags_report_feedback_interval": {
			flag:         "report-feedback-interval",
			expectedType: reflect.String.String(),
		},
		"test_flags_current_state_delay": {
			flag:         "current-state-delay",
			expectedType: reflect.String.String(),
		},
		"test_flags_phase_timeout": {
			flag:         "phase-timeout",
			expectedType: reflect.String.String(),
		},
	}
	for testName, testCase := range tests {
		t.Run(testName, func(t *testing.T) {
			testFlag := flagSet.Lookup(testCase.flag)
			if testFlag == nil {
				t.Errorf("flag %s, not found", testCase.flag)
			}
			flagType, _ := flag.UnquoteUsage(testFlag)
			if flagType == "" {
				if testCase.expectedType != reflect.Bool.String() {
					t.Errorf("incorrect type: %s for flag %s, expecting: %s", reflect.Bool.String(), testFlag.Name, testCase.expectedType)
				}
				return
			}
			if flagType != testCase.expectedType {
				t.Errorf("incorrect type: %s for flag %s, expecting: %s", reflect.TypeOf(testFlag.Value).Name(), testFlag.Name, testCase.expectedType)
			}
		})
	}
}

func TestParseConfigFilePath(t *testing.T) {
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()
	testPath := "/some/path/file"

	t.Run("test_cfg_file_overridden", func(t *testing.T) {
		if ParseConfigFilePath() != logFileDefault {
			t.Error("config file not set to default")
		}
	})
	t.Run("test_cfg_file_default", func(t *testing.T) {
		os.Args = []string{oldArgs[0], fmt.Sprintf("--%s=%s", configFileFlagID, testPath)}
		if ParseConfigFilePath() != testPath {
			t.Error("config file not overridden by environment variable")
		}
	})
}

func TestParseDomainsFlag(t *testing.T) {
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()
	testDomains := "myDomain"

	t.Run("test_parse_domains_flag_1", func(t *testing.T) {
		os.Args = []string{oldArgs[0], fmt.Sprintf("-%s=%s", domainsFlagID, testDomains)}
		actualDomains := parseDomainsFlag()
		if len(actualDomains) != 1 && !actualDomains[testDomains] {
			t.Error("domains not set")
		}
	})

	t.Run("test_parse_domains_flag_2", func(t *testing.T) {
		os.Args = []string{oldArgs[0], fmt.Sprintf("--%s=%s", domainsFlagID, testDomains)}
		actualDomains := parseDomainsFlag()
		if len(actualDomains) != 1 && !actualDomains[testDomains] {
			t.Error("domains not set")
		}
	})

	t.Run("test_parse_domains_flag_3", func(t *testing.T) {
		os.Args = []string{oldArgs[0], fmt.Sprintf("-%s", domainsFlagID), testDomains}
		actualDomains := parseDomainsFlag()
		if len(actualDomains) != 1 && !actualDomains[testDomains] {
			t.Error("domains not set")
		}
	})

	t.Run("test_parse_domains_flag_4", func(t *testing.T) {
		os.Args = []string{oldArgs[0], fmt.Sprintf("-%s", domainsFlagID), testDomains}
		actualDomains := parseDomainsFlag()
		if len(actualDomains) != 1 && !actualDomains[testDomains] {
			t.Error("domains not set")
		}
	})

	t.Run("test_parse_domains_flag_err", func(t *testing.T) {
		invalidDomainsFlagID := "invalid"
		os.Args = []string{oldArgs[0], fmt.Sprintf("--%s=%s", invalidDomainsFlagID, testDomains)}
		actualDomains := parseDomainsFlag()
		if len(actualDomains) != 0 {
			t.Errorf("\"incorrect value: %v , expecting: empty \"", actualDomains)
		}
	})

	t.Run("test_parse_domains_flag_err_1", func(t *testing.T) {
		os.Args = []string{oldArgs[0], fmt.Sprintf("-%s", domainsFlagID)}
		actualDomains := parseDomainsFlag()
		if len(actualDomains) != 0 {
			t.Errorf("\"incorrect value: %v , expecting: empty \"", actualDomains)
		}
	})

	t.Run("test_parse_domains_flag_err_2", func(t *testing.T) {
		os.Args = []string{oldArgs[0], fmt.Sprintf("--%s", domainsFlagID)}
		actualDomains := parseDomainsFlag()
		if len(actualDomains) != 0 {
			t.Errorf("\"incorrect value: %v , expecting: empty \"", actualDomains)
		}
	})
}

func TestParseFlags(t *testing.T) {
	testVersion := "testVersion"
	t.Run("test_config_agent_not_configured_with_domain_specific_flags", func(t *testing.T) {
		oldArgs := os.Args
		defer func() { os.Args = oldArgs }()

		testConfigPath := "../config/testdata/config.json"
		expectedAgentsFromConfigFile := map[string]*api.UpdateManagerConfig{
			"self-update": {
				Name:           "self-update",
				RebootRequired: false,
				ReadTimeout:    "20s",
			},
			"containers": {
				Name:           "containers",
				RebootRequired: true,
				ReadTimeout:    "30s",
			},
			"test-domain": {
				Name:           "test-domain",
				RebootRequired: true,
				ReadTimeout:    "50s",
			},
		}

		os.Args = []string{oldArgs[0], fmt.Sprintf("--%s=%s", configFileFlagID, testConfigPath)}
		cfg := newDefaultConfig()
		configFilePath := ParseConfigFilePath()
		if configFilePath != "" {
			assert.NoError(t, LoadConfigFromFile(configFilePath, cfg))
		}
		parseFlags(cfg, testVersion)
		assert.Equal(t, expectedAgentsFromConfigFile, cfg.Agents)
	})
	t.Run("test_agent_not_configured_with_flags", func(t *testing.T) {
		oldArgs := os.Args
		defer func() { os.Args = oldArgs }()

		os.Args = []string{os.Args[0], ""}
		cfg := newDefaultConfig()
		expCfg := newDefaultConfig()
		expCfg.Agents = newDefaultAgentsConfig()
		parseFlags(cfg, testVersion)
		assert.Equal(t, expCfg.Agents, cfg.Agents)
	})
	t.Run("test_default_agent_reconfigured_with_flags", func(t *testing.T) {
		oldArgs := os.Args
		defer func() { os.Args = oldArgs }()

		reconfiguredRTO := "2m"
		os.Args = []string{os.Args[0], "--containers-read-timeout", reconfiguredRTO}

		reconfiguredDefaultContainersAgent := newDefaultAgentsConfig()
		reconfiguredDefaultContainersAgent["containers"].ReadTimeout = reconfiguredRTO
		cfg := newDefaultConfig()
		parseFlags(cfg, testVersion)
		assert.Equal(t, reconfiguredDefaultContainersAgent, cfg.Agents)
	})
	t.Run("test_agent_configured_with_all_domain_specific_flags", func(t *testing.T) {
		oldArgs := os.Args
		defer func() { os.Args = oldArgs }()

		testDomainName := "test-domain"
		testDomainRTO := "5m"
		testDomainRR := true
		testAgents := make(map[string]*api.UpdateManagerConfig)
		testAgents[testDomainName] = &api.UpdateManagerConfig{
			Name:           testDomainName,
			ReadTimeout:    testDomainRTO,
			RebootRequired: testDomainRR,
		}
		os.Args = []string{os.Args[0],
			fmt.Sprintf("--%s=%s", domainsFlagID, testDomainName),
			fmt.Sprintf("--%s=%s", "test-domain-read-timeout", testDomainRTO),
			fmt.Sprintf("--%s=%v", "test-domain-reboot-required", testDomainRR)}
		cfg := newDefaultConfig()
		parseFlags(cfg, testVersion)
		assert.Equal(t, testAgents, cfg.Agents)
	})
	t.Run("test_config_agent_reconfigured_with_domain_specific_flags", func(t *testing.T) {
		oldArgs := os.Args
		defer func() { os.Args = oldArgs }()

		testConfigPath := "../config/testdata/config.json"
		reconfiguredRTO := "2m"
		expectedAgentsFromConfigFile := map[string]*api.UpdateManagerConfig{
			"self-update": {
				Name:           "self-update",
				RebootRequired: false,
				ReadTimeout:    reconfiguredRTO,
			},
			"containers": {
				Name:           "containers",
				RebootRequired: true,
				ReadTimeout:    "30s",
			},
			"test-domain": {
				Name:           "test-domain",
				RebootRequired: true,
				ReadTimeout:    "50s",
			},
		}

		os.Args = []string{oldArgs[0], fmt.Sprintf("--%s=%s", configFileFlagID, testConfigPath),
			fmt.Sprintf("--%s=%s", "self-update-read-timeout", reconfiguredRTO)}
		cfg := newDefaultConfig()
		configFilePath := ParseConfigFilePath()
		if configFilePath != "" {
			assert.NoError(t, LoadConfigFromFile(configFilePath, cfg))
		}
		parseFlags(cfg, testVersion)
		assert.Equal(t, expectedAgentsFromConfigFile, cfg.Agents)
	})
	t.Run("test_overwrite_config_agents_with_domain_specific_flags", func(t *testing.T) {
		oldArgs := os.Args
		defer func() { os.Args = oldArgs }()

		testDomainName := "testDomain"
		testDomainRTO := "6m"
		testDomainRR := true
		testConfigPath := "../config/testdata/config.json"
		expectedAgents := map[string]*api.UpdateManagerConfig{
			testDomainName: {
				Name:           testDomainName,
				RebootRequired: testDomainRR,
				ReadTimeout:    testDomainRTO,
			},
		}

		os.Args = []string{oldArgs[0], fmt.Sprintf("--%s=%s", configFileFlagID, testConfigPath),
			fmt.Sprintf("--%s=%s", domainsFlagID, testDomainName),
			fmt.Sprintf("--%s=%v", "testDomain-reboot-required", testDomainRR),
			fmt.Sprintf("--%s=%s", "testDomain-read-timeout", testDomainRTO)}
		cfg := newDefaultConfig()
		configFilePath := ParseConfigFilePath()
		if configFilePath != "" {
			assert.NoError(t, LoadConfigFromFile(configFilePath, cfg))
		}
		parseFlags(cfg, testVersion)
		assert.Equal(t, expectedAgents, cfg.Agents)
	})
}
