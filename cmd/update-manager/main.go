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

	uac, um, err := initUpdateManager(cfg)
	if err == nil {
		err = app.Launch(cfg, uac, um)
	}

	if err != nil {
		logger.Error("failed to init Update Manager", err, nil)
		loggerOut.Close()
		os.Exit(1)
	}
}

func initUpdateManager(cfg *config.Config) (api.UpdateAgentClient, api.UpdateManager, error) {
	var (
		uac api.UpdateAgentClient
		occ api.OwnerConsentClient
		um  api.UpdateManager
		err error
	)

	if cfg.ThingsEnabled {
		uac, err = mqtt.NewUpdateAgentThingsClient(cfg.Domain, cfg.MQTT)
	} else {
		uac, err = mqtt.NewUpdateAgentClient(cfg.Domain, cfg.MQTT)
	}
	if err != nil {
		return nil, nil, err
	}

	if len(cfg.OwnerConsentPhases) != 0 {
		if occ, err = mqtt.NewOwnerConsentClient(cfg.Domain, uac); err != nil {
			return nil, nil, err
		}
	}
	if um, err = orchestration.NewUpdateManager(version, cfg, uac, orchestration.NewUpdateOrchestrator(cfg, occ)); err != nil {
		return nil, nil, err
	}
	return uac, um, nil
}
