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

	if err := orchestrator.waitIdentification(ctx); err != nil {
		return false, err
	}

	err := orchestrator.waitCompletion(ctx)
	return orchestrator.operation.rebootRequired && orchestrator.operation.status == types.StatusCompleted, err
}

func (orchestrator *updateOrchestrator) waitIdentification(ctx context.Context) error {
	select {
	case <-orchestrator.operation.identErrChan:
		return fmt.Errorf(orchestrator.operation.identErrMsg)
	case <-orchestrator.operation.errChan:
		return fmt.Errorf(orchestrator.operation.errMsg)
	case <-orchestrator.operation.identDone:
		logger.Debug("the identification phase is completed")
		return nil
	case <-ctx.Done():
		orchestrator.operation.status = types.StatusIncomplete
		return fmt.Errorf("the update manager instance is terminated")
	}
}

func (orchestrator *updateOrchestrator) waitCompletion(ctx context.Context) error {
	select {
	case <-orchestrator.operation.errChan:
		return fmt.Errorf(orchestrator.operation.errMsg)
	case <-orchestrator.operation.done:
		return nil
	case <-ctx.Done():
		orchestrator.operation.status = types.StatusIncomplete
		return fmt.Errorf("the update manager instance is terminated")
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
			desiredStateCallback: desiredStateCallback
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
