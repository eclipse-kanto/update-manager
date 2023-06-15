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

package domain

import (
	"github.com/eclipse-kanto/update-manager/api/types"
	"github.com/eclipse-kanto/update-manager/logger"
)

func (updateManager *domainUpdateManager) HandleDesiredStateFeedback(activityID string, timestamp int64, desiredStateFeedback *types.DesiredStateFeedback) error {
	updateManager.desiredStateLock.Lock()
	defer updateManager.desiredStateLock.Unlock()

	if updateManager.updateOperation == nil {
		logger.Debug("[%s] ignoring received desired state feedback: %v", updateManager.Name(), desiredStateFeedback)
		return nil
	}

	logger.Debug("[%s] received desired state feedback: %v", updateManager.Name(), desiredStateFeedback)

	if updateManager.updateOperation.activityID != activityID {
		logger.Warn("[%s] activity id mismatch for received desired state feedback event  - expecting %s, received %s",
			updateManager.Name(), updateManager.updateOperation.activityID, activityID)
		return nil
	}

	updateManager.eventCallback.HandleDesiredStateFeedbackEvent(updateManager.Name(), activityID, desiredStateFeedback.Baseline, desiredStateFeedback.Status, desiredStateFeedback.Message, desiredStateFeedback.Actions)
	return nil
}

func (updateManager *domainUpdateManager) HandleCurrentState(activityID string, timestamp int64, inventory *types.Inventory) error {
	updateManager.currentStateLock.Lock()
	defer updateManager.currentStateLock.Unlock()

	logger.Debug("[%s] received current state event: %v", updateManager.Name(), inventory)

	if updateManager.currentState.inventory != nil && (timestamp < updateManager.currentState.timestamp) {
		logger.Warn("[%s] received current state event with outdated timestamp - last known %s, received %s",
			updateManager.Name(), updateManager.currentState.timestamp, timestamp)
		return nil
	}

	// update current state
	updateManager.currentState.inventory = inventory
	updateManager.currentState.timestamp = timestamp
	updateManager.eventCallback.HandleCurrentStateEvent(updateManager.Name(), activityID, inventory)

	if activityID != "" && activityID == updateManager.currentState.expActivityID {
		updateManager.currentState.receiveChan <- true
	}

	return nil
}
