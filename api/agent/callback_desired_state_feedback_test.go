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
	"testing"

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
}
