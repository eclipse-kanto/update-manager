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
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/eclipse-kanto/update-manager/api/types"
	"github.com/eclipse-kanto/update-manager/test"
	"github.com/eclipse-kanto/update-manager/test/mocks"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

var dummyDesiredState = &types.DesiredState{
	Domains: []*types.Domain{
		{ID: "testDomain"},
	},
}

var dummyDesiredStateCommand = &types.DesiredStateCommand{
	Command: types.CommandUpdate,
}

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
	t.Run("test_start_no_error", func(t *testing.T) {
		mockUpdateManager.EXPECT().SetCallback(updAgent)
		mockClient.EXPECT().Start(gomock.Any()).Return(nil)

		returnErr := updAgent.Start(context.Background())

		assert.Equal(t, context.Background(), updAgent.ctx)
		assert.NoError(t, returnErr)
	})
	t.Run("test_start_with_error", func(t *testing.T) {
		mockUpdateManager.EXPECT().SetCallback(updAgent)
		mockClient.EXPECT().Start(gomock.Any()).Return(errors.New("start error"))

		returnErr := updAgent.Start(context.Background())

		assert.Equal(t, context.Background(), updAgent.ctx)
		assert.Error(t, returnErr)
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

	t.Run("test_dispose_no_error", func(t *testing.T) {
		mockUpdateManager.EXPECT().Dispose().Return(nil)
		mockClient.EXPECT().Stop()

		assert.NoError(t, updAgent.Stop())
	})
	t.Run("test_dispose_with_error", func(t *testing.T) {
		mockUpdateManager.EXPECT().Dispose().Return(errors.New("dispose error"))
		assert.Error(t, updAgent.Stop())
		assert.Nil(t, updAgent.currentStateNotifier)
	})
	t.Run("test_stop_with_error", func(t *testing.T) {
		mockUpdateManager.EXPECT().Dispose().Return(nil)
		mockClient.EXPECT().Stop().Return(errors.New("stop error"))
		assert.Error(t, updAgent.Stop())
		assert.Nil(t, updAgent.currentStateNotifier)
	})
	t.Run("test_dispose_notifiers_not_nil", func(t *testing.T) {
		updAgent.desiredStateFeedbackNotifier = &desiredStateFeedbackNotifier{
			internalTimer: time.AfterFunc(time.Millisecond, nil),
		}
		updAgent.currentStateNotifier = &currentStateNotifier{
			internalTimer: time.AfterFunc(time.Millisecond, nil),
		}

		mockUpdateManager.EXPECT().Dispose().Return(nil)
		mockClient.EXPECT().Stop()

		returnErr := updAgent.Stop()

		assert.Equal(t, nil, returnErr)
		assert.NotNil(t, updAgent.currentStateNotifier)
		assert.NotNil(t, updAgent.desiredStateFeedbackNotifier)
	})
}

func TestHandleDesiredState(t *testing.T) {
	mockCtr := gomock.NewController(t)
	defer mockCtr.Finish()

	mockUpdateManager := mocks.NewMockUpdateManager(mockCtr)
	updAgent := &updateAgent{
		client:  mocks.NewMockUpdateAgentClient(mockCtr),
		manager: mockUpdateManager,
		ctx:     context.Background(),
	}

	ch := make(chan bool, 1)
	mockUpdateManager.EXPECT().Apply(context.Background(), test.ActivityID, dummyDesiredState).DoAndReturn(
		func(ctx context.Context, activityID string, state *types.DesiredState) {
			ch <- true
		})
	assert.NoError(t, updAgent.HandleDesiredState(test.ActivityID, 0, dummyDesiredState))
	<-ch
}

func TestHandleDesiredStateCommand(t *testing.T) {
	mockCtr := gomock.NewController(t)
	defer mockCtr.Finish()

	mockUpdateManager := mocks.NewMockUpdateManager(mockCtr)
	updAgent := &updateAgent{
		client:  mocks.NewMockUpdateAgentClient(mockCtr),
		manager: mockUpdateManager,
		ctx:     context.Background(),
	}

	dsCommand := &types.DesiredStateCommand{
		Command: types.CommandUpdate,
	}
	ch := make(chan bool, 1)
	mockUpdateManager.EXPECT().Command(context.Background(), test.ActivityID, dummyDesiredStateCommand).DoAndReturn(func(ctx context.Context, activityID string, command *types.DesiredStateCommand) {
		ch <- true
		assert.True(t, reflect.DeepEqual(dsCommand, command))
	})
	assert.NoError(t, updAgent.HandleDesiredStateCommand(test.ActivityID, 0, dummyDesiredStateCommand))
	<-ch

}

func TestHandleCurrentStateGet(t *testing.T) {
	mockCtr := gomock.NewController(t)
	defer mockCtr.Finish()

	mockClient := mocks.NewMockUpdateAgentClient(mockCtr)
	mockUpdateManager := mocks.NewMockUpdateManager(mockCtr)

	updAgent := &updateAgent{
		client:  mockClient,
		manager: mockUpdateManager,
		ctx:     context.Background(),
	}

	activityID := prefixInitCurrentStateID + test.ActivityID

	t.Run("test_handle_current_state_get_error", func(t *testing.T) {
		mockUpdateManager.EXPECT().Get(context.Background(), test.ActivityID).Return(nil, errors.New("get error"))
		err := updAgent.HandleCurrentStateGet(test.ActivityID, 0)
		assert.NotNil(t, err)
	})

	t.Run("test_handle_current_state_get_ok", func(t *testing.T) {
		mockUpdateManager.EXPECT().WatchEvents(context.Background()).Times(1)
		mockUpdateManager.EXPECT().Get(context.Background(), activityID).Return(test.Inventory, nil)
		mockClient.EXPECT().SendCurrentState(activityID, test.Inventory).Times(1).Return(nil)

		err := updAgent.HandleCurrentStateGet(activityID, 0)

		assert.Nil(t, err)
	})

	t.Run("test_handle_current_state_get_publish_err", func(t *testing.T) {
		mockUpdateManager.EXPECT().WatchEvents(context.Background()).Times(1)
		mockUpdateManager.EXPECT().Get(context.Background(), activityID).Return(test.Inventory, nil)
		mockClient.EXPECT().SendCurrentState(activityID, test.Inventory).Times(1).Return(errors.New("send current state error"))
		err := updAgent.HandleCurrentStateGet(activityID, 0)

		assert.Error(t, err)
	})

}
