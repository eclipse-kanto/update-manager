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
	"slices"
	"time"

	"github.com/eclipse-kanto/update-manager/api"
	"github.com/eclipse-kanto/update-manager/api/types"
	"github.com/eclipse-kanto/update-manager/logger"
)

var orderedCommands = []types.CommandType{types.CommandDownload, types.CommandUpdate, types.CommandActivate, types.CommandCleanup}

func (orchestrator *updateOrchestrator) apply(ctx context.Context) (bool, error) {
	orchestrator.notifyFeedback(types.StatusIdentifying, "")
	for updateManagerForDomain, statePerDomain := range orchestrator.operation.statesPerDomain {
		go func(updateManager api.UpdateManager, state *types.DesiredState) {
			updateManager.Apply(ctx, orchestrator.operation.activityID, state)
		}(updateManagerForDomain, statePerDomain)
	}

	// send DOWNLOAD command when identification is done
	running, err := orchestrator.waitCommandSignal(ctx, types.CommandDownload, handleCommandSignal)
	if err != nil {
		return false, err
	}
	// send the rest of the commands in order
	for i := 1; i < len(orderedCommands) && running; i++ {
		running, err = orchestrator.waitCommandSignal(ctx, orderedCommands[i], handleCommandSignal)
	}
	// wait for the last command(CLEANUP) to finish
	if running {
		_, _, err = orchestrator.waitSignal(ctx, orchestrator.operation.done)
	}
	return orchestrator.operation.rebootRequired && orchestrator.operation.status == types.StatusCompleted, err
}

type commandSignalHandler func(ctx context.Context, command types.CommandType, orchestrator *updateOrchestrator)

func (orchestrator *updateOrchestrator) waitCommandSignal(ctx context.Context, command types.CommandType, handle commandSignalHandler) (bool, error) {
	signalValue, timeout, err := orchestrator.waitSignal(ctx, orchestrator.operation.commandChannels[command])
	if err != nil {
		if timeout {
			if command == types.CommandDownload {
				orchestrator.operation.updateStatus(types.StatusIdentificationFailed)
			} else {
				orchestrator.operation.updateStatus(types.StatusIncomplete)
			}
		}
		return false, fmt.Errorf("failed to wait for command '%s' signal: %v", command, err)
	}
	if signalValue {
		go handle(ctx, command, orchestrator)
	}
	return signalValue, nil
}

func (orchestrator *updateOrchestrator) waitSignal(ctx context.Context, signal chan bool) (bool, bool, error) {
	select {
	case <-time.After(orchestrator.phaseTimeout):
		return false, true, fmt.Errorf("not received in %v", orchestrator.phaseTimeout)
	case <-orchestrator.operation.errChan:
		return false, false, fmt.Errorf(orchestrator.operation.errMsg)
	case value := <-signal:
		return value, false, nil
	case <-ctx.Done():
		orchestrator.operation.updateStatus(types.StatusIncomplete)
		return false, false, fmt.Errorf("the update manager instance is terminated")
	}
}

func handleCommandSignal(ctx context.Context, command types.CommandType, orchestrator *updateOrchestrator) {
	orchestrator.operationLock.Lock()
	defer orchestrator.operationLock.Unlock()

	if orchestrator.operation == nil {
		return
	}

	if err := orchestrator.getOwnerConsent(ctx, command); err != nil {
		// should a rollback be performed at this point?
		orchestrator.operation.errMsg = err.Error()
		orchestrator.operation.errChan <- true
		return
	}

	executeCommand := func(status types.StatusType) {
		for domain, domainStatus := range orchestrator.operation.domains {
			if domainStatus == status {
				orchestrator.command(ctx, orchestrator.operation.activityID, domain, command)
			}
		}
	}

	switch command {
	case types.CommandDownload:
		executeCommand(types.StatusIdentified)
	case types.CommandUpdate:
		executeCommand(types.BaselineStatusDownloadSuccess)
	case types.CommandActivate:
		executeCommand(types.BaselineStatusUpdateSuccess)
	case types.CommandCleanup:
		executeCommand(types.BaselineStatusActivationSuccess)
	case types.CommandRollback:
		// nothing to do
	default:
		logger.Error("unknown command %s", command)
	}
}

func (orchestrator *updateOrchestrator) getOwnerConsent(ctx context.Context, command types.CommandType) error {
	if command == "" || !slices.Contains(orchestrator.cfg.OwnerConsentCommands, command) {
		return nil
	}
	if command == types.CommandRollback || command == types.CommandCleanup {
		// no need for owner consent
		return nil
	}

	if orchestrator.ownerConsentClient == nil {
		return fmt.Errorf("owner consent client not available")
	}

	if err := orchestrator.ownerConsentClient.Start(orchestrator); err != nil {
		return err
	}
	defer func() {
		if err := orchestrator.ownerConsentClient.Stop(); err != nil {
			logger.Error("failed to stop owner consent client: %v", err)
		}
	}()

	if err := orchestrator.ownerConsentClient.SendOwnerConsent(orchestrator.operation.activityID, &types.OwnerConsent{Command: command}); err != nil {
		return err
	}

	select {
	case approved := <-orchestrator.operation.ownerConsented:
		if !approved {
			return fmt.Errorf("owner approval not granted")
		}
		return nil
	case <-time.After(orchestrator.phaseTimeout):
		return fmt.Errorf("owner consent not granted in %v", orchestrator.phaseTimeout)
	case <-ctx.Done():
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
