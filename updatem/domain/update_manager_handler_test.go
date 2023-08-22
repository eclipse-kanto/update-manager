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
	"fmt"
	"strconv"
	"testing"

	"github.com/eclipse-kanto/update-manager/api/types"
	"github.com/eclipse-kanto/update-manager/test/mocks"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestHandleDesiredStateFeedbackNoActiveOperation(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	desiredStateClient := mocks.NewMockDesiredStateClient(mockCtrl)
	desiredStateClient.EXPECT().Domain().Return(testDomain)
	eventCallback := mocks.NewMockUpdateManagerCallback(mockCtrl)
	updateManager := createTestDomainUpdateManager(desiredStateClient, eventCallback)

	assert.Nil(t, updateManager.HandleDesiredStateFeedback("", 0, nil))
}

func TestHandleDesiredStateFeedback(t *testing.T) {
	testStatus := []types.StatusType{types.StatusCompleted, types.StatusIncomplete, types.StatusRunning}

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	desiredStateClient := mocks.NewMockDesiredStateClient(mockCtrl)
	desiredStateClient.EXPECT().Domain().Return(testDomain).AnyTimes()
	eventCallback := mocks.NewMockUpdateManagerCallback(mockCtrl)
	for i, status := range testStatus {
		eventCallback.EXPECT().HandleDesiredStateFeedbackEvent(testDomain, testActivityID, fmt.Sprintf("baseline%d", i), status, fmt.Sprintf("desired state operation %s", status), []*types.Action{})
	}

	updateManager := createTestDomainUpdateManager(desiredStateClient, eventCallback)
	updateManager.updateOperation = newUpdateOperation(testActivityID)

	for i, status := range testStatus {
		assert.Nil(t, updateManager.HandleDesiredStateFeedback(testActivityID, 0, &types.DesiredStateFeedback{
			Baseline: "baseline" + strconv.Itoa(i),
			Status:   status,
			Message:  "desired state operation " + string(status),
			Actions:  []*types.Action{},
		}))
	}
}

func TestHandleDesiredStateFeedbackMismatchActivityID(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	desiredStateClient := mocks.NewMockDesiredStateClient(mockCtrl)
	desiredStateClient.EXPECT().Domain().Return(testDomain).Times(2)
	eventCallback := mocks.NewMockUpdateManagerCallback(mockCtrl)

	updateManager := createTestDomainUpdateManager(desiredStateClient, eventCallback)
	updateManager.updateOperation = newUpdateOperation(testActivityID)
	assert.Nil(t, updateManager.HandleDesiredStateFeedback("wrongActivityID", 0, nil))
}

func TestHandleCurrentState(t *testing.T) {
	tests := map[string]struct {
		oldTimestamp int64
		newTimestamp int64
		oldInventory *types.Inventory
		newInventory *types.Inventory
		ignoreUpdate bool
	}{
		"initial-current-state": {
			newInventory: dummyInventory,
		},
		"newer-current-state": {
			oldTimestamp: 12345,
			newTimestamp: 54321,
			oldInventory: simpleInventory,
			newInventory: dummyInventory,
		},
		"current-state-with-older-timestamp": {
			oldTimestamp: 54321,
			newTimestamp: 12345,
			oldInventory: simpleInventory,
			newInventory: dummyInventory,
			ignoreUpdate: true,
		},
	}

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	desiredStateClient := mocks.NewMockDesiredStateClient(mockCtrl)
	desiredStateClient.EXPECT().Domain().Return(testDomain).AnyTimes()
	eventCallback := mocks.NewMockUpdateManagerCallback(mockCtrl)

	for name, test := range tests {
		t.Log(name)

		if !test.ignoreUpdate {
			eventCallback.EXPECT().HandleCurrentStateEvent(testDomain, "", test.newInventory)
		}

		updateManager := createTestDomainUpdateManager(desiredStateClient, eventCallback)
		updateManager.currentState.inventory = test.oldInventory
		updateManager.currentState.timestamp = test.oldTimestamp

		assert.Nil(t, updateManager.HandleCurrentState("", test.newTimestamp, test.newInventory))
		if test.ignoreUpdate {
			assert.Equal(t, test.oldInventory, updateManager.currentState.inventory)
			assert.Equal(t, test.oldTimestamp, updateManager.currentState.timestamp)
		} else {
			assert.Equal(t, test.newInventory, updateManager.currentState.inventory)
			assert.Equal(t, test.newTimestamp, updateManager.currentState.timestamp)
		}
	}
}
