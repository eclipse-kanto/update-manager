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
	"errors"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"

	"github.com/eclipse-kanto/update-manager/api/types"
	"github.com/eclipse-kanto/update-manager/test/mocks"

	"github.com/golang/mock/gomock"
)

func TestHandleDesiredStateFeedbackEvent(t *testing.T) {
	mockCtr := gomock.NewController(t)
	defer mockCtr.Finish()

	tests := map[string]struct {
		status    types.StatusType
		message   string
		actions   []*types.Action
		sendError error
	}{
		"test_feedback_without_actions": {
			status:  types.StatusCompleted,
			message: "operation completed",
			actions: []*types.Action{},
		},
		"test_feedback_without_actions_send_error": {
			status:    types.StatusIdentifying,
			message:   "identifying...",
			actions:   []*types.Action{},
			sendError: errors.New("send error"),
		},
		"test_feedback_with_actions": {
			status:  types.StatusIdentified,
			message: "actions identified",
			actions: []*types.Action{
				{
					Component: &types.Component{ID: "mydomain:mycomponent", Version: "1.2.3"},
					Status:    types.ActionStatusIdentified,
					Message:   "actions identified",
				},
			},
		},
		"test_feedback_with_actions_send_error": {
			status:  types.StatusIncompleteInconsistent,
			message: "incomplete, inconsistent",
			actions: []*types.Action{
				{
					Component: &types.Component{ID: "mydomain:mycomponent", Version: "1.2.3"},
					Status:    types.ActionStatusUpdateFailure,
					Message:   "update failed",
					Progress:  50,
				},
			},
			sendError: errors.New("send error"),
		},
	}

	mockClient := mocks.NewMockUpdateAgentClient(mockCtr)

	updAgent := &updateAgent{
		client: mockClient,
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			mockClient.EXPECT().SendDesiredStateFeedback(testActivityID, &types.DesiredStateFeedback{
				Status:  test.status,
				Message: test.message,
				Actions: test.actions,
			}).Return(test.sendError)
			updAgent.HandleDesiredStateFeedbackEvent("", testActivityID, "", test.status, test.message, test.actions)
		})
	}

	t.Run("test_feedback_dsFeedbackNotifier_not_nil", func(t *testing.T) {
		actions := []*types.Action{
			{
				Component: &types.Component{},
				Status:    types.ActionStatusUpdateSuccess,
				Message:   "update success",
			},
		}
		updAgent := &updateAgent{
			client: mockClient,
		}
		mockClient.EXPECT().SendDesiredStateFeedback(gomock.Any(), gomock.Any()).DoAndReturn(func(activityID string, feedback *types.DesiredStateFeedback) error {
			expectedFeedback := &types.DesiredStateFeedback{
				Status:  types.StatusCompleted,
				Message: "operation completed",
				Actions: actions,
			}
			assert.Equal(t, testActivityID, activityID)
			assert.Equal(t, expectedFeedback, feedback)
			return nil
		})
		dsNotifier := &desiredStateFeedbackNotifier{
			internalTimer: time.AfterFunc(time.Millisecond, nil),
		}
		updAgent.desiredStateFeedbackNotifier = dsNotifier
		updAgent.HandleDesiredStateFeedbackEvent("", testActivityID, "", types.StatusCompleted, "operation completed", actions)

		assert.Nil(t, updAgent.desiredStateFeedbackNotifier.internalTimer)

		updAgent.desiredStateFeedbackNotifier = nil
	})

	t.Run("test_feedback_interval_invalid", func(t *testing.T) {
		actions := []*types.Action{
			{
				Component: &types.Component{},
				Status:    types.ActionStatusDownloading,
				Message:   "downloading",
			},
		}
		updAgent := &updateAgent{
			client: mockClient,
		}
		updAgent.desiredStateFeedbackReportInterval = -1 * time.Second
		updAgent.HandleDesiredStateFeedbackEvent("", testActivityID, "", types.StatusRunning, "operation running", actions)

		assert.Nil(t, updAgent.desiredStateFeedbackNotifier)
	})

	t.Run("test_feedback_actions_status_running_no_err", func(t *testing.T) {
		actions := []*types.Action{
			{
				Component: &types.Component{},
				Status:    types.ActionStatusDownloading,
				Message:   "downloading",
			},
		}
		updAgent := &updateAgent{
			client: mockClient,
		}
		mockClient.EXPECT().SendDesiredStateFeedback(gomock.Any(), gomock.Any()).DoAndReturn(func(activityID string, feedback *types.DesiredStateFeedback) error {
			expectedFeedback := &types.DesiredStateFeedback{
				Status:  types.StatusRunning,
				Message: "",
				Actions: actions,
			}
			assert.Equal(t, testActivityID, activityID)
			assert.Equal(t, expectedFeedback, feedback)
			return nil
		})
		updAgent.desiredStateFeedbackReportInterval = time.Millisecond
		updAgent.HandleDesiredStateFeedbackEvent("", testActivityID, "", types.StatusRunning, "operation running", actions)
		updAgent.desiredStateFeedbackNotifier.internalTimer.Stop()
	})

	t.Run("test_feedback_actions_status_running_multiple_events_timer_recreation_ok", func(t *testing.T) {
		actions := []*types.Action{
			{
				Component: &types.Component{},
				Status:    types.ActionStatusDownloading,
				Message:   "downloading",
			},
		}
		updAgent := &updateAgent{
			client: mockClient,
		}
		mockClient.EXPECT().SendDesiredStateFeedback(gomock.Any(), gomock.Any()).DoAndReturn(func(activityID string, feedback *types.DesiredStateFeedback) error {
			expectedFeedback := &types.DesiredStateFeedback{
				Status:  types.StatusRunning,
				Message: "",
				Actions: actions,
			}
			assert.Equal(t, testActivityID, activityID)
			assert.Equal(t, expectedFeedback, feedback)
			return nil
		})
		updAgent.desiredStateFeedbackReportInterval = time.Second
		updAgent.HandleDesiredStateFeedbackEvent("", testActivityID, "", types.StatusRunning, "operation running", actions)
		timer1 := updAgent.desiredStateFeedbackNotifier.internalTimer
		updAgent.HandleDesiredStateFeedbackEvent("", testActivityID, "", types.StatusRunning, "operation running", actions)
		timer2 := updAgent.desiredStateFeedbackNotifier.internalTimer
		assert.Equal(t, timer1, timer2)
		updAgent.desiredStateFeedbackNotifier.internalTimer.Stop()
	})

	t.Run("test_status_incomplete", func(t *testing.T) {
		actions := []*types.Action{
			{
				Component: &types.Component{},
				Status:    types.ActionStatusDownloading,
				Message:   "downloading",
			},
		}
		updAgent := &updateAgent{
			client: mockClient,
		}
		mockClient.EXPECT().SendDesiredStateFeedback(gomock.Any(), gomock.Any()).DoAndReturn(func(activityID string, feedback *types.DesiredStateFeedback) error {
			expectedFeedback := &types.DesiredStateFeedback{
				Status:  types.StatusIncomplete,
				Message: "downloading",
				Actions: actions,
			}
			assert.Equal(t, testActivityID, activityID)
			assert.Equal(t, expectedFeedback, feedback)
			return nil
		})
		updAgent.HandleDesiredStateFeedbackEvent("", testActivityID, "", types.StatusIncomplete, "downloading", actions)
	})

	t.Run("test_status_identifying", func(t *testing.T) {
		actions := []*types.Action{
			{
				Component: &types.Component{},
				Status:    types.ActionStatusActivating,
				Message:   "activating",
			},
		}
		updAgent := &updateAgent{
			client: mockClient,
		}
		mockClient.EXPECT().SendDesiredStateFeedback(gomock.Any(), gomock.Any()).DoAndReturn(func(activityID string, feedback *types.DesiredStateFeedback) error {
			expectedFeedback := &types.DesiredStateFeedback{
				Status:  types.StatusIdentifying,
				Message: "identifying",
				Actions: actions,
			}
			assert.Equal(t, testActivityID, activityID)
			assert.Equal(t, expectedFeedback, feedback)
			return nil
		})
		updAgent.HandleDesiredStateFeedbackEvent("", testActivityID, "", types.StatusIdentifying, "identifying", actions)
	})
}
