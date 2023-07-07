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
	"github.com/eclipse-kanto/update-manager/api/types"
	"testing"
	"time"

	"github.com/eclipse-kanto/update-manager/test/mocks"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestNewCurrentStateNotifier(t *testing.T) {
	mockCtr := gomock.NewController(t)
	defer mockCtr.Finish()

	t.Run("test_new_current_state_notifier", func(t *testing.T) {
		mockClient := mocks.NewMockUpdateAgentClient(mockCtr)
		interval := time.Second
		updAgent := &updateAgent{
			client: mockClient,
		}
		expectedNotifier := &currentStateNotifier{
			interval: interval,
			agent:    updAgent,
		}
		actualNotifier := newCurrentStateNotifier(interval, updAgent)

		assert.Equal(t, expectedNotifier, actualNotifier)
	})
}

func TestCurrentStateTimerSet(t *testing.T) {
	mockCtr := gomock.NewController(t)
	defer mockCtr.Finish()

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

	t.Run("test_set_internal_timer_not_nil", func(t *testing.T) {
		t.Parallel()
		mockClient := mocks.NewMockUpdateAgentClient(mockCtr)
		interval := time.Second
		updAgent := &updateAgent{
			client: mockClient,
		}

		notifier := newCurrentStateNotifier(interval, updAgent)
		notifier.internalTimer = time.AfterFunc(interval, func() {})

		notifier.set(testActivityID, inventory)

		assert.Equal(t, inventory, notifier.currentState)
		if notifier.internalTimer != nil {
			if !notifier.internalTimer.Stop() {
				<-notifier.internalTimer.C
			}
		}
	})

	t.Run("test_set_internal_timer_nil", func(t *testing.T) {
		mockClient := mocks.NewMockUpdateAgentClient(mockCtr)
		interval := time.Second
		updAgent := &updateAgent{
			client: mockClient,
		}

		notifier := newCurrentStateNotifier(interval, updAgent)

		notifier.set(testActivityID, inventory)

		assert.Equal(t, inventory, notifier.currentState)
		if notifier.internalTimer != nil {
			if !notifier.internalTimer.Stop() {
				<-notifier.internalTimer.C
			}
		}
	})
}

func TestCurrentStateTimerStop(t *testing.T) {
	mockCtr := gomock.NewController(t)
	defer mockCtr.Finish()

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

	t.Run("test_stop", func(t *testing.T) {
		mockClient := mocks.NewMockUpdateAgentClient(mockCtr)
		interval := time.Second
		updAgent := &updateAgent{
			client: mockClient,
		}

		notifier := newCurrentStateNotifier(interval, updAgent)
		notifier.internalTimer = time.AfterFunc(interval, func() {})
		notifier.currentState = inventory

		notifier.stop()

		assert.Nil(t, notifier.currentState)
	})
}

func TestCurrentStateTimerNotifyEvent(t *testing.T) {
	mockCtr := gomock.NewController(t)
	defer mockCtr.Finish()

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

	t.Run("test_notifyEvent_internal_timer_not_nil", func(t *testing.T) {
		mockClient := mocks.NewMockUpdateAgentClient(mockCtr)
		interval := time.Second
		updAgent := &updateAgent{
			client: mockClient,
		}

		notifier := newCurrentStateNotifier(interval, updAgent)
		notifier.internalTimer = time.AfterFunc(interval, func() {})
		notifier.currentState = inventory
		mockClient.EXPECT().SendCurrentState(gomock.Any(), gomock.Any()).Return(nil)

		notifier.notifyEvent()

		assert.Nil(t, notifier.internalTimer)
	})

	t.Run("test_notifyEvent_internal_timer_nil", func(t *testing.T) {
		mockClient := mocks.NewMockUpdateAgentClient(mockCtr)
		interval := time.Second
		updAgent := &updateAgent{
			client: mockClient,
		}

		notifier := newCurrentStateNotifier(interval, updAgent)

		notifier.notifyEvent()

	})
}
