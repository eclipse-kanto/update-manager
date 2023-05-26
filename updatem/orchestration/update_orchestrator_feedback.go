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
	"strings"

	"github.com/eclipse-kanto/update-manager/api/types"
	"github.com/eclipse-kanto/update-manager/api/util"
	"github.com/eclipse-kanto/update-manager/logger"
)

func (orchestrator *updateOrchestrator) HandleDesiredStateFeedbackEvent(domain, activityID, baseline string, status types.StatusType, message string, actions []*types.Action) {
	orchestrator.operationLock.Lock()
	defer orchestrator.operationLock.Unlock()

	if !orchestrator.validateActivity(domain, activityID, baseline, status) {
		return
	}

	orchestrator.updateActions(domain, actions)

	switch status {
	case types.StatusIdentified:
		orchestrator.handleDomainIdentified(domain)
	case types.StatusIdentificationFailed:
		orchestrator.handleDomainIdentificationFailed(domain, message)
	case types.StatusCompleted:
		orchestrator.handleDomainCompletedEvent(domain)
	case types.StatusIncomplete:
		orchestrator.handleDomainIncomplete(domain, message)
	case types.BaselineStatusDownloading:
		orchestrator.handleDomainDownloading(domain)
	case types.BaselineStatusDownloadSuccess:
		orchestrator.handleDomainDownloadSuccess(domain)
	case types.BaselineStatusDownloadFailure:
		orchestrator.handleDomainDownloadFailure(domain, message)
	case types.BaselineStatusUpdating:
		orchestrator.handleDomainUpdating(domain)
	case types.BaselineStatusUpdateSuccess:
		orchestrator.handleDomainUpdateSuccess(domain)
	case types.BaselineStatusUpdateFailure:
		orchestrator.handleDomainUpdateFailure(domain, message)
	case types.BaselineStatusActivating:
		orchestrator.handleDomainActivating(domain)
	case types.BaselineStatusActivationSuccess:
		orchestrator.handleDomainActivationSuccess(domain)
	case types.BaselineStatusActivationFailure:
		orchestrator.handleDomainActivationFailure(domain, message)
	case types.BaselineStatusCleanup:
		orchestrator.handleDomainCleanup(domain)
	case types.BaselineStatusCleanupSuccess:
		orchestrator.handleDomainCleanupSuccess(domain)
	case types.BaselineStatusCleanupFailure:
		orchestrator.handleDomainCleanupFailure(domain, message)
	}
}

func (orchestrator *updateOrchestrator) validateActivity(domain, activityID, baseline string, status types.StatusType) bool {
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
	if status != types.StatusIdentified && status != types.StatusIdentificationFailed &&
		status != types.StatusCompleted && status != types.StatusIncomplete &&
		status != types.BaselineStatusDownloading && status != types.BaselineStatusDownloadSuccess && status != types.BaselineStatusDownloadFailure &&
		status != types.BaselineStatusUpdating && status != types.BaselineStatusUpdateSuccess && status != types.BaselineStatusUpdateFailure &&
		status != types.BaselineStatusActivating && status != types.BaselineStatusActivationSuccess && status != types.BaselineStatusActivationFailure &&
		status != types.BaselineStatusCleanup && status != types.BaselineStatusCleanupSuccess && status != types.BaselineStatusCleanupFailure &&
		status != types.BaselineStatusRollback && status != types.BaselineStatusRollbackSuccess && status != types.BaselineStatusRollbackFailure {
		logger.Warn("received desired state feedback event for baseline [%s] and domain [%s] with unsupported status '%s'", baseline, domain, status)
	}
	return true
}

func (orchestrator *updateOrchestrator) handleDomainIdentified(domain string) {
	if !orchestrator.checkIdentificationStatus(domain) {
		return
	}
	isIdentificationFailed := false
	orchestrator.operation.domains[domain] = types.StatusIdentified
	for _, domainUpdateStatus := range orchestrator.operation.domains {
		if domainUpdateStatus == types.StatusIdentificationFailed {
			isIdentificationFailed = true
		}
		if domainUpdateStatus == types.StatusIdentifying {
			return
		}
	}
	if isIdentificationFailed {
		orchestrator.updateStatusIdentificationFailed(domain, "")
		return
	}
	orchestrator.notifyFeedback(types.StatusIdentified, "")
	orchestrator.updateStatusRunning()
	orchestrator.operation.status = types.StatusRunning
	for domain := range orchestrator.operation.domains {
		orchestrator.command(context.Background(), orchestrator.operation.activityID, domain, types.CommandDownload)
	}
	orchestrator.operation.identDone <- true
}

func (orchestrator *updateOrchestrator) handleDomainIdentificationFailed(domain, errMsg string) {
	if !orchestrator.checkIdentificationStatus(domain) {
		return
	}
	orchestrator.operation.domains[domain] = types.StatusIdentificationFailed
	for _, domainUpdateStatus := range orchestrator.operation.domains {
		if domainUpdateStatus == types.StatusIdentifying {
			return
		}
	}
	orchestrator.updateStatusIdentificationFailed(domain, errMsg)
}

func (orchestrator *updateOrchestrator) handleDomainCompletedEvent(domain string) {
	if orchestrator.operation.status != types.StatusRunning {
		return
	}
	domainStatus := orchestrator.operation.domains[domain]
	if domainStatus == types.StatusCompleted || domainStatus == types.StatusIncomplete {
		return
	}
	orchestrator.operation.domains[domain] = types.StatusCompleted
	orchestrator.command(context.Background(), orchestrator.operation.activityID, domain, types.CommandCleanup)
}

func (orchestrator *updateOrchestrator) handleDomainIncomplete(domain, errMsg string) {
	if orchestrator.operation.status != types.StatusRunning {
		return
	}
	domainStatus := orchestrator.operation.domains[domain]
	if domainStatus == types.StatusCompleted || domainStatus == types.StatusIncomplete {
		return
	}
	orchestrator.operation.delayedStatus = types.StatusIncomplete
	orchestrator.operation.domains[domain] = types.StatusIncomplete
	orchestrator.command(context.Background(), orchestrator.operation.activityID, domain, types.CommandCleanup)
}

func (orchestrator *updateOrchestrator) handleDomainDownloadSuccess(domain string) {
	if orchestrator.operation.status != types.StatusRunning {
		return
	}
	domainStatus := orchestrator.operation.domains[domain]
	if domainStatus != types.StatusIdentified && domainStatus != types.BaselineStatusDownloading {
		return
	}
	orchestrator.operation.domains[domain] = types.BaselineStatusDownloadSuccess
	orchestrator.command(context.Background(), orchestrator.operation.activityID, domain, types.CommandUpdate)
	orchestrator.domainUpdateRunning()
}

func (orchestrator *updateOrchestrator) handleDomainDownloadFailure(domain string, errMsg string) {
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

func (orchestrator *updateOrchestrator) handleDomainDownloading(domain string) {
	if orchestrator.operation.status != types.StatusRunning {
		return
	}
	domainStatus := orchestrator.operation.domains[domain]
	if domainStatus != types.StatusIdentified && domainStatus != types.BaselineStatusDownloading {
		return
	}
	orchestrator.domainUpdateRunning()
}

func (orchestrator *updateOrchestrator) handleDomainUpdateSuccess(domain string) {
	if orchestrator.operation.status != types.StatusRunning {
		return
	}
	domainStatus := orchestrator.operation.domains[domain]
	if domainStatus != types.BaselineStatusDownloadSuccess && domainStatus != types.BaselineStatusUpdating {
		return
	}
	orchestrator.operation.domains[domain] = types.BaselineStatusUpdateSuccess
	orchestrator.command(context.Background(), orchestrator.operation.activityID, domain, types.CommandActivate)
	orchestrator.domainUpdateRunning()
}

func (orchestrator *updateOrchestrator) handleDomainUpdateFailure(domain string, errMsg string) {
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

func (orchestrator *updateOrchestrator) handleDomainUpdating(domain string) {
	if orchestrator.operation.status != types.StatusRunning {
		return
	}
	domainStatus := orchestrator.operation.domains[domain]
	if domainStatus != types.BaselineStatusDownloadSuccess && domainStatus != types.BaselineStatusUpdating {
		return
	}
	orchestrator.domainUpdateRunning()
}

func (orchestrator *updateOrchestrator) handleDomainActivationSuccess(domain string) {
	if orchestrator.operation.status != types.StatusRunning {
		return
	}
	domainStatus := orchestrator.operation.domains[domain]
	if domainStatus != types.BaselineStatusUpdateSuccess && domainStatus != types.BaselineStatusActivating {
		return
	}
	orchestrator.operation.domains[domain] = types.BaselineStatusActivationSuccess
	orchestrator.command(context.Background(), orchestrator.operation.activityID, domain, types.CommandCleanup)
	orchestrator.domainUpdateRunning()
}

func (orchestrator *updateOrchestrator) handleDomainActivationFailure(domain string, errMsg string) {
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

func (orchestrator *updateOrchestrator) handleDomainActivating(domain string) {
	if orchestrator.operation.status != types.StatusRunning {
		return
	}
	domainStatus := orchestrator.operation.domains[domain]
	if domainStatus != types.BaselineStatusUpdateSuccess && domainStatus != types.BaselineStatusActivating {
		return
	}
	orchestrator.domainUpdateRunning()
}

func (orchestrator *updateOrchestrator) handleDomainCleanupSuccess(domain string) {
	if orchestrator.operation.status != types.StatusRunning {
		return
	}
	domainStatus := orchestrator.operation.domains[domain]
	if domainStatus != types.BaselineStatusActivationSuccess && domainStatus != types.BaselineStatusDownloadFailure &&
		domainStatus != types.BaselineStatusUpdateFailure && domainStatus != types.BaselineStatusActivationFailure &&
		domainStatus != types.StatusCompleted && domainStatus != types.StatusIncomplete {
		return
	}
	orchestrator.operation.domains[domain] = types.BaselineStatusCleanupSuccess
	orchestrator.domainUpdateCompleted(domain)
}

func (orchestrator *updateOrchestrator) handleDomainCleanupFailure(domain string, errMsg string) {
	if orchestrator.operation.status != types.StatusRunning {
		return
	}
	domainStatus := orchestrator.operation.domains[domain]
	if domainStatus != types.BaselineStatusActivationSuccess && domainStatus != types.BaselineStatusDownloadFailure &&
		domainStatus != types.BaselineStatusUpdateFailure && domainStatus != types.BaselineStatusActivationFailure &&
		domainStatus != types.StatusCompleted && domainStatus != types.StatusIncomplete {
		return
	}
	orchestrator.operation.domains[domain] = types.BaselineStatusCleanupFailure
	orchestrator.domainUpdateCompleted(domain)
}

func (orchestrator *updateOrchestrator) handleDomainCleanup(domain string) {
	if orchestrator.operation.status != types.StatusRunning {
		return
	}
	domainStatus := orchestrator.operation.domains[domain]
	if domainStatus != types.BaselineStatusActivationSuccess && domainStatus != types.BaselineStatusDownloadFailure &&
		domainStatus != types.BaselineStatusUpdateFailure && domainStatus != types.BaselineStatusActivationFailure {
		return
	}
	orchestrator.domainUpdateRunning()
}

func (orchestrator *updateOrchestrator) domainUpdateCompleted(domain string) {
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
		orchestrator.operation.status = types.StatusIncomplete
		orchestrator.operation.errMsg = "the update process is incompleted"
		orchestrator.operation.errChan <- true
		return
	}
	orchestrator.operation.status = types.StatusCompleted
	orchestrator.operation.identDone <- true
	orchestrator.operation.done <- true
}

func (orchestrator *updateOrchestrator) domainUpdateRunning() {
	orchestrator.notifyFeedback(types.StatusRunning, "")
}

func (orchestrator *updateOrchestrator) updateStatusRunning() {
	orchestrator.notifyFeedback(types.StatusRunning, "")
}

func (orchestrator *updateOrchestrator) updateStatusIdentificationFailed(domain, errMsg string) {
	orchestrator.operation.status = types.StatusIdentificationFailed
	orchestrator.operation.identErrMsg = fmt.Sprintf("[%s]: %s", domain, errMsg)
	orchestrator.operation.identErrChan <- true
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

	if logger.IsTraceEnabled() {
		traceMsg := fmt.Sprintf("the current actions for domain [%s] are: %v", domain, orchestrator.toActionsString())
		logger.Trace(traceMsg)
	}
	domainActions := orchestrator.operation.actions[domain]
	if domainActions == nil {
		orchestrator.operation.actions[domain] = map[string]*types.Action{}
	}
	for _, action := range actions {
		orchestrator.operation.actions[domain][action.Component.ID] = action
	}

	if logger.IsTraceEnabled() {
		traceMsg := fmt.Sprintf("the updated actions for domain [%s] are: %s", domain, orchestrator.toActionsString())
		logger.Trace(traceMsg)
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

func (orchestrator *updateOrchestrator) toActionsString() string {
	actionsString := "["
	for _, domainActions := range orchestrator.operation.actions {
		for _, domainAction := range domainActions {
			actionString := fmt.Sprintf("{Component:{ID:%s Version:%s} Status:%s Progress:%v Message:%s}", domainAction.Component.ID, domainAction.Component.Version, domainAction.Status, domainAction.Progress, domainAction.Message)
			actionsString = actionsString + actionString + " "
		}
	}
	actionsString = strings.Trim(actionsString, " ") + "]"
	return actionsString
}

func (orchestrator *updateOrchestrator) notifyFeedback(status types.StatusType, message string) {
	orchestrator.operation.desiredStateCallback.HandleDesiredStateFeedbackEvent(
		orchestrator.Name(), orchestrator.operation.activityID, "", util.FixIncompleteInconsistentStatus(status), message, orchestrator.toActionsList())
}
