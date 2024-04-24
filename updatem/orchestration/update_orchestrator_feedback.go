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

	"github.com/eclipse-kanto/update-manager/api/types"
	"github.com/eclipse-kanto/update-manager/api/util"
	"github.com/eclipse-kanto/update-manager/logger"
)

type statusHandler func(*updateOrchestrator, string, string, []*types.Action)

var statusHandlers = map[types.StatusType]statusHandler{
	types.StatusIdentified:                handleDomainIdentified,
	types.StatusIdentificationFailed:      handleDomainIdentificationFailed,
	types.StatusCompleted:                 handleDomainCompletedEvent,
	types.StatusIncomplete:                handleDomainIncomplete,
	types.BaselineStatusDownloading:       handleDomainDownloading,
	types.BaselineStatusDownloadSuccess:   handleDomainDownloadSuccess,
	types.BaselineStatusDownloadFailure:   handleDomainDownloadFailure,
	types.BaselineStatusUpdating:          handleDomainUpdating,
	types.BaselineStatusUpdateSuccess:     handleDomainUpdateSuccess,
	types.BaselineStatusUpdateFailure:     handleDomainUpdateFailure,
	types.BaselineStatusActivating:        handleDomainActivating,
	types.BaselineStatusActivationSuccess: handleDomainActivationSuccess,
	types.BaselineStatusActivationFailure: handleDomainActivationFailure,
	types.BaselineStatusCleanup:           handleDomainCleanup,
	types.BaselineStatusCleanupSuccess:    handleDomainCleanupSuccess,
	types.BaselineStatusCleanupFailure:    handleDomainCleanupFailure,
}

func (orchestrator *updateOrchestrator) HandleDesiredStateFeedbackEvent(domain, activityID, baseline string, status types.StatusType, message string, actions []*types.Action) {
	orchestrator.operationLock.Lock()
	defer orchestrator.operationLock.Unlock()

	if !orchestrator.validateActivity(domain, activityID) {
		return
	}

	orchestrator.updateActions(domain, actions)

	if handler, ok := statusHandlers[status]; !ok {
		logger.Warn("received desired state feedback event for baseline [%s] and domain [%s] with unsupported status '%s'", baseline, domain, status)
	} else {
		handler(orchestrator, domain, message, actions)
	}

}

func (orchestrator *updateOrchestrator) validateActivity(domain, activityID string) bool {
	if orchestrator.operation == nil {
		logger.Warn("received desired state feedback event for domain [%s], but there is no active update operation", domain)
		return false
	}
	if orchestrator.operation.activityID != activityID {
		logger.Warn("activity id mismatch for received desired state feedback event for domain [%s]  - expecting %s, received %s", domain, orchestrator.operation.activityID, activityID)
		return false
	}
	if _, ok := orchestrator.operation.domains[domain]; !ok {
		logger.Warn("received desired state feedback event for unexpected domain [%s]", domain)
		return false
	}
	return true
}

func handleDomainIdentified(orchestrator *updateOrchestrator, domain, message string, actions []*types.Action) {
	if !orchestrator.checkIdentificationStatus(domain) {
		return
	}
	if len(actions) == 0 {
		// no actions, any further commands shall not be sent
		orchestrator.operation.domains[domain] = types.BaselineStatusCleanupSuccess
	} else {
		orchestrator.operation.domains[domain] = types.StatusIdentified
	}

	isIdentificationFailed := false
	isIdentified := false
	for _, domainUpdateStatus := range orchestrator.operation.domains {
		if domainUpdateStatus == types.StatusIdentificationFailed {
			isIdentificationFailed = true
		} else if domainUpdateStatus == types.StatusIdentified {
			isIdentified = true
		}
		if domainUpdateStatus == types.StatusIdentifying {
			// there are domains still identifying, wait for them
			return
		}
	}
	if isIdentificationFailed {
		orchestrator.updateStatusIdentificationFailed(domain, "")
		return
	}

	orchestrator.notifyFeedback(types.StatusIdentified, "")
	if isIdentified {
		orchestrator.domainUpdateRunning()
		orchestrator.operation.updateStatus(types.StatusRunning)
		orchestrator.operation.commandChannels[types.CommandDownload] <- true
	} else {
		// no actions(status CleanupSuccess for all domains), operation is done
		orchestrator.operation.updateStatus(types.StatusCompleted)
		orchestrator.operation.commandChannels[types.CommandDownload] <- false
	}
}

func handleDomainIdentificationFailed(orchestrator *updateOrchestrator, domain, message string, actions []*types.Action) {
	if !orchestrator.checkIdentificationStatus(domain) {
		return
	}
	orchestrator.operation.domains[domain] = types.StatusIdentificationFailed
	for _, domainUpdateStatus := range orchestrator.operation.domains {
		if domainUpdateStatus == types.StatusIdentifying {
			return
		}
	}
	orchestrator.updateStatusIdentificationFailed(domain, message)
}

func handleDomainCompletedEvent(orchestrator *updateOrchestrator, domain, message string, actions []*types.Action) {
	domainStatus := orchestrator.operation.domains[domain]
	if domainStatus == types.StatusCompleted || domainStatus == types.StatusIncomplete ||
		domainStatus == types.BaselineStatusCleanupSuccess || domainStatus == types.BaselineStatusCleanupFailure {
		return
	}
	orchestrator.operation.domains[domain] = types.BaselineStatusCleanupSuccess
	orchestrator.domainUpdateCompleted()
}

func handleDomainIncomplete(orchestrator *updateOrchestrator, domain, message string, actions []*types.Action) {
	domainStatus := orchestrator.operation.domains[domain]
	if domainStatus == types.StatusCompleted || domainStatus == types.StatusIncomplete ||
		domainStatus == types.BaselineStatusCleanupSuccess || domainStatus == types.BaselineStatusCleanupFailure {
		return
	}
	orchestrator.operation.delayedStatus = types.StatusIncomplete
	orchestrator.operation.domains[domain] = types.BaselineStatusCleanupFailure
	orchestrator.domainUpdateCompleted()
}

func handleDomainDownloadSuccess(orchestrator *updateOrchestrator, domain, message string, actions []*types.Action) {
	if orchestrator.operation.status != types.StatusRunning {
		return
	}
	domainStatus := orchestrator.operation.domains[domain]
	if domainStatus != types.StatusIdentified && domainStatus != types.BaselineStatusDownloading {
		return
	}
	orchestrator.operation.domains[domain] = types.BaselineStatusDownloadSuccess
	for _, status := range orchestrator.operation.domains {
		if status == types.StatusIdentified || status == types.BaselineStatusDownloading {
			return
		}
	}
	orchestrator.operation.commandChannels[types.CommandUpdate] <- true
	orchestrator.domainUpdateRunning()
}

func handleDomainDownloadFailure(orchestrator *updateOrchestrator, domain, message string, actions []*types.Action) {
	if orchestrator.operation.status != types.StatusRunning {
		return
	}
	domainStatus := orchestrator.operation.domains[domain]
	if domainStatus != types.StatusIdentified && domainStatus != types.BaselineStatusDownloading {
		return
	}
	orchestrator.operation.delayedStatus = types.StatusIncomplete
	orchestrator.operation.domains[domain] = types.BaselineStatusDownloadFailure
	orchestrator.command(context.Background(), orchestrator.operation.activityID, domain, types.CommandCleanup)
}

func handleDomainDownloading(orchestrator *updateOrchestrator, domain, message string, actions []*types.Action) {
	if orchestrator.operation.status != types.StatusRunning {
		return
	}
	domainStatus := orchestrator.operation.domains[domain]
	if domainStatus != types.StatusIdentified && domainStatus != types.BaselineStatusDownloading {
		return
	}
	orchestrator.domainUpdateRunning()
}

func handleDomainUpdateSuccess(orchestrator *updateOrchestrator, domain, message string, actions []*types.Action) {
	if orchestrator.operation.status != types.StatusRunning {
		return
	}
	domainStatus := orchestrator.operation.domains[domain]
	if domainStatus != types.BaselineStatusDownloadSuccess && domainStatus != types.BaselineStatusUpdating {
		return
	}
	orchestrator.operation.domains[domain] = types.BaselineStatusUpdateSuccess
	for _, status := range orchestrator.operation.domains {
		if status == types.BaselineStatusDownloadSuccess || status == types.BaselineStatusUpdating {
			return
		}
	}
	orchestrator.operation.commandChannels[types.CommandActivate] <- true
	orchestrator.domainUpdateRunning()
}

func handleDomainUpdateFailure(orchestrator *updateOrchestrator, domain, message string, actions []*types.Action) {
	if orchestrator.operation.status != types.StatusRunning {
		return
	}
	domainStatus := orchestrator.operation.domains[domain]
	if domainStatus != types.BaselineStatusDownloadSuccess && domainStatus != types.BaselineStatusUpdating {
		return
	}
	orchestrator.operation.delayedStatus = types.StatusIncomplete
	orchestrator.operation.domains[domain] = types.BaselineStatusUpdateFailure
	orchestrator.command(context.Background(), orchestrator.operation.activityID, domain, types.CommandCleanup)
}

func handleDomainUpdating(orchestrator *updateOrchestrator, domain, message string, actions []*types.Action) {
	if orchestrator.operation.status != types.StatusRunning {
		return
	}
	domainStatus := orchestrator.operation.domains[domain]
	if domainStatus != types.BaselineStatusDownloadSuccess && domainStatus != types.BaselineStatusUpdating {
		return
	}
	orchestrator.domainUpdateRunning()
}

func handleDomainActivationSuccess(orchestrator *updateOrchestrator, domain, message string, actions []*types.Action) {
	if orchestrator.operation.status != types.StatusRunning {
		return
	}
	domainStatus := orchestrator.operation.domains[domain]
	if domainStatus != types.BaselineStatusUpdateSuccess && domainStatus != types.BaselineStatusActivating {
		return
	}
	orchestrator.operation.domains[domain] = types.BaselineStatusActivationSuccess

	for _, status := range orchestrator.operation.domains {
		if status == types.BaselineStatusUpdateSuccess || status == types.BaselineStatusActivating {
			return
		}
	}
	orchestrator.operation.commandChannels[types.CommandCleanup] <- true
	orchestrator.domainUpdateRunning()
}

func handleDomainActivationFailure(orchestrator *updateOrchestrator, domain, message string, actions []*types.Action) {
	if orchestrator.operation.status != types.StatusRunning {
		return
	}
	domainStatus := orchestrator.operation.domains[domain]
	if domainStatus != types.BaselineStatusUpdateSuccess && domainStatus != types.BaselineStatusActivating {
		return
	}
	orchestrator.operation.delayedStatus = types.StatusIncomplete
	orchestrator.operation.domains[domain] = types.BaselineStatusActivationFailure
	orchestrator.command(context.Background(), orchestrator.operation.activityID, domain, types.CommandCleanup)
}

func handleDomainActivating(orchestrator *updateOrchestrator, domain, message string, actions []*types.Action) {
	if orchestrator.operation.status != types.StatusRunning {
		return
	}
	domainStatus := orchestrator.operation.domains[domain]
	if domainStatus != types.BaselineStatusUpdateSuccess && domainStatus != types.BaselineStatusActivating {
		return
	}
	orchestrator.domainUpdateRunning()
}

func handleDomainCleanupSuccess(orchestrator *updateOrchestrator, domain, message string, actions []*types.Action) {
	domainStatus := orchestrator.operation.domains[domain]
	if domainStatus != types.BaselineStatusActivationSuccess && domainStatus != types.BaselineStatusDownloadFailure &&
		domainStatus != types.BaselineStatusUpdateFailure && domainStatus != types.BaselineStatusActivationFailure &&
		domainStatus != types.StatusCompleted && domainStatus != types.StatusIncomplete {
		return
	}
	orchestrator.operation.domains[domain] = types.BaselineStatusCleanupSuccess
	orchestrator.domainUpdateCompleted()
}

func handleDomainCleanupFailure(orchestrator *updateOrchestrator, domain, message string, actions []*types.Action) {
	domainStatus := orchestrator.operation.domains[domain]
	if domainStatus != types.BaselineStatusActivationSuccess && domainStatus != types.BaselineStatusDownloadFailure &&
		domainStatus != types.BaselineStatusUpdateFailure && domainStatus != types.BaselineStatusActivationFailure &&
		domainStatus != types.StatusCompleted && domainStatus != types.StatusIncomplete {
		return
	}
	orchestrator.operation.domains[domain] = types.BaselineStatusCleanupFailure
	orchestrator.domainUpdateCompleted()
}

func handleDomainCleanup(orchestrator *updateOrchestrator, domain, message string, actions []*types.Action) {
	domainStatus := orchestrator.operation.domains[domain]
	if domainStatus != types.BaselineStatusActivationSuccess && domainStatus != types.BaselineStatusDownloadFailure &&
		domainStatus != types.BaselineStatusUpdateFailure && domainStatus != types.BaselineStatusActivationFailure {
		return
	}
	orchestrator.domainUpdateRunning()
}

func (orchestrator *updateOrchestrator) domainUpdateCompleted() {
	for _, domainUpdateStatus := range orchestrator.operation.domains {
		if domainUpdateStatus != types.BaselineStatusCleanupSuccess && domainUpdateStatus != types.BaselineStatusCleanupFailure {
			return
		}
	}
	for domain, actions := range orchestrator.operation.actions {
		domainCfg := orchestrator.cfg.Agents[domain]
		if domainCfg != nil {
			if len(actions) > 0 && domainCfg.RebootRequired {
				orchestrator.operation.rebootRequired = true
				break
			}
		}
	}
	if orchestrator.operation.delayedStatus == types.StatusIncomplete {
		orchestrator.operation.updateStatus(types.StatusIncomplete)
		orchestrator.operation.errMsg = "the update process is incompleted"
		orchestrator.operation.errChan <- true
		return
	}
	orchestrator.operation.updateStatus(types.StatusCompleted)
	orchestrator.operation.done <- true
}

func (orchestrator *updateOrchestrator) domainUpdateRunning() {
	orchestrator.notifyFeedback(types.StatusRunning, "")
}

func (orchestrator *updateOrchestrator) updateStatusIdentificationFailed(domain, errMsg string) {
	orchestrator.operation.updateStatus(types.StatusIdentificationFailed)
	orchestrator.operation.errMsg = fmt.Sprintf("[%s]: %s", domain, errMsg)
	orchestrator.operation.errChan <- true
}

func (orchestrator *updateOrchestrator) checkIdentificationStatus(domain string) bool {
	if orchestrator.operation.status != types.StatusIdentifying && orchestrator.operation.status != types.StatusIdentified &&
		orchestrator.operation.status != types.StatusIdentificationFailed {
		return false
	}
	domainUpdateStatus := orchestrator.operation.domains[domain]
	if domainUpdateStatus == types.StatusIdentified {
		logger.Warn("update for domain [%s] has already identified", domain)
		return false
	}
	if domainUpdateStatus == types.StatusIdentificationFailed {
		logger.Warn("update for domain [%s] has already failed identification", domain)
		return false
	}
	return true
}

func (orchestrator *updateOrchestrator) updateActions(domain string, actions []*types.Action) {
	orchestrator.actionsLock.Lock()
	defer orchestrator.actionsLock.Unlock()

	domainActions := orchestrator.operation.actions[domain]
	if domainActions == nil {
		orchestrator.operation.actions[domain] = map[string]*types.Action{}
	}
	for _, action := range actions {
		orchestrator.operation.actions[domain][action.Component.ID] = action
	}
}

func (orchestrator *updateOrchestrator) toActionsList() []*types.Action {
	orchestrator.actionsLock.Lock()
	defer orchestrator.actionsLock.Unlock()

	actions := []*types.Action{}
	for _, domainActions := range orchestrator.operation.actions {
		for _, domainAction := range domainActions {
			actions = append(actions, util.FixActivationActionStatus(domainAction))
		}
	}
	return actions
}

func (orchestrator *updateOrchestrator) notifyFeedback(status types.StatusType, message string) {
	orchestrator.operation.desiredStateCallback.HandleDesiredStateFeedbackEvent(
		orchestrator.Name(), orchestrator.operation.activityID, "", util.FixIncompleteInconsistentStatus(status), message, orchestrator.toActionsList())
}
