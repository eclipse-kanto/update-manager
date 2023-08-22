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
	"github.com/eclipse-kanto/update-manager/api/types"
	"github.com/eclipse-kanto/update-manager/logger"
)

func (updateManager *aggregatedUpdateManager) HandleDesiredStateFeedbackEvent(domain, activityID, baseline string, status types.StatusType, message string, actions []*types.Action) {
	updateManager.eventLock.Lock()
	defer updateManager.eventLock.Unlock()

	updateManager.updateOrchestrator.HandleDesiredStateFeedbackEvent(domain, activityID, baseline, status, message, actions)
}

func (updateManager *aggregatedUpdateManager) HandleCurrentStateEvent(name string, activityID string, currentState *types.Inventory) {
	updateManager.eventLock.Lock()
	defer updateManager.eventLock.Unlock()

	logger.Debug("received current state for domain [%s] and activityID [%s]", name, activityID)
	updateManager.domainsInventory[name] = currentState
	if activityID == "" {
		inventory := toFullInventory(updateManager.asSoftwareNode(), updateManager.domainsInventory)
		updateManager.eventCallback.HandleCurrentStateEvent(updateManager.Name(), activityID, inventory)
	}
}
