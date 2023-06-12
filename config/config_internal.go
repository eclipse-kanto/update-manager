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

import "github.com/eclipse-kanto/update-manager/api"

const (
	// default log config
	logFileDefault       = ""
	logLevelDefault      = "INFO"
	logFileSizeDefault   = 2
	logFileCountDefault  = 5
	logFileMaxAgeDefault = 28

	domainDefault                 = "device"
	rebootEnabledDefault          = true
	rebootAfterDefault            = "30s"
	reportFeedbackIntervalDefault = "1m"
	currentStateDelayDefault      = "30s"
	phaseTimeoutDefault           = "10m"
	readTimeoutDefault            = "1m"

	domainContainers = "containers"
)

// Config represents the Update Manager configuration.
type Config struct {
	*BaseConfig
	Agents                 map[string]*api.UpdateManagerConfig `json:"agents,omitempty"`
	RebootEnabled          bool                                `json:"rebootEnabled"`
	RebootAfter            string                              `json:"rebootAfter"`
	ReportFeedbackInterval string                              `json:"reportFeedbackInterval"`
	CurrentStateDelay      string                              `json:"currentStateDelay"`
	PhaseTimeout           string                              `json:"phaseTimeout"`
}

func newDefaultConfig() *Config {
	return &Config{
		BaseConfig:             DefaultDomainConfig(domainDefault),
		Agents:                 nil,
		RebootEnabled:          rebootEnabledDefault,
		RebootAfter:            rebootAfterDefault,
		ReportFeedbackInterval: reportFeedbackIntervalDefault,
		CurrentStateDelay:      currentStateDelayDefault,
		PhaseTimeout:           phaseTimeoutDefault,
	}
}

func newDefaultAgentsConfig() map[string]*api.UpdateManagerConfig {
	agents := make(map[string]*api.UpdateManagerConfig)

	addAgentConfig(agents, domainContainers)

	return agents
}

func addAgentConfig(config map[string]*api.UpdateManagerConfig, name string) {
	config[name] = &api.UpdateManagerConfig{
		Name:           name,
		RebootRequired: false,
		ReadTimeout:    readTimeoutDefault,
	}
}

// LoadConfig loads a new configuration instance using flags and config file (if set).
func LoadConfig(version string) (*Config, error) {
	configFilePath := ParseConfigFilePath()
	config := newDefaultConfig()
	if configFilePath != "" {
		if err := LoadConfigFromFile(configFilePath, config); err != nil {
			return nil, err
		}
	}
	parseFlags(config, version)
	return config, nil
}

func prepareAgentsConfig(cfg *Config, domains map[string]bool) {
	if cfg.Agents == nil {
		cfg.Agents = newDefaultAgentsConfig()
	}
	if len(domains) > 0 {
		for name := range cfg.Agents {
			if _, ok := domains[name]; !ok {
				// delete all agent configs not present as domains flag
				delete(cfg.Agents, name)
			} else {
				// remove known domain from internal map
				delete(domains, name)
			}
		}
		// create new agent config for all unknown domains
		for name := range domains {
			addAgentConfig(cfg.Agents, name)
		}
	}
}
