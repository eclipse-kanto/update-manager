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
	"fmt"
	"time"

	"github.com/eclipse-kanto/update-manager/api"
	"github.com/eclipse-kanto/update-manager/api/types"
	"github.com/eclipse-kanto/update-manager/logger"
)

func (orchestrator *updateOrchestrator) apply(ctx context.Context) (bool, error) {
	orchestrator.notifyFeedback(types.StatusIdentifying, "")
	for updateManagerForDomain, statePerDomain := range orchestrator.operation.statesPerDomain {
		go func(updateManager api.UpdateManager, state *types.DesiredState) {
			updateManager.Apply(ctx, orchestrator.operation.activityID, state)
		}(updateManagerForDomain, statePerDomain)
	}

	wait, err := orchestrator.waitPhase(ctx, phaseIdentification, handlePhaseCompletion)
	if err != nil {
		return false, err
	}

	for i := 1; i < len(orderedPhases) && wait; i++ {
		wait, err = orchestrator.waitPhase(ctx, orderedPhases[i], handlePhaseCompletion)
	}
	return orchestrator.operation.rebootRequired && orchestrator.operation.status == types.StatusCompleted, err
}

type phaseHandler func(ctx context.Context, phase phase, orchestrator *updateOrchestrator)

func (orchestrator *updateOrchestrator) waitPhase(ctx context.Context, currentPhase phase, handle phaseHandler) (bool, error) {
	select {
	case <-time.After(orchestrator.phaseTimeout):
		if currentPhase == phaseIdentification {
			orchestrator.operation.updateStatus(types.StatusIdentificationFailed)
		} else {
			orchestrator.operation.updateStatus(types.StatusIncomplete)
		}
		return false, fmt.Errorf("%s phase not done in %v", currentPhase, orchestrator.phaseTimeout)
	case <-orchestrator.operation.errChan:
		return false, fmt.Errorf(orchestrator.operation.errMsg)
	case running := <-orchestrator.operation.phaseChannels[currentPhase]:
		logger.Info("the %s phase is done", currentPhase)
		if running {
			go handle(ctx, currentPhase, orchestrator)
			return true, nil
		}
		return false, nil
	case <-ctx.Done():
		orchestrator.operation.updateStatus(types.StatusIncomplete)
		return false, fmt.Errorf("the update manager instance is terminated")
	}
}

func handlePhaseCompletion(ctx context.Context, completedPhase phase, orchestrator *updateOrchestrator) {
	orchestrator.operationLock.Lock()
	defer orchestrator.operationLock.Unlock()

	if orchestrator.operation == nil {
		return
	}

	executeCommand := func(status types.StatusType, command types.CommandType) {
		for domain, domainStatus := range orchestrator.operation.domains {
			if domainStatus == status {
				orchestrator.command(ctx, orchestrator.operation.activityID, domain, command)
			}
		}
	}
	switch completedPhase {
	case phaseIdentification:
		executeCommand(types.StatusIdentified, types.CommandDownload)
	case phaseDownload:
		executeCommand(types.BaselineStatusDownloadSuccess, types.CommandUpdate)
	case phaseUpdate:
		executeCommand(types.BaselineStatusUpdateSuccess, types.CommandActivate)
	case phaseActivation:
		executeCommand(types.BaselineStatusActivationSuccess, types.CommandCleanup)
	default:
		logger.Error("unknown phase %s", completedPhase)
	}
}

func (orchestrator *updateOrchestrator) command(ctx context.Context, activityID, domain string, commandName types.CommandType) {
	domainAgent := orchestrator.getDomainAgent(domain)
	if domainAgent == nil {
		return
	}
	command := &types.DesiredStateCommand{
		Command: commandName,
	}
	domainAgent.Command(ctx, activityID, command)
}

func (orchestrator *updateOrchestrator) getDomainAgent(name string) api.UpdateManager {
	for domainAgent := range orchestrator.operation.statesPerDomain {
		if domainAgent.Name() == name {
			return domainAgent
		}
	}
	return nil
}

func (orchestrator *updateOrchestrator) setupUpdateOperation(domainAgents map[string]api.UpdateManager,
	activityID string, desiredState *types.DesiredState, desiredStateCallback api.DesiredStateFeedbackHandler) error {
	orchestrator.operationLock.Lock()
	defer orchestrator.operationLock.Unlock()

	operation, err := newUpdateOperation(domainAgents, activityID, desiredState, desiredStateCallback)
	if err != nil {
		orchestrator.operation = &updateOperation{
			status:               types.StatusIncomplete,
			desiredStateCallback: desiredStateCallback,
		}
		return err
	}
	orchestrator.operation = operation
	return nil
}

func (orchestrator *updateOrchestrator) disposeUpdateOperation() {
	orchestrator.operationLock.Lock()
	defer orchestrator.operationLock.Unlock()
	orchestrator.operation = nil
}
