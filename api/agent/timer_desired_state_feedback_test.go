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

func TestNewDesiredStateFeedbackNotifier(t *testing.T) {
	mockCtr := gomock.NewController(t)
	defer mockCtr.Finish()

	actions := []*types.Action{}
	reportedActions := []*types.Action{}

	t.Run("test_new_desired_state_feedback_notifier", func(t *testing.T) {
		mockClient := mocks.NewMockUpdateAgentClient(mockCtr)
		interval := time.Second
		updAgent := &updateAgent{
			client: mockClient,
		}
		expectedNotifier := &desiredStateFeedbackNotifier{
			interval:        interval,
			agent:           updAgent,
			actions:         actions,
			reportedActions: reportedActions,
		}
		actualNotifier := newDesiredStateFeedbackNotifier(interval, updAgent)

		assert.Equal(t, expectedNotifier, actualNotifier)
	})
}

func TestDesiredStateFeedbackTimerSet(t *testing.T) {
	mockCtr := gomock.NewController(t)
	defer mockCtr.Finish()

	actions := []*types.Action{
		{
			Component: &types.Component{
				ID:      "action 1 component",
				Version: "1.0.0",
			},
		},
	}

	actions2 := []*types.Action{
		{
			Component: &types.Component{
				ID:      "action2",
				Version: "2.0.0",
			},
		},
	}

	reportedActions := []*types.Action{
		{
			Component: &types.Component{
				ID:      "action 1 component",
				Version: "1.0.0",
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

	t.Run("test_set_internal_timer_not_nil", func(t *testing.T) {
		mockClient := mocks.NewMockUpdateAgentClient(mockCtr)
		interval := time.Second
		updAgent := &updateAgent{
			client: mockClient,
		}

		notifier := newDesiredStateFeedbackNotifier(interval, updAgent)
		notifier.internalTimer = time.AfterFunc(interval, func() {})

		notifier.set(testActivityID, actions)

		assert.Equal(t, actions, notifier.actions)
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

		notifier := newDesiredStateFeedbackNotifier(interval, updAgent)
		mockClient.EXPECT().SendDesiredStateFeedback(gomock.Any(), gomock.Any()).Times(1)
		notifier.set(testActivityID, actions)

		assert.Equal(t, reportedActions, notifier.reportedActions)
		assert.Equal(t, testActivityID, notifier.activityID)
		if notifier.internalTimer != nil {
			if !notifier.internalTimer.Stop() {
				<-notifier.internalTimer.C
			}
		}
	})

	t.Run("test_actions_change_during_timeout", func(t *testing.T) {
		mockClient := mocks.NewMockUpdateAgentClient(mockCtr)
		interval := 2 * time.Second
		updAgent := &updateAgent{
			client: mockClient,
		}
		notifier := newDesiredStateFeedbackNotifier(interval, updAgent)
		mockClient.EXPECT().SendDesiredStateFeedback(gomock.Any(), gomock.Any()).Times(1)
		notifier.set(testActivityID, actions)
		assert.Equal(t, actions, notifier.actions)
		assert.Equal(t, reportedActions, notifier.reportedActions)
		assert.Equal(t, testActivityID, notifier.activityID)
		time.Sleep(1 * time.Second)
		mockClient.EXPECT().SendDesiredStateFeedback(gomock.Any(), gomock.Any()).Times(1)
		notifier.set(testActivityID, actions2)
		assert.Equal(t, actions2, notifier.actions)
		assert.NotEqual(t, reportedActions2, notifier.reportedActions)
		assert.Equal(t, testActivityID, notifier.activityID)
		time.Sleep(2 * time.Second)
		assert.Equal(t, actions2, notifier.actions)
		assert.Equal(t, reportedActions2, notifier.reportedActions)
		assert.Equal(t, testActivityID, notifier.activityID)
		if notifier.internalTimer != nil {
			if !notifier.internalTimer.Stop() {
				<-notifier.internalTimer.C
			}
		}
	})
	t.Run("test_no_event_published_on_resetting_the_same_actions_during_timeout", func(t *testing.T) {
		mockClient := mocks.NewMockUpdateAgentClient(mockCtr)
		interval := 2 * time.Second
		updAgent := &updateAgent{
			client: mockClient,
		}
		notifier := newDesiredStateFeedbackNotifier(interval, updAgent)
		mockClient.EXPECT().SendDesiredStateFeedback(gomock.Any(), gomock.Any()).Times(1)
		notifier.set(testActivityID, actions)
		assert.Equal(t, actions, notifier.actions)
		assert.Equal(t, reportedActions, notifier.reportedActions)
		assert.Equal(t, testActivityID, notifier.activityID)
		time.Sleep(1 * time.Second)
		mockClient.EXPECT().SendDesiredStateFeedback(gomock.Any(), gomock.Any()).Times(0)
		notifier.set(testActivityID, actions)
		time.Sleep(2 * time.Second)
		assert.Equal(t, actions, notifier.actions)
		assert.Equal(t, reportedActions, notifier.reportedActions)
		assert.Equal(t, testActivityID, notifier.activityID)
		if notifier.internalTimer != nil {
			notifier.internalTimer.Stop()
		}
	})
}

func TestDesiredStateFeedbackTimerStop(t *testing.T) {
	mockCtr := gomock.NewController(t)
	defer mockCtr.Finish()

	t.Run("test_stop", func(t *testing.T) {
		mockClient := mocks.NewMockUpdateAgentClient(mockCtr)
		interval := time.Second
		updAgent := &updateAgent{
			client: mockClient,
		}

		actions := []*types.Action{
			{
				Component: &types.Component{
					ID:      "action 1 component",
					Version: "1.0.0",
				},
				Status:  types.ActionStatusUpdateSuccess,
				Message: "update success",
			},
		}
		reportedActions := []*types.Action{
			{
				Component: &types.Component{
					ID:      "action 1 component",
					Version: "1.0.0",
				},
				Status:  types.ActionStatusUpdateSuccess,
				Message: "update success",
			},
		}
		notifier := newDesiredStateFeedbackNotifier(interval, updAgent)
		notifier.actions = actions
		notifier.reportedActions = reportedActions
		notifier.internalTimer = time.AfterFunc(interval, func() {})

		notifier.stop()

		assert.Equal(t, "", notifier.activityID)
		assert.Equal(t, []*types.Action{}, notifier.actions)
		assert.Equal(t, []*types.Action{}, notifier.reportedActions)
		assert.Nil(t, notifier.internalTimer)
	})
}

func TestDesiredStateFeedbackTimerNotifyEvent(t *testing.T) {
	mockCtr := gomock.NewController(t)
	defer mockCtr.Finish()

	actions := []*types.Action{
		{
			Component: &types.Component{
				ID:      "action 1 component",
				Version: "1.0.0",
			},
			Status:  types.ActionStatusUpdateSuccess,
			Message: "update success",
		},
	}

	reportedActions := []*types.Action{
		{
			Component: &types.Component{
				ID:      "action 1 component",
				Version: "1.0.0",
			},
			Status:  types.ActionStatusUpdateSuccess,
			Message: "update success",
		},
	}

	t.Run("test_notifyEvent_internal_timer_nil", func(t *testing.T) {
		mockClient := mocks.NewMockUpdateAgentClient(mockCtr)
		interval := time.Second
		updAgent := &updateAgent{
			client: mockClient,
		}

		notifier := newDesiredStateFeedbackNotifier(interval, updAgent)

		notifier.notifyEvent()
	})

	t.Run("test_notifyEvent_internal_timer_not_nil_actions_equal", func(t *testing.T) {
		mockClient := mocks.NewMockUpdateAgentClient(mockCtr)
		interval := time.Second
		updAgent := &updateAgent{
			client: mockClient,
		}

		notifier := newDesiredStateFeedbackNotifier(interval, updAgent)
		notifier.internalTimer = time.AfterFunc(interval, func() {})
		notifier.actions = actions
		notifier.reportedActions = reportedActions

		notifier.notifyEvent()

		assert.Nil(t, notifier.internalTimer)
	})

	t.Run("test_notifyEvent_internal_timer_not_nil_actions_not_equal", func(t *testing.T) {
		mockClient := mocks.NewMockUpdateAgentClient(mockCtr)
		interval := time.Second
		updAgent := &updateAgent{
			client: mockClient,
		}

		notifier := newDesiredStateFeedbackNotifier(interval, updAgent)
		notifier.internalTimer = time.AfterFunc(interval, func() {})
		notifier.actions = actions
		notifier.activityID = testActivityID

		mockClient.EXPECT().SendDesiredStateFeedback(gomock.Any(), gomock.Any())
		notifier.notifyEvent()

		assert.Nil(t, notifier.internalTimer)
		assert.Equal(t, reportedActions, notifier.reportedActions)
	})
}

func TestDesiredStateFeedbackTimerToActionsString(t *testing.T) {
	mockCtr := gomock.NewController(t)
	defer mockCtr.Finish()

	actions := []*types.Action{
		{
			Component: &types.Component{
				ID:      "action 1 component",
				Version: "1.0.0",
			},
			Status:  types.ActionStatusUpdateSuccess,
			Message: "update success",
		},
	}

	t.Run("test_to_actions_string", func(t *testing.T) {
		expected := "[ {Component:{ID:action 1 component Version:1.0.0} Status:UPDATE_SUCCESS Progress:0 Message:update success} ]"

		assert.Equal(t, expected, toActionsString(actions))
	})
}
