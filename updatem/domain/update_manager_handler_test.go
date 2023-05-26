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
	"testing"

	"github.com/eclipse-kanto/update-manager/api/types"
	"github.com/eclipse-kanto/update-manager/test/mocks"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

const feedbackDummyStatus1 = `
{
	"activityId": "testActivityId",
	"payload": {
		"status": "DUMMY_STATUS_1",
		"message": "dummy status 1",
		"actions":[
		]
	}
  }
`
const feedbackDummyStatus2 = `
{
	"activityId": "testActivityId",
	"payload": {
		"status": "DUMMY_STATUS_2",
		"message": "dummy status 2",
		"actions":[
		]
	}
  }
`

const feedbackDummyStatus3 = `
{
	"activityId": "testActivityId",
	"payload": {
		"status": "DUMMY_STATUS_3",
		"message": "dummy status 3",
		"actions":[
		]
	}
  }
`

const stateCompletedWrongActivityID = `
{
	"activityId": "wrongActivityId",
	"payload": {
		"status": "COMPLETED",
		"message": "completed operation",
		"actions":[
		]
	}
  }
`

const dummyStatus1 types.StatusType = "DUMMY_STATUS_1"
const dummyStatus2 types.StatusType = "DUMMY_STATUS_2"
const dummyStatus3 types.StatusType = "DUMMY_STATUS_3"

func TestNoActiveOperation(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	desiredStateClient := mocks.NewMockDesiredStateClient(mockCtrl)
	desiredStateClient.EXPECT().Domain().Return(testDomain)
	eventCallback := mocks.NewMockUpdateManagerCallback(mockCtrl)

	updateManager := createTestDomainUpdateManager(desiredStateClient, eventCallback)
	err := updateManager.HandleDesiredStateFeedback(nil)
	assert.Nil(t, err)
}

func TestHandleDesiredStateFeedback(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	desiredStateClient := mocks.NewMockDesiredStateClient(mockCtrl)
	desiredStateClient.EXPECT().Domain().Return(testDomain).Times(6)

	eventCallback := mocks.NewMockUpdateManagerCallback(mockCtrl)
	eventCallback.EXPECT().HandleDesiredStateFeedbackEvent(testDomain, testActivityID, "", dummyStatus1, "dummy status 1", []*types.Action{})
	eventCallback.EXPECT().HandleDesiredStateFeedbackEvent(testDomain, testActivityID, "", dummyStatus2, "dummy status 2", []*types.Action{})
	eventCallback.EXPECT().HandleDesiredStateFeedbackEvent(testDomain, testActivityID, "", dummyStatus3, "dummy status 3", []*types.Action{})

	updateManager := createTestDomainUpdateManager(desiredStateClient, eventCallback)
	updateManager.updateOperation = newUpdateOperation(testActivityID)
	err := updateManager.HandleDesiredStateFeedback([]byte(feedbackDummyStatus1))
	assert.Nil(t, err)
	err = updateManager.HandleDesiredStateFeedback([]byte(feedbackDummyStatus2))
	assert.Nil(t, err)
	err = updateManager.HandleDesiredStateFeedback([]byte(feedbackDummyStatus3))
	assert.Nil(t, err)
}

func TestMismatchActivityID(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	desiredStateClient := mocks.NewMockDesiredStateClient(mockCtrl)
	eventCallback := mocks.NewMockUpdateManagerCallback(mockCtrl)
	desiredStateClient.EXPECT().Domain().Return(testDomain).Times(2)

	updateManager := createTestDomainUpdateManager(desiredStateClient, eventCallback)
	updateManager.updateOperation = newUpdateOperation(testActivityID)
	err := updateManager.HandleDesiredStateFeedback([]byte(stateCompletedWrongActivityID))
	assert.Nil(t, err)
}

func TestCurrentState(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	desiredStateClient := mocks.NewMockDesiredStateClient(mockCtrl)
	desiredStateClient.EXPECT().Domain().Return(testDomain)
	eventCallback := mocks.NewMockUpdateManagerCallback(mockCtrl)

	updateManager := createTestDomainUpdateManager(desiredStateClient, eventCallback)
	err := updateManager.HandleCurrentState(nil)
	assert.Nil(t, err)
}
