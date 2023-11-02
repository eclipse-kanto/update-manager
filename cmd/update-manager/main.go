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

package main

import (
	"log"
	"os"

	"github.com/eclipse-kanto/update-manager/api"
	"github.com/eclipse-kanto/update-manager/cmd/update-manager/app"
	"github.com/eclipse-kanto/update-manager/config"
	"github.com/eclipse-kanto/update-manager/logger"
	"github.com/eclipse-kanto/update-manager/mqtt"
	"github.com/eclipse-kanto/update-manager/updatem/orchestration"
)

var (
	version = "development"
)

func main() {
	cfg, err := config.LoadConfig(version)
	if err != nil {
		log.Fatal("failed to load local configuration: ", err)
	}

	loggerOut, err := logger.SetupLogger(cfg.Log, "[update-manager]")
	if err != nil {
		log.Fatal("failed to initialize logger: ", err)
		return
	}
	defer loggerOut.Close()

	var client api.UpdateAgentClient
	if cfg.ThingsEnabled {
		client, err = mqtt.NewUpdateAgentThingsClient(cfg.Domain, cfg.MQTT)
	} else {
		client, err = mqtt.NewUpdateAgentClient(cfg.Domain, cfg.MQTT)
	}
	if err == nil {
		updateManager, err := orchestration.NewUpdateManager(version, cfg, client, orchestration.NewUpdateOrchestrator(cfg))
		if err == nil {
			err = app.Launch(cfg, client, updateManager)
		}
	}

	if err != nil {
		logger.Error("failed to init Update Manager", err, nil)
		loggerOut.Close()
		os.Exit(1)
	}
}
