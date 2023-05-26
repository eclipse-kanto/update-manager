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

package orchestration

import (
	"context"
	"sync"
	"time"

	"github.com/eclipse-kanto/update-manager/api"
	"github.com/eclipse-kanto/update-manager/api/types"
	"github.com/eclipse-kanto/update-manager/api/util"
	"github.com/eclipse-kanto/update-manager/config"
	"github.com/eclipse-kanto/update-manager/logger"
)

type updateOrchestrator struct {
	operationLock sync.Mutex
	actionsLock   sync.Mutex

	cfg          *config.Config
	phaseTimeout time.Duration

	operation *updateOperation
}

func (orchestrator *updateOrchestrator) Name() string {
	return orchestrator.cfg.Domain
}

// NewUpdateOrchestrator creates a new update orchestrator that does not handle cross-domain dependencies
func NewUpdateOrchestrator(cfg *config.Config) api.UpdateOrchestrator {
	return &updateOrchestrator{
		cfg:          cfg,
		phaseTimeout: util.ParseDuration("phase-timeout", cfg.PhaseTimeout, 10*time.Minute, 10*time.Minute),
	}
}

func (orchestrator *updateOrchestrator) Apply(ctx context.Context, domainAgents map[string]api.UpdateManager,
	activityID string, desiredState *types.DesiredState, desiredStateCallback api.DesiredStateFeedbackHandler) bool {
	var applyErr error
	defer func() {
		message := ""
		if applyErr != nil {
			message = applyErr.Error()
		}
		orchestrator.notifyFeedback(orchestrator.operation.status, message)
		orchestrator.disposeUpdateOperation()
	}()

	err := orchestrator.setupUpdateOperation(domainAgents, activityID, desiredState, desiredStateCallback)
	if err != nil {
		logger.Error(err.Error())
		applyErr = err
		return false
	}

	rebootRequired, applyErr := orchestrator.apply(ctx)
	return rebootRequired
}
