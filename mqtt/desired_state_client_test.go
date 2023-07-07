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

package mqtt

import (
	"errors"
	"strings"
	"testing"

	"github.com/eclipse-kanto/update-manager/api/types"
	mqttmocks "github.com/eclipse-kanto/update-manager/mqtt/mocks"
	"github.com/eclipse-kanto/update-manager/test/mocks"

	pahomqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestDomain(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockPaho := mqttmocks.NewMockClient(mockCtrl)

	updateAgentClient := &updateAgentClient{
		mqttClient: newInternalClient("testDomain", &ConnectionConfig{}, mockPaho),
	}

	assert.Equal(t, "testDomain", NewDesiredStateClient("testDomain", updateAgentClient).Domain())
}

func TestDesiredStateClientStart(t *testing.T) {
	tests := map[string]testCaseOutgoing{
		"test_subscribe_ok":      {domain: "testdomain", isTimedOut: false},
		"test_subscribe_timeout": {domain: "mydomain", isTimedOut: true},
	}

	mockCtrl, mockPaho, mockToken := setupCommonMocks(t)
	defer mockCtrl.Finish()

	mockStateHandler := mocks.NewMockStateHandler(mockCtrl)

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			desiredStateClient := &desiredStateClient{
				mqttClient: newInternalClient(test.domain, mqttTestConfig, mockPaho),
				domain:     test.domain,
			}
			mockPaho.EXPECT().SubscribeMultiple(map[string]byte{
				test.domain + "update/currentstate":         1,
				test.domain + "update/desiredstatefeedback": 1},
				gomock.Any()).Return(mockToken)
			setupMockToken(mockToken, mqttTestConfig.SubscribeTimeout, test.isTimedOut)

			assertOutgoingResult(t, test.isTimedOut, desiredStateClient.Start(mockStateHandler))
			if test.isTimedOut {
				assert.Nil(t, desiredStateClient.stateHandler)
			} else {
				assert.Equal(t, mockStateHandler, desiredStateClient.stateHandler)
			}
		})
	}
}

func TestDesiredStateClientStop(t *testing.T) {
	tests := map[string]testCaseOutgoing{
		"test_unsubscribe_ok":      {domain: "testdomain", isTimedOut: false},
		"test_unsubscribe_timeout": {domain: "mydomain", isTimedOut: true},
	}

	mockCtrl, mockPaho, mockToken := setupCommonMocks(t)
	defer mockCtrl.Finish()

	mockStateHandler := mocks.NewMockStateHandler(mockCtrl)

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			desiredStateClient := &desiredStateClient{
				mqttClient:   newInternalClient(test.domain, mqttTestConfig, mockPaho),
				domain:       test.domain,
				stateHandler: mockStateHandler,
			}
			mockPaho.EXPECT().Unsubscribe(test.domain+"update/currentstate", test.domain+"update/desiredstatefeedback").Return(mockToken)
			setupMockToken(mockToken, mqttTestConfig.UnsubscribeTimeout, test.isTimedOut)

			assertOutgoingResult(t, test.isTimedOut, desiredStateClient.Stop())
			if test.isTimedOut {
				assert.Equal(t, mockStateHandler, desiredStateClient.stateHandler)
			} else {
				assert.Nil(t, desiredStateClient.stateHandler)
			}
		})
	}
}

func TestSendDesiredState(t *testing.T) {
	tests := map[string]testCaseOutgoing{
		"test_send_desired_state_ok":    {domain: "testdomain", isTimedOut: false},
		"test_send_desired_state_error": {domain: "mydomain", isTimedOut: true},
	}

	mockCtrl, mockPaho, mockToken := setupCommonMocks(t)
	defer mockCtrl.Finish()

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			testDesiredState := &types.DesiredState{
				Domains: []*types.Domain{
					{ID: test.domain},
				},
			}
			desiredStateClient := NewDesiredStateClient(test.domain, &updateAgentClient{
				mqttClient: newInternalClient("testDomain", mqttTestConfig, mockPaho),
			})
			mockPaho.EXPECT().Publish(test.domain+"update/desiredstate", uint8(1), false, gomock.Any()).DoAndReturn(
				func(topic string, qos byte, retained bool, payload interface{}) pahomqtt.Token {
					desiresState := &types.DesiredState{}
					envelope, err := types.FromEnvelope(payload.([]byte), desiresState)
					assert.NoError(t, err)
					assert.Equal(t, name, envelope.ActivityID)
					assert.True(t, envelope.Timestamp > 0)
					assert.Equal(t, testDesiredState, desiresState)
					return mockToken
				})
			setupMockToken(mockToken, mqttTestConfig.AcknowledgeTimeout, test.isTimedOut)

			assertOutgoingResult(t, test.isTimedOut, desiredStateClient.SendDesiredState(name, testDesiredState))
		})
	}
}

func TestSendDesiredStateCommand(t *testing.T) {
	tests := map[string]testCaseOutgoing{
		"test_send_desired_state_command_ok":    {domain: "testdomain", isTimedOut: false},
		"test_send_desired_state_command_error": {domain: "mydomain", isTimedOut: true},
	}

	mockCtrl, mockPaho, mockToken := setupCommonMocks(t)
	defer mockCtrl.Finish()

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			testDesiredStateCommand := &types.DesiredStateCommand{
				Command:  types.CommandDownload,
				Baseline: name,
			}
			desiredStateClient := NewDesiredStateClient(test.domain, &updateAgentClient{
				mqttClient: newInternalClient(test.domain, mqttTestConfig, mockPaho),
			})
			mockPaho.EXPECT().Publish(test.domain+"update/desiredstate/command", uint8(1), false, gomock.Any()).DoAndReturn(
				func(topic string, qos byte, retained bool, payload interface{}) pahomqtt.Token {
					desiresStateCommand := &types.DesiredStateCommand{}
					envelope, err := types.FromEnvelope(payload.([]byte), desiresStateCommand)
					assert.NoError(t, err)
					assert.Equal(t, name, envelope.ActivityID)
					assert.True(t, envelope.Timestamp > 0)
					assert.Equal(t, testDesiredStateCommand, desiresStateCommand)
					return mockToken
				})
			setupMockToken(mockToken, mqttTestConfig.AcknowledgeTimeout, test.isTimedOut)

			assertOutgoingResult(t, test.isTimedOut, desiredStateClient.SendDesiredStateCommand(name, testDesiredStateCommand))
		})
	}
}

func TestSendCurrentStateGet(t *testing.T) {
	tests := map[string]testCaseOutgoing{
		"test_send_current_state_get_ok":    {domain: "testdomain", isTimedOut: false},
		"test_send_current_state_get_error": {domain: "mydomain", isTimedOut: true},
	}

	mockCtrl, mockPaho, mockToken := setupCommonMocks(t)
	defer mockCtrl.Finish()

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			desiredStateClient := NewDesiredStateClient(test.domain, &updateAgentClient{
				mqttClient: newInternalClient(test.domain, mqttTestConfig, mockPaho),
			})
			mockPaho.EXPECT().Publish(test.domain+"update/currentstate/get", uint8(1), false, gomock.Any()).DoAndReturn(
				func(topic string, qos byte, retained bool, payload interface{}) pahomqtt.Token {
					envelope, err := types.FromEnvelope(payload.([]byte), nil)
					assert.NoError(t, err)
					assert.Equal(t, name, envelope.ActivityID)
					assert.True(t, envelope.Timestamp > 0)
					assert.Nil(t, envelope.Payload)
					return mockToken
				})
			setupMockToken(mockToken, mqttTestConfig.AcknowledgeTimeout, test.isTimedOut)

			assertOutgoingResult(t, test.isTimedOut, desiredStateClient.SendCurrentStateGet(name))
		})
	}
}

func TestHandleCurrentStateMessage(t *testing.T) {
	tests := map[string]testCaseIncoming{
		"test_handle_current_state_ok":         {domain: "testdomain", handlerError: nil},
		"test_handler_current_state_error":     {domain: "mydomain", handlerError: errors.New("handler error")},
		"test_handle_current_state_json_error": {domain: "testdomain", handlerError: nil},
	}

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockMessage := mqttmocks.NewMockMessage(mockCtrl)

	testCurrentState := &types.Inventory{
		SoftwareNodes: []*types.SoftwareNode{
			{
				InventoryNode: types.InventoryNode{
					ID: "test-software-node",
				},
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {

			testBytes, expectedCalls := testBytesToEnvelope(t, name, testCurrentState)

			stateHandler := mocks.NewMockStateHandler(mockCtrl)
			stateHandler.EXPECT().HandleCurrentState(name, gomock.Any(), testCurrentState).Times(expectedCalls).Return(test.handlerError)

			desiredStateClient := &desiredStateClient{
				mqttClient:   newInternalClient(test.domain, &ConnectionConfig{}, nil),
				domain:       test.domain,
				stateHandler: stateHandler,
			}
			mockMessage.EXPECT().Topic().Return(test.domain + "update/currentstate")
			mockMessage.EXPECT().Payload().Return(testBytes)

			desiredStateClient.handleMessage(nil, mockMessage)
		})
	}
}

func TestHandleDesiredStateFeedbackMessage(t *testing.T) {
	tests := map[string]testCaseIncoming{
		"test_handle_desired_state_feedback_ok":         {domain: "testdomain", handlerError: nil},
		"test_handle_desired_state_feedback_error":      {domain: "mydomain", handlerError: errors.New("handler error")},
		"test_handle_desired_state_feedback_json_error": {domain: "testdomain", handlerError: nil},
	}

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockMessage := mqttmocks.NewMockMessage(mockCtrl)

	testFeedback := &types.DesiredStateFeedback{
		Status: types.StatusIdentified,
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			testBytes, expectedCalls := testBytesToEnvelope(t, name, testFeedback)

			stateHandler := mocks.NewMockStateHandler(mockCtrl)
			stateHandler.EXPECT().HandleDesiredStateFeedback(name, gomock.Any(), testFeedback).Times(expectedCalls).Return(test.handlerError)

			desiredStateClient := &desiredStateClient{
				mqttClient:   newInternalClient(test.domain, &ConnectionConfig{}, nil),
				domain:       test.domain,
				stateHandler: stateHandler,
			}
			mockMessage.EXPECT().Topic().Return(test.domain + "update/desiredstatefeedback")
			mockMessage.EXPECT().Payload().Return(testBytes)

			desiredStateClient.handleMessage(nil, mockMessage)
		})
	}
}

// In case of JSON error test return invalid []byte and change the expected number of tested method calls
func testBytesToEnvelope(t *testing.T, name string, payload interface{}) ([]byte, int) {
	var err error
	testBytes := []byte{}
	expectedCalls := 0
	if !strings.Contains(name, "json_error") {
		testBytes, err = types.ToEnvelope(name, payload)
		assert.NoError(t, err)
		expectedCalls = 1
	}
	return testBytes, expectedCalls
}
