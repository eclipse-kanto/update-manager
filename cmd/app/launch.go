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

package app

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/eclipse-kanto/update-manager/api"
	"github.com/eclipse-kanto/update-manager/api/agent"
	"github.com/eclipse-kanto/update-manager/api/util"
	"github.com/eclipse-kanto/update-manager/config"
	"github.com/eclipse-kanto/update-manager/logger"
)

const (
	defaultReportFeedbackInterval = 60 * time.Second
	defaultCurrentStateDelay      = 30 * time.Second
)

// Launch is the entry point for lauching of the Update Manager instance
func Launch(cfg *config.Config, client api.UpdateAgentClient, updateManager api.UpdateManager) error {
	ua, err := initComponent(cfg, client, updateManager)
	if err != nil {
		logger.ErrorErr(err, "failed to init Update Manager")
		return err
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err = startComponent(ctx, ua)
	if err != nil {
		logger.ErrorErr(err, "failed to start Update Manager")
		return err
	}
	logger.Debug("successfully started Update Manager")

	var signalChan = make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGHUP)

	sig := <-signalChan
	cancel()
	logger.Debug("received OS SIGNAL >> %d ! Will exit!", sig)
	stopComponent(ua)

	return nil
}

func startComponent(ctx context.Context, agent api.UpdateAgent) error {
	logger.Debug("starting Update Manager")
	return agent.Start(ctx)
}

func stopComponent(agent api.UpdateAgent) {
	logger.Debug("stopping Update Manager")
	agent.Stop()
	logger.Debug("stopping Update Manager finished")
}

func initComponent(cfg *config.Config, client api.UpdateAgentClient, manager api.UpdateManager) (api.UpdateAgent, error) {
	logger.Debug("creating Update Manager instance")
	return agent.NewUpdateAgent(client, manager,
		agent.WithCurrentStateReportDelay(util.ParseDuration("current-state-delay", cfg.CurrentStateDelay, defaultCurrentStateDelay, 0*time.Minute)),
		agent.WithDesiredStateFeedbackReportInterval(util.ParseDuration("report-feedback-interval", cfg.ReportFeedbackInterval, defaultReportFeedbackInterval, 0*time.Minute))), nil
}
