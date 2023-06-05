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
	"encoding/json"
	"fmt"
	"testing"

	"github.com/eclipse-kanto/update-manager/api/types"
	"github.com/eclipse-kanto/update-manager/test/mocks"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestHandleDesiredStateFeedbackEvent(t *testing.T) {
	mockCtr := gomock.NewController(t)
	defer mockCtr.Finish()

	mockClient := mocks.NewMockUpdateAgentClient(mockCtr)

	updAgent := &updateAgent{
		client: mockClient,
	}

	t.Run("test_actions_nil_publishDesiredState_no_err", func(t *testing.T) {
		mockClient.EXPECT().PublishDesiredStateFeedback(gomock.Any()).Return(nil)
		updAgent.HandleDesiredStateFeedbackEvent("", testActivityID, "", types.StatusCompleted, "operation completed", []*types.Action{})
	})

	t.Run("test_actions_nil_publishDesiredState_err", func(t *testing.T) {
		mockClient.EXPECT().PublishDesiredStateFeedback(gomock.Any()).DoAndReturn(func(bytes []byte) error {
			feedbackEnvelope := &types.Envelope{}
			err := json.Unmarshal(bytes, feedbackEnvelope)
			assert.NotNil(t, err)
			expectedFeedback := map[string]interface{}{"message": "operation completed", "status": "COMPLETED"}
			assert.Equal(t, testActivityID, feedbackEnvelope.ActivityID)
			assert.Equal(t, expectedFeedback, feedbackEnvelope.Payload)
			return nil
		})
		updAgent.HandleDesiredStateFeedbackEvent("", testActivityID, "", types.StatusCompleted, "operation completed", []*types.Action{})
	})

	t.Run("test_actions_not_nil_publishDesiredState_no_err", func(t *testing.T) {
		actions := []*types.Action{
			{
				Component: &types.Component{},
				Status:    types.ActionStatusUpdateSuccess,
				Message:   "update success",
			},
		}
		mockClient.EXPECT().PublishDesiredStateFeedback(gomock.Any()).DoAndReturn(func(bytes []byte) error {
			feedbackEnvelope := &types.Envelope{}
			err := json.Unmarshal(bytes, feedbackEnvelope)
			assert.NotNil(t, err)
			expectedFeedback := map[string]interface{}{"actions": []interface{}{map[string]interface{}{"component": map[string]interface{}{}, "message": "update success", "status": "UPDATE_SUCCESS"}}, "message": "operation completed", "status": "COMPLETED"}
			assert.Equal(t, testActivityID, feedbackEnvelope.ActivityID)
			assert.Equal(t, expectedFeedback, feedbackEnvelope.Payload)
			return nil
		})
		updAgent.HandleDesiredStateFeedbackEvent("", testActivityID, "", types.StatusCompleted, "operation completed", actions)
	})

	t.Run("test_actions_not_nil_publishDesiredState_err", func(t *testing.T) {
		actions := []*types.Action{
			{
				Component: &types.Component{},
				Status:    types.ActionStatusUpdateSuccess,
				Message:   "update success",
			},
		}
		mockClient.EXPECT().PublishDesiredStateFeedback(gomock.Any()).DoAndReturn(func(bytes []byte) error {
			feedbackEnvelope := &types.Envelope{}
			err := json.Unmarshal(bytes, feedbackEnvelope)
			assert.NotNil(t, err)
			expectedFeedback := map[string]interface{}{"actions": []interface{}{map[string]interface{}{"component": map[string]interface{}{}, "message": "update success", "status": "UPDATE_SUCCESS"}}, "message": "operation completed", "status": "COMPLETED"}
			assert.Equal(t, testActivityID, feedbackEnvelope.ActivityID)
			assert.Equal(t, expectedFeedback, feedbackEnvelope.Payload)
			return fmt.Errorf("cannot publish message")
		})
		updAgent.HandleDesiredStateFeedbackEvent("", testActivityID, "", types.StatusCompleted, "operation completed", actions)
	})
}
