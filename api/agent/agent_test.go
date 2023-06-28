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
	"testing"

	"github.com/eclipse-kanto/update-manager/api/types"
	"github.com/eclipse-kanto/update-manager/test/mocks"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

const testActivityID = "testActivityId"

var dummyDesiredState = &types.DesiredState{
	Domains: []*types.Domain{
		{ID: "testDomain"},
	},
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

	ch := make(chan bool, 1)
	mockUpdateManager.EXPECT().Apply(context.Background(), testActivityID, dummyDesiredState).DoAndReturn(
		func(ctx context.Context, activityID string, state *types.DesiredState) {
			ch <- true
		})
	assert.NoError(t, updAgent.HandleDesiredState(testActivityID, 0, dummyDesiredState))
	<-ch
}
