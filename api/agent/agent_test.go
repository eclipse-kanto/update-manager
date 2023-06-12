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

package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/eclipse-kanto/update-manager/api/types"
	"github.com/eclipse-kanto/update-manager/test/mocks"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

const (
	testActivityID = "testActivityId"

	dummyDesiredStateJSON = `
	{
		"activityId": "testActivityId",
		"payload": {
			"domains": [
				{
					"id": "testDomain"
				}
			]
		}
	 }
	`
)

func TestNewUpdateAgent(t *testing.T) {
	mockCtr := gomock.NewController(t)
	defer mockCtr.Finish()

	mockClient := mocks.NewMockUpdateAgentClient(mockCtr)
	mockUpdateManager := mocks.NewMockUpdateManager(mockCtr)

	assert.Equal(t, &updateAgent{
		client:  mockClient,
		manager: mockUpdateManager,
	}, NewUpdateAgent(mockClient, mockUpdateManager))
}

func TestStart(t *testing.T) {
	mockCtr := gomock.NewController(t)
	defer mockCtr.Finish()

	mockClient := mocks.NewMockUpdateAgentClient(mockCtr)
	mockUpdateManager := mocks.NewMockUpdateManager(mockCtr)

	updAgent := &updateAgent{
		client:  mockClient,
		manager: mockUpdateManager,
		ctx:     nil,
	}
	t.Run("test_err_nil", func(t *testing.T) {
		mockUpdateManager.EXPECT().SetCallback(updAgent)
		mockClient.EXPECT().Connect(gomock.Any()).Return(nil)

		returnErr := updAgent.Start(context.Background())

		assert.Equal(t, context.Background(), updAgent.ctx)
		assert.Equal(t, nil, returnErr)
	})
	t.Run("test_err_not_nil", func(t *testing.T) {
		mockUpdateManager.EXPECT().SetCallback(updAgent)
		mockClient.EXPECT().Connect(gomock.Any()).Return(fmt.Errorf("errNotNil"))

		returnErr := updAgent.Start(context.Background())

		assert.Equal(t, context.Background(), updAgent.ctx)
		assert.Equal(t, fmt.Errorf("errNotNil"), returnErr)
	})
}

func TestStop(t *testing.T) {
	mockCtr := gomock.NewController(t)
	defer mockCtr.Finish()

	mockClient := mocks.NewMockUpdateAgentClient(mockCtr)
	mockUpdateManager := mocks.NewMockUpdateManager(mockCtr)

	updAgent := &updateAgent{
		client:  mockClient,
		manager: mockUpdateManager,
	}

	t.Run("test_err_nil_currentStateNotifier_nil", func(t *testing.T) {
		updAgent.currentStateNotifier = nil

		mockUpdateManager.EXPECT().Dispose().Return(nil)
		mockClient.EXPECT().Disconnect()

		assert.Nil(t, updAgent.Stop())
		assert.Nil(t, updAgent.currentStateNotifier)
	})
	t.Run("test_err_not_nil_currentStateNotifier_nil", func(t *testing.T) {
		updAgent.currentStateNotifier = nil

		mockUpdateManager.EXPECT().Dispose().Return(fmt.Errorf("errNotNil"))

		assert.Equal(t, fmt.Errorf("errNotNil"), updAgent.Stop())
		assert.Nil(t, updAgent.currentStateNotifier)
	})
}

func TestGetCurrentState(t *testing.T) {
	mockCtr := gomock.NewController(t)
	defer mockCtr.Finish()

	mockUpdateManager := mocks.NewMockUpdateManager(mockCtr)

	updAgent := &updateAgent{
		manager: mockUpdateManager,
		ctx:     context.Background(),
	}

	inventory := &types.Inventory{
		SoftwareNodes: []*types.SoftwareNode{
			{
				InventoryNode: types.InventoryNode{
					ID:      "update-manager",
					Version: "development",
					Name:    "Update Manager",
				},
				Type: "APPLICATION",
			},
		},
	}

	t.Run("test_no_activity_id_get_err_nil", func(t *testing.T) {
		mockUpdateManager.EXPECT().Get(context.Background(), "").Return(inventory, nil)
		currentStateBytes, err := updAgent.GetCurrentState(context.Background(), "")
		assert.Nil(t, err)

		expectedPayload := map[string]interface{}{"softwareNodes": []interface{}{map[string]interface{}{"id": "update-manager", "name": "Update Manager", "type": "APPLICATION", "version": "development"}}}
		inventoryEnvelope := &types.Envelope{}
		assert.Nil(t, json.Unmarshal(currentStateBytes, inventoryEnvelope))
		assert.Equal(t, expectedPayload, inventoryEnvelope.Payload)
	})

	t.Run("test_no_activity_id_get_err_not_nil", func(t *testing.T) {
		mockUpdateManager.EXPECT().Get(context.Background(), "").Return(nil, fmt.Errorf("cannot get current state"))
		_, err := updAgent.GetCurrentState(context.Background(), "")
		assert.NotNil(t, err)
	})

	t.Run("test_activity_id_not_empty_get_err_nil", func(t *testing.T) {
		mockUpdateManager.EXPECT().Get(context.Background(), testActivityID).Return(inventory, nil)

		currentStateBytes, err := updAgent.GetCurrentState(context.Background(), testActivityID)
		assert.Nil(t, err)

		expectedPayload := map[string]interface{}{"softwareNodes": []interface{}{map[string]interface{}{"id": "update-manager", "name": "Update Manager", "type": "APPLICATION", "version": "development"}}}
		inventoryEnvelope := &types.Envelope{}
		assert.Nil(t, json.Unmarshal(currentStateBytes, inventoryEnvelope))
		assert.Equal(t, expectedPayload, inventoryEnvelope.Payload)
	})

	t.Run("test_activity_id_not_empty_get_err_not_nil", func(t *testing.T) {
		mockUpdateManager.EXPECT().Get(context.Background(), testActivityID).Return(nil, fmt.Errorf("cannot get current state"))
		_, err := updAgent.GetCurrentState(context.Background(), testActivityID)
		assert.NotNil(t, err)
	})
}

func TestHandleDesiredState(t *testing.T) {
	mockCtr := gomock.NewController(t)
	defer mockCtr.Finish()

	mockClient := mocks.NewMockUpdateAgentClient(mockCtr)
	mockUpdateManager := mocks.NewMockUpdateManager(mockCtr)

	updAgent := &updateAgent{
		client:  mockClient,
		manager: mockUpdateManager,
		ctx:     context.Background(),
	}

	t.Run("test_invalid_desired_state", func(t *testing.T) {
		assert.NotNil(t, updAgent.HandleDesiredState([]byte("invalid-desired-state")))
	})

	t.Run("test_correct_desired_state", func(t *testing.T) {
		desiredState := &types.DesiredState{
			Domains: []*types.Domain{
				{
					ID: "testDomain",
				},
			},
		}
		ch := make(chan bool, 1)
		mockUpdateManager.EXPECT().Apply(context.Background(), testActivityID, gomock.Any()).DoAndReturn(func(ctx context.Context, activityID string, state *types.DesiredState) {
			ch <- true
			assert.Equal(t, desiredState, state)
		})
		assert.Nil(t, updAgent.HandleDesiredState([]byte(dummyDesiredStateJSON)))
		<-ch
	})
}
