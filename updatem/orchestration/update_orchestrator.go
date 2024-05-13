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

	cfg                 *config.Config
	phaseTimeout        time.Duration
	ownerConsentTimeout time.Duration
	ownerConsentClient  api.OwnerConsentClient

	operation *updateOperation
}

func (orchestrator *updateOrchestrator) Name() string {
	return orchestrator.cfg.Domain
}

// NewUpdateOrchestrator creates a new update orchestrator that does not handle cross-domain dependencies
func NewUpdateOrchestrator(cfg *config.Config, ownerApprovalClient api.OwnerConsentClient) api.UpdateOrchestrator {
	ua := &updateOrchestrator{
		cfg:                 cfg,
		phaseTimeout:        util.ParseDuration("phase-timeout", cfg.PhaseTimeout, 10*time.Minute, 10*time.Minute),
		ownerConsentTimeout: util.ParseDuration("owner-consent-timeout", cfg.OwnerConsentTimeout, 30*time.Minute, 30*time.Minute),
		ownerConsentClient:  ownerApprovalClient,
	}
	return ua
}

// Apply is called by the update manager.
// It triggers the update process with the given activity ID and desired state specification and orchestrates the process on the given domain update agents.
// The method returns true if reboot is required after the operation is complete.
// Whether reboot is necessary shall be determined by the domain update agents.
// For example, usually container updates shall not need reboot, but if a new OS image is written on the device it may be activated on next reboot.
func (orchestrator *updateOrchestrator) Apply(ctx context.Context, domainAgents map[string]api.UpdateManager,
	activityID string, desiredState *types.DesiredState, desiredStateCallback api.DesiredStateFeedbackHandler) bool {
	var applyErr error
	defer func() {
		message := ""
		if applyErr != nil {
			message = applyErr.Error()
		}

		orchestrator.operation.statusLock.Lock()
		status := orchestrator.operation.status
		orchestrator.operation.statusLock.Unlock()

		orchestrator.notifyFeedback(status, message)
		orchestrator.disposeUpdateOperation()
	}()

	if err := orchestrator.setupUpdateOperation(domainAgents, activityID, desiredState, desiredStateCallback); err != nil {
		logger.Error(err.Error())
		applyErr = err
		return false
	}

	rebootRequired, applyErr := orchestrator.apply(ctx)
	if applyErr != nil {
		logger.Error("failed to apply '%s' desired state: %v", activityID, applyErr)
	}
	return rebootRequired
}

func (orchestrator *updateOrchestrator) HandleOwnerConsentFeedback(activityID string, timestamp int64, consent *types.OwnerConsentFeedback) error {
	if orchestrator.operation != nil && activityID == orchestrator.operation.activityID {
		logger.Info("owner consent received with status: %v, timestamp: %d", consent.Status, timestamp)
		orchestrator.operation.ownerConsented <- consent.Status == types.StatusApproved
	}
	return nil
}
