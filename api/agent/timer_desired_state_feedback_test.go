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

	"github.com/eclipse-kanto/update-manager/api/types"
	"github.com/eclipse-kanto/update-manager/test"
	"github.com/eclipse-kanto/update-manager/test/mocks"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

var (
	testActions = []*types.Action{
		{
			Component: &types.Component{
				ID:      "action 1 component",
				Version: "1.0.0",
			},
		},
	}
	reportedActions = []*types.Action{
		{
			Component: &types.Component{
				ID:      "action 1 component",
				Version: "1.0.0",
			},
		},
	}
)

func TestNewDesiredStateFeedbackNotifier(t *testing.T) {
	mockCtr := gomock.NewController(t)
	defer mockCtr.Finish()

	updAgent := &updateAgent{
		client: mocks.NewMockUpdateAgentClient(mockCtr),
	}
	expectedNotifier := &desiredStateFeedbackNotifier{
		interval:        test.Interval,
		agent:           updAgent,
		actions:         []*types.Action{},
		reportedActions: []*types.Action{},
	}
	assert.Equal(t, expectedNotifier, newDesiredStateFeedbackNotifier(test.Interval, updAgent))

}

func TestDesiredStateFeedbackTimerSet(t *testing.T) {
	mockCtr := gomock.NewController(t)
	defer mockCtr.Finish()

	testActions2 := []*types.Action{
		{
			Component: &types.Component{
				ID:      "action2",
				Version: "2.0.0",
			},
		},
	}

	reportedActions2 := []*types.Action{
		{
			Component: &types.Component{
				ID:      "action2",
				Version: "2.0.0",
			},
		},
	}
	mockClient := mocks.NewMockUpdateAgentClient(mockCtr)
	updAgent := &updateAgent{
		client: mockClient,
	}

	t.Run("test_set_internal_timer_not_nil", func(t *testing.T) {
		notifier := initDesiredStateFeedbackNotifier(updAgent)
		notifier.set(test.ActivityID, testActions)
		assert.Equal(t, testActions, notifier.actions)
		stopDesiredStateFeedbackNotifierInternalTimer(notifier)
	})

	t.Run("test_set_internal_timer_nil", func(t *testing.T) {
		notifier := newDesiredStateFeedbackNotifier(test.Interval, updAgent)
		mockClient.EXPECT().SendDesiredStateFeedback(gomock.Any(), gomock.Any()).Times(1)

		notifier.set(test.ActivityID, testActions)
		assert.Equal(t, reportedActions, notifier.reportedActions)
		assert.Equal(t, test.ActivityID, notifier.activityID)
		stopDesiredStateFeedbackNotifierInternalTimer(notifier)
	})

	t.Run("test_actions_change_during_timeout", func(t *testing.T) {
		notifier := newDesiredStateFeedbackNotifier(test.Interval, updAgent)
		mockClient.EXPECT().SendDesiredStateFeedback(gomock.Any(), gomock.Any()).Times(1)
		// initial actions set
		notifier.set(test.ActivityID, testActions)
		notifier.lock.Lock()
		assert.Equal(t, testActions, notifier.actions)
		assert.Equal(t, reportedActions, notifier.reportedActions)
		assert.Equal(t, test.ActivityID, notifier.activityID)
		notifier.lock.Unlock()
		mockClient.EXPECT().SendDesiredStateFeedback(gomock.Any(), gomock.Any()).Times(1)
		// update the actions with a new set
		notifier.set(test.ActivityID, testActions2)
		notifier.lock.Lock()
		// assert that the actions have not updated before the interval has passed
		assert.Equal(t, testActions2, notifier.actions)
		assert.NotEqual(t, reportedActions2, notifier.reportedActions)
		assert.Equal(t, test.ActivityID, notifier.activityID)
		notifier.lock.Unlock()
		time.Sleep(test.Interval + 100*time.Millisecond)
		notifier.lock.Lock()
		// check that the actions have been properly updated after the timeout
		assert.Equal(t, testActions2, notifier.actions)
		assert.Equal(t, reportedActions2, notifier.reportedActions)
		assert.Equal(t, test.ActivityID, notifier.activityID)
		notifier.lock.Unlock()
		stopDesiredStateFeedbackNotifierInternalTimer(notifier)
	})
	t.Run("test_no_event_published_on_resetting_the_same_actions_during_timeout", func(t *testing.T) {
		notifier := newDesiredStateFeedbackNotifier(2*test.Interval, updAgent)
		mockClient.EXPECT().SendDesiredStateFeedback(gomock.Any(), gomock.Any()).Times(1)
		notifier.set(test.ActivityID, testActions)
		notifier.lock.Lock()
		assert.Equal(t, testActions, notifier.actions)
		assert.Equal(t, reportedActions, notifier.reportedActions)
		assert.Equal(t, test.ActivityID, notifier.activityID)
		notifier.lock.Unlock()
		mockClient.EXPECT().SendDesiredStateFeedback(gomock.Any(), gomock.Any()).Times(0)
		notifier.set(test.ActivityID, testActions)
		time.Sleep(test.Interval + 100*time.Millisecond)
		notifier.lock.Lock()
		assert.Equal(t, testActions, notifier.actions)
		assert.Equal(t, reportedActions, notifier.reportedActions)
		assert.Equal(t, test.ActivityID, notifier.activityID)
		notifier.lock.Unlock()
		stopDesiredStateFeedbackNotifierInternalTimer(notifier)
	})
}

func TestDesiredStateFeedbackTimerStop(t *testing.T) {
	mockCtr := gomock.NewController(t)
	defer mockCtr.Finish()

	updAgent := &updateAgent{
		client: mocks.NewMockUpdateAgentClient(mockCtr),
	}

	notifier := initDesiredStateFeedbackNotifier(updAgent)
	notifier.reportedActions = reportedActions

	notifier.stop()
	assert.Equal(t, "", notifier.activityID)
	assert.Equal(t, []*types.Action{}, notifier.actions)
	assert.Equal(t, []*types.Action{}, notifier.reportedActions)
	assert.Nil(t, notifier.internalTimer)
}

func TestDesiredStateFeedbackTimerNotifyEvent(t *testing.T) {
	mockCtr := gomock.NewController(t)
	defer mockCtr.Finish()
	mockClient := mocks.NewMockUpdateAgentClient(mockCtr)
	updAgent := &updateAgent{
		client: mockClient,
	}
	t.Run("test_notifyEvent_internal_timer_nil", func(t *testing.T) {
		notifier := newDesiredStateFeedbackNotifier(test.Interval, updAgent)
		notifier.notifyEvent()
	})

	t.Run("test_notifyEvent_internal_timer_not_nil_actions_equal", func(t *testing.T) {
		notifier := initDesiredStateFeedbackNotifier(updAgent)
		notifier.reportedActions = reportedActions
		notifier.notifyEvent()
		assert.Nil(t, notifier.internalTimer)
	})

	t.Run("test_notifyEvent_internal_timer_not_nil_actions_not_equal", func(t *testing.T) {
		notifier := initDesiredStateFeedbackNotifier(updAgent)
		mockClient.EXPECT().SendDesiredStateFeedback(gomock.Any(), gomock.Any())

		notifier.notifyEvent()
		assert.Nil(t, notifier.internalTimer)
		assert.Equal(t, reportedActions, notifier.reportedActions)
	})
}

func TestDesiredStateFeedbackTimerToActionsString(t *testing.T) {
	mockCtr := gomock.NewController(t)
	defer mockCtr.Finish()

	testActions := []*types.Action{
		{
			Component: &types.Component{
				ID:      "action 1 component",
				Version: "1.0.0",
			},
			Status:  types.ActionStatusUpdateSuccess,
			Message: "update success",
		},
	}

	expected := "[ {Component:{ID:action 1 component Version:1.0.0} Status:UPDATE_SUCCESS Progress:0 Message:update success} ]"
	assert.Equal(t, expected, toActionsString(testActions))

}

func initDesiredStateFeedbackNotifier(updAgent *updateAgent) *desiredStateFeedbackNotifier {
	notifier := newDesiredStateFeedbackNotifier(test.Interval, updAgent)
	notifier.activityID = test.ActivityID
	notifier.internalTimer = time.AfterFunc(test.Interval, func() {})
	notifier.actions = testActions
	return notifier
}

func stopDesiredStateFeedbackNotifierInternalTimer(notifier *desiredStateFeedbackNotifier) {
	notifier.lock.Lock()
	defer notifier.lock.Unlock()

	if notifier.internalTimer != nil {
		notifier.internalTimer.Stop()
	}
}
