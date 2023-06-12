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
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/eclipse-kanto/update-manager/api"
	"github.com/eclipse-kanto/update-manager/api/types"
	"github.com/eclipse-kanto/update-manager/test/mocks"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

const (
	testDomain         = "test-domain"
	testActivityID     = "testActivityId"
	initCurrentStateID = "initial-current-state"
)

var dummyInventory *types.Inventory = &types.Inventory{
	HardwareNodes: []*types.HardwareNode{
		{
			InventoryNode: types.InventoryNode{
				ID:      "hardware-node-id",
				Version: "1.0.0",
				Name:    "Hardware Node",
				Parameters: []*types.KeyValuePair{
					{
						Key:   "x",
						Value: "y",
					},
				},
			},
			Addressable: true,
		},
	},
	SoftwareNodes: []*types.SoftwareNode{
		{
			InventoryNode: types.InventoryNode{
				ID:      "software-node-id",
				Version: "1.0.0",
				Name:    "Software Node",
				Parameters: []*types.KeyValuePair{
					{
						Key:   "x",
						Value: "y",
					},
				},
			},
			Type: types.SoftwareTypeApplication,
		},
	},
	Associations: []*types.Association{
		{
			SourceID: "hardware-node-id",
			TargetID: "software-node-id",
		},
	},
}

var command *types.DesiredStateCommand = &types.DesiredStateCommand{
	Baseline: "dummyBaseline",
	Command:  "dummyCommand",
}

func TestUpdateManagerName(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	desiredStateClient := mocks.NewMockDesiredStateClient(mockCtrl)
	desiredStateClient.EXPECT().Domain().Return(testDomain)

	assert.Equal(t, testDomain, createTestDomainUpdateManager(desiredStateClient, nil).Name())
}

func TestGetCurrentState(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	eventCallback := mocks.NewMockUpdateManagerCallback(mockCtrl)
	eventCallback.EXPECT().HandleCurrentStateEvent(testDomain, initCurrentStateID, dummyInventory)

	updateManager := createTestDomainUpdateManager(nil, eventCallback)
	updateManager.readTimeout = time.Second

	desiredStateClient := mocks.NewMockDesiredStateClient(mockCtrl)
	desiredStateClient.EXPECT().PublishGetCurrentState(gomock.Any()).DoAndReturn(
		func(bytes []byte) error {
			envelope := &types.Envelope{
				ActivityID: initCurrentStateID,
				Timestamp:  time.Now().UnixNano() / int64(time.Millisecond),
				Payload:    dummyInventory,
			}
			envelopeBytes, _ := json.Marshal(envelope)
			assert.Nil(t, json.Unmarshal(bytes, envelope))
			assert.Equal(t, initCurrentStateID, envelope.ActivityID)
			assert.Nil(t, updateManager.HandleCurrentState(envelopeBytes))
			return nil
		},
	)
	desiredStateClient.EXPECT().Domain().Return(testDomain).Times(3)
	updateManager.desiredStateClient = desiredStateClient

	currentState, err := updateManager.Get(context.Background(), initCurrentStateID)
	assert.Nil(t, err)
	assert.Equal(t, dummyInventory, currentState)
}

func TestGetCurrentStateByActivityID(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	eventCallback := mocks.NewMockUpdateManagerCallback(mockCtrl)
	eventCallback.EXPECT().HandleCurrentStateEvent(testDomain, "testActivityId", dummyInventory)

	updateManager := createTestDomainUpdateManager(nil, eventCallback)
	updateManager.readTimeout = time.Second

	desiredStateClient := mocks.NewMockDesiredStateClient(mockCtrl)
	desiredStateClient.EXPECT().PublishGetCurrentState(gomock.Any()).DoAndReturn(
		func(bytes []byte) error {
			envelope := &types.Envelope{
				ActivityID: testActivityID,
				Timestamp:  time.Now().UnixNano() / int64(time.Millisecond),
				Payload:    dummyInventory,
			}
			envelopeBytes, _ := json.Marshal(envelope)
			assert.Nil(t, json.Unmarshal(bytes, envelope))
			assert.Equal(t, testActivityID, envelope.ActivityID)
			assert.Nil(t, updateManager.HandleCurrentState(envelopeBytes))
			return nil
		},
	)
	desiredStateClient.EXPECT().Domain().Return(testDomain).Times(3)
	updateManager.desiredStateClient = desiredStateClient

	currentState, err := updateManager.Get(context.Background(), testActivityID)
	assert.Nil(t, err)
	assert.Equal(t, dummyInventory, currentState)
}

func TestGetCurrentStateInvalidPayload(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	eventCallback := mocks.NewMockUpdateManagerCallback(mockCtrl)
	eventCallback.EXPECT().HandleCurrentStateEvent(testDomain, testActivityID, dummyInventory)

	updateManager := createTestDomainUpdateManager(nil, eventCallback)
	updateManager.readTimeout = time.Second

	desiredStateClient := mocks.NewMockDesiredStateClient(mockCtrl)
	desiredStateClient.EXPECT().PublishGetCurrentState(gomock.Any()).DoAndReturn(
		func(bytes []byte) error {
			var envelope *types.Envelope

			envelope = &types.Envelope{
				ActivityID: testActivityID,
				Timestamp:  time.Now().UnixNano() / int64(time.Millisecond),
				Payload:    "invalid-payload",
			}
			assert.Nil(t, json.Unmarshal(bytes, envelope))
			assert.Equal(t, testActivityID, envelope.ActivityID)
			assert.NotNil(t, updateManager.HandleCurrentState([]byte("invalid-payload")))

			envelope = &types.Envelope{
				ActivityID: testActivityID,
				Timestamp:  time.Now().UnixNano() / int64(time.Millisecond),
				Payload:    dummyInventory,
			}
			envelopeBytes, _ := json.Marshal(envelope)
			assert.Nil(t, json.Unmarshal(bytes, envelope))
			assert.Equal(t, testActivityID, envelope.ActivityID)
			assert.Nil(t, updateManager.HandleCurrentState(envelopeBytes))

			return nil
		},
	)
	desiredStateClient.EXPECT().Domain().Return(testDomain).AnyTimes()
	updateManager.desiredStateClient = desiredStateClient

	currentState, err := updateManager.Get(context.Background(), testActivityID)
	assert.Nil(t, err)
	assert.Equal(t, dummyInventory, currentState)
}

func TestGetCurrentStateTimeout(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	updateManager := createTestDomainUpdateManager(nil, nil)

	desiredStateClient := mocks.NewMockDesiredStateClient(mockCtrl)
	desiredStateClient.EXPECT().PublishGetCurrentState(gomock.Any())
	desiredStateClient.EXPECT().Domain().Return(testDomain)
	updateManager.desiredStateClient = desiredStateClient

	currentState, err := updateManager.Get(context.Background(), "testActivityID")
	assert.Nil(t, currentState)
	assert.NotNil(t, err)
}

func TestGetCurrentStateMismatchActivityID(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	eventCallback := mocks.NewMockUpdateManagerCallback(mockCtrl)
	eventCallback.EXPECT().HandleCurrentStateEvent(testDomain, testActivityID, dummyInventory)

	updateManager := createTestDomainUpdateManager(nil, eventCallback)
	updateManager.readTimeout = time.Second

	desiredStateClient := mocks.NewMockDesiredStateClient(mockCtrl)
	desiredStateClient.EXPECT().PublishGetCurrentState(gomock.Any()).DoAndReturn(
		func(bytes []byte) error {
			envelope := &types.Envelope{
				ActivityID: testActivityID,
				Timestamp:  time.Now().UnixNano() / int64(time.Millisecond),
				Payload:    dummyInventory,
			}
			envelopeBytes, _ := json.Marshal(envelope)
			assert.Nil(t, json.Unmarshal(bytes, envelope))
			assert.Equal(t, "wrongActivityId", envelope.ActivityID)
			assert.Nil(t, updateManager.HandleCurrentState(envelopeBytes))
			return nil
		},
	)
	desiredStateClient.EXPECT().Domain().Return(testDomain).AnyTimes()
	updateManager.desiredStateClient = desiredStateClient

	currentState, err := updateManager.Get(context.Background(), "wrongActivityId")
	assert.Equal(t, dummyInventory, currentState)
	assert.NotNil(t, err)
}

func TestGetCurrentStateUpdateManagerTerminated(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	updateManager := createTestDomainUpdateManager(nil, nil)

	desiredStateClient := mocks.NewMockDesiredStateClient(mockCtrl)
	desiredStateClient.EXPECT().PublishGetCurrentState(gomock.Any())
	desiredStateClient.EXPECT().Domain().Return(testDomain).Times(1)
	updateManager.desiredStateClient = desiredStateClient

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	currentState, err := updateManager.Get(ctx, "testActivityID")
	assert.Nil(t, currentState)
	assert.NotNil(t, err)
}

func TestDisposeUpdateManager(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	desiredStateClient := mocks.NewMockDesiredStateClient(mockCtrl)
	desiredStateClient.EXPECT().Unsubscribe()

	assert.Nil(t, createTestDomainUpdateManager(desiredStateClient, nil).Dispose())
}

func TestSetCallback(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	eventCallback := mocks.NewMockUpdateManagerCallback(mockCtrl)

	updateManager := createTestDomainUpdateManager(nil, nil)
	updateManager.SetCallback(eventCallback)
	assert.Equal(t, eventCallback, updateManager.eventCallback)
}

func TestStartWatchEvents(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	desiredStateClient := mocks.NewMockDesiredStateClient(mockCtrl)
	updateManager := createTestDomainUpdateManager(desiredStateClient, nil)
	desiredStateClient.EXPECT().Subscribe(gomock.Any())

	updateManager.WatchEvents(context.Background())
}

func TestCommandRequest(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	desiredStateClient := mocks.NewMockDesiredStateClient(mockCtrl)
	desiredStateClient.EXPECT().Domain().Return(testDomain).AnyTimes()
	desiredStateClient.EXPECT().PublishDesiredStateCommand(gomock.Any())
	eventCallback := mocks.NewMockUpdateManagerCallback(mockCtrl)

	updateManager := createTestDomainUpdateManager(desiredStateClient, eventCallback)
	updateManager.updateOperation = &updateOperation{
		activityID: testActivityID,
	}

	updateManager.Command(context.Background(), testActivityID, command)
}

func TestCommandRequestPublishError(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	desiredStateClient := mocks.NewMockDesiredStateClient(mockCtrl)
	desiredStateClient.EXPECT().Domain().Return(testDomain).AnyTimes()
	desiredStateClient.EXPECT().PublishDesiredStateCommand(gomock.Any()).Return(fmt.Errorf("cannot publish desired state command request"))
	eventCallback := mocks.NewMockUpdateManagerCallback(mockCtrl)
	eventCallback.EXPECT().HandleDesiredStateFeedbackEvent(testDomain, testActivityID, "", types.StatusIncomplete, gomock.Any(), []*types.Action{})

	updateManager := createTestDomainUpdateManager(desiredStateClient, eventCallback)
	updateManager.updateOperation = &updateOperation{
		activityID: testActivityID,
	}

	updateManager.Command(context.Background(), testActivityID, command)
}

func TestCommandRequestNoOperation(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	desiredStateClient := mocks.NewMockDesiredStateClient(mockCtrl)
	desiredStateClient.EXPECT().Domain().Return(testDomain).AnyTimes()
	eventCallback := mocks.NewMockUpdateManagerCallback(mockCtrl)

	updateManager := createTestDomainUpdateManager(desiredStateClient, eventCallback)

	updateManager.Command(context.Background(), testActivityID, command)
}

func TestCommandRequestMismatchActivityId(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	desiredStateClient := mocks.NewMockDesiredStateClient(mockCtrl)
	desiredStateClient.EXPECT().Domain().Return(testDomain).AnyTimes()
	eventCallback := mocks.NewMockUpdateManagerCallback(mockCtrl)

	updateManager := createTestDomainUpdateManager(desiredStateClient, eventCallback)
	updateManager.updateOperation = &updateOperation{
		activityID: testActivityID,
	}

	updateManager.Command(context.Background(), "mistmatchActivityID", command)
}

func TestDesiredStateNotSent(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	desiredStateClient := mocks.NewMockDesiredStateClient(mockCtrl)
	desiredStateClient.EXPECT().Domain().Return(testDomain).AnyTimes()
	desiredStateClient.EXPECT().PublishDesiredState(gomock.Any()).Return(fmt.Errorf("error"))

	eventCallback := mocks.NewMockUpdateManagerCallback(mockCtrl)
	eventCallback.EXPECT().HandleDesiredStateFeedbackEvent(testDomain, testActivityID, "", types.StatusIncomplete, "error. cannot send desired state manifest to domain test-domain", []*types.Action{})

	updateManager := createTestDomainUpdateManager(desiredStateClient, eventCallback)
	updateManager.Apply(context.Background(), testActivityID, &types.DesiredState{
		Domains: []*types.Domain{
			{ID: testDomain},
		},
	})
}

func createTestDomainUpdateManager(desiredStateClient api.DesiredStateClient, eventCallback api.UpdateManagerCallback) *domainUpdateManager {
	return &domainUpdateManager{
		desiredStateLock:   sync.Mutex{},
		currentStateLock:   sync.Mutex{},
		desiredStateClient: desiredStateClient,
		eventCallback:      eventCallback,
		currentState:       &internalCurrentState{},
	}
}
