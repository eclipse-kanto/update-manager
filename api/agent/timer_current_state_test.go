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
	"testing"
	"time"

	"github.com/eclipse-kanto/update-manager/test/mocks"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestNewCurrentStateNotifier(t *testing.T) {
	mockCtr := gomock.NewController(t)
	defer mockCtr.Finish()

	updAgent := &updateAgent{
		client: mocks.NewMockUpdateAgentClient(mockCtr),
	}
	expectedNotifier := &currentStateNotifier{
		interval: interval,
		agent:    updAgent,
	}
	assert.Equal(t, expectedNotifier, newCurrentStateNotifier(interval, updAgent))
}

func TestCurrentStateTimerSet(t *testing.T) {
	mockCtr := gomock.NewController(t)
	defer mockCtr.Finish()

	updAgent := &updateAgent{
		client: mocks.NewMockUpdateAgentClient(mockCtr),
	}

	t.Run("test_set_internal_timer_not_nil", func(t *testing.T) {
		notifier := initCurrentStateNotifier(updAgent)

		notifier.set(testActivityID, inventory)
		assert.Equal(t, inventory, notifier.currentState)
		stopCurrentStateNotifierInternalTimer(notifier)
	})

	t.Run("test_set_internal_timer_nil", func(t *testing.T) {
		notifier := newCurrentStateNotifier(interval, updAgent)

		notifier.set(testActivityID, inventory)
		assert.Equal(t, inventory, notifier.currentState)
		stopCurrentStateNotifierInternalTimer(notifier)
	})
}

func TestCurrentStateTimerStop(t *testing.T) {
	mockCtr := gomock.NewController(t)
	defer mockCtr.Finish()

	updAgent := &updateAgent{
		client: mocks.NewMockUpdateAgentClient(mockCtr),
	}
	notifier := initCurrentStateNotifier(updAgent)

	notifier.stop()
	assert.Nil(t, notifier.currentState)
}

func TestCurrentStateTimerNotifyEvent(t *testing.T) {
	mockCtr := gomock.NewController(t)
	defer mockCtr.Finish()

	mockClient := mocks.NewMockUpdateAgentClient(mockCtr)
	updAgent := &updateAgent{
		client: mockClient,
	}

	t.Run("test_notifyEvent_internal_timer_not_nil", func(t *testing.T) {
		notifier := initCurrentStateNotifier(updAgent)
		mockClient.EXPECT().SendCurrentState(gomock.Any(), gomock.Any()).Return(nil)

		notifier.notifyEvent()
		assert.Nil(t, notifier.internalTimer)
	})

	t.Run("test_notifyEvent_internal_timer_nil", func(t *testing.T) {
		notifier := newCurrentStateNotifier(interval, updAgent)
		notifier.notifyEvent()
		stopCurrentStateNotifierInternalTimer(notifier)
	})
}

func initCurrentStateNotifier(updAgent *updateAgent) *currentStateNotifier {
	notifier := newCurrentStateNotifier(interval, updAgent)
	notifier.activityID = testActivityID
	notifier.internalTimer = time.AfterFunc(interval, func() {})
	notifier.currentState = inventory
	return notifier
}

func stopCurrentStateNotifierInternalTimer(notifier *currentStateNotifier) {
	notifier.lock.Lock()
	defer notifier.lock.Unlock()
	if notifier.internalTimer != nil {
		notifier.internalTimer.Stop()
	}
}
