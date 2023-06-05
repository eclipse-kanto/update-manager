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

func (updateManager *domainUpdateManager) HandleDesiredStateFeedback(desiredStateFeedback []byte) error {
	updateManager.desiredStateLock.Lock()
	defer updateManager.desiredStateLock.Unlock()

	if updateManager.updateOperation == nil {
		logger.Debug("[%s] ignoring received desired state feedback: %s", updateManager.Name(), desiredStateFeedback)
		return nil
	}

	logger.Debug("[%s] received desired state feedback: %s", updateManager.Name(), desiredStateFeedback)
	envelope, err := types.FromEnvelope(desiredStateFeedback, &types.DesiredStateFeedback{})
	if err != nil {
		return err
	}

	ds := envelope.Payload.(*types.DesiredStateFeedback)
	if updateManager.updateOperation.activityID != envelope.ActivityID {
		logger.Warn("[%s] activity id mismatch for received desired state feedback event  - expecting %s, received %s",
			updateManager.Name(), updateManager.updateOperation.activityID, envelope.ActivityID)
		return nil
	}

	updateManager.eventCallback.HandleDesiredStateFeedbackEvent(updateManager.Name(), envelope.ActivityID, ds.Baseline, ds.Status, ds.Message, ds.Actions)
	return nil
}

func (updateManager *domainUpdateManager) HandleCurrentState(currentState []byte) error {
	if currentState == nil {
		logger.Warn("[%s] received empty current state, ignoring it", updateManager.Name())
		return nil
	}

	logger.Debug("[%s] received current state: %s", updateManager.Name(), currentState)
	envelope, err := types.FromEnvelope(currentState, &types.Inventory{})
	if err != nil {
		return err
	}

	updateManager.currentStateLock.Lock()
	defer updateManager.currentStateLock.Unlock()

	inventory := envelope.Payload.(*types.Inventory)
	if updateManager.currentState.inventory != nil && (envelope.Timestamp < updateManager.currentState.timestamp) {
		logger.Warn("[%d] received current state event with outdated timestamp - expecting %s, received %s",
			updateManager.currentState.timestamp, envelope.Timestamp)
		return nil
	}
	updateManager.updateCurrentState(envelope.ActivityID, envelope.Timestamp, inventory)
	return nil
}

func (updateManager *domainUpdateManager) updateCurrentState(activityID string, timestamp int64, newInventory *types.Inventory) {
	updateManager.currentState.inventory = newInventory
	updateManager.currentState.timestamp = timestamp
	updateManager.eventCallback.HandleCurrentStateEvent(updateManager.Name(), activityID, newInventory)

	if activityID != "" && activityID == updateManager.currentState.expActivityID {
		updateManager.currentState.receiveChan <- true
	}
}
