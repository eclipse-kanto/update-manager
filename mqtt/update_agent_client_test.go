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
	"testing"
	"time"

	"github.com/eclipse-kanto/update-manager/api/types"
	mqttmocks "github.com/eclipse-kanto/update-manager/mqtt/mocks"
	"github.com/eclipse-kanto/update-manager/test/mocks"

	pahomqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

type testCaseOutgoing struct {
	domain     string
	isTimedOut bool
}

type testCaseIncoming struct {
	domain          string
	handlerError    error
	expectedJSONErr bool
}

var mqttTestConfig = &internalConnectionConfig{
	ConnectTimeout:     time.Duration(10) * time.Millisecond,
	DisconnectTimeout:  time.Duration(8) * time.Millisecond,
	AcknowledgeTimeout: time.Duration(5) * time.Millisecond,
	SubscribeTimeout:   time.Duration(4) * time.Millisecond,
	UnsubscribeTimeout: time.Duration(3) * time.Millisecond,
}

func setupCommonMocks(t *testing.T) (*gomock.Controller, *mqttmocks.MockClient, *mqttmocks.MockToken) {
	mockCtrl := gomock.NewController(t)
	return mockCtrl, mqttmocks.NewMockClient(mockCtrl), mqttmocks.NewMockToken(mockCtrl)
}

func setupMockToken(mockToken *mqttmocks.MockToken, duration time.Duration, isTimedOut bool) {
	if isTimedOut {
		mockToken.EXPECT().WaitTimeout(duration).Return(false)
	} else {
		mockToken.EXPECT().WaitTimeout(duration).Return(true)
		mockToken.EXPECT().Error()
	}
}

func assertOutgoingResult(t *testing.T, isTimedOut bool, operationErr error) {
	if isTimedOut {
		assert.Error(t, operationErr)
	} else {
		assert.NoError(t, operationErr)
	}
}

func TestUpdateAgentClientStart(t *testing.T) {
	tests := map[string]testCaseOutgoing{
		"test_connect_ok":      {domain: "testdomain", isTimedOut: false},
		"test_connect_timeout": {domain: "mydomain", isTimedOut: true},
	}

	mockCtrl, mockPaho, mockToken := setupCommonMocks(t)
	defer mockCtrl.Finish()

	mockHandler := mocks.NewMockUpdateAgentHandler(mockCtrl)

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			updateAgentClient := &updateAgentClient{
				domain:     test.domain,
				mqttClient: newInternalClient(test.domain, mqttTestConfig, mockPaho),
			}

			mockPaho.EXPECT().Connect().Return(mockToken)
			setupMockToken(mockToken, mqttTestConfig.ConnectTimeout, test.isTimedOut)

			assertOutgoingResult(t, test.isTimedOut, updateAgentClient.Start(mockHandler))
			assert.Equal(t, mockHandler, updateAgentClient.handler)
		})
	}
}

func TestUpdateAgentClientStop(t *testing.T) {
	tests := map[string]testCaseOutgoing{
		"test_disconnect_ok":      {domain: "testdomain", isTimedOut: false},
		"test_disconnect_timeout": {domain: "mydomain", isTimedOut: true},
	}

	mockCtrl, mockPaho, mockToken := setupCommonMocks(t)
	defer mockCtrl.Finish()

	mockHandler := mocks.NewMockUpdateAgentHandler(mockCtrl)

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			updateAgentClient := &updateAgentClient{
				domain:     test.domain,
				mqttClient: newInternalClient(test.domain, mqttTestConfig, mockPaho),
				handler:    mockHandler,
			}

			mockPaho.EXPECT().Unsubscribe(test.domain+"update/desiredstate", test.domain+"update/desiredstate/command", test.domain+"update/currentstate/get").Return(mockToken)
			mockPaho.EXPECT().Disconnect(disconnectQuiesce)
			setupMockToken(mockToken, mqttTestConfig.UnsubscribeTimeout, test.isTimedOut)

			assert.NoError(t, updateAgentClient.Stop())
			assert.Nil(t, updateAgentClient.handler)
		})
	}
}

func TestSendDesiredStateFeedback(t *testing.T) {
	tests := map[string]testCaseOutgoing{
		"test_send_desired_state_feedback_ok":    {domain: "testdomain", isTimedOut: false},
		"test_send_desired_state_feedback_error": {domain: "mydomain", isTimedOut: true},
	}

	mockCtrl, mockPaho, mockToken := setupCommonMocks(t)
	defer mockCtrl.Finish()

	testFeedback := &types.DesiredStateFeedback{
		Status: types.StatusCompleted,
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			updateAgentClient := &updateAgentClient{
				domain:     test.domain,
				mqttClient: newInternalClient(test.domain, mqttTestConfig, mockPaho),
			}
			mockPaho.EXPECT().Publish(test.domain+"update/desiredstatefeedback", uint8(1), false, gomock.Any()).DoAndReturn(
				func(topic string, qos byte, retained bool, payload interface{}) pahomqtt.Token {
					desiresStateFeedback := &types.DesiredStateFeedback{}
					envelope, err := types.FromEnvelope(payload.([]byte), desiresStateFeedback)
					assert.NoError(t, err)
					assert.Equal(t, name, envelope.ActivityID)
					assert.True(t, envelope.Timestamp > 0)
					assert.Equal(t, testFeedback, desiresStateFeedback)
					return mockToken
				})

			assert.NoError(t, updateAgentClient.SendDesiredStateFeedback(name, testFeedback))
		})
	}
}

func TestSendCurrentState(t *testing.T) {
	tests := map[string]testCaseOutgoing{
		"test_send_current_state_ok":    {domain: "testdomain", isTimedOut: false},
		"test_send_current_state_error": {domain: "mydomain", isTimedOut: true},
	}

	mockCtrl, mockPaho, mockToken := setupCommonMocks(t)
	defer mockCtrl.Finish()

	testCurrentState := &types.Inventory{
		SoftwareNodes: []*types.SoftwareNode{
			{
				InventoryNode: types.InventoryNode{
					ID: "test-software-node",
				},
				Type: types.SoftwareTypeApplication,
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			updateAgentClient := &updateAgentClient{
				domain:     test.domain,
				mqttClient: newInternalClient(test.domain, mqttTestConfig, mockPaho),
			}
			mockPaho.EXPECT().Publish(test.domain+"update/currentstate", uint8(1), true, gomock.Any()).DoAndReturn(
				func(topic string, qos byte, retained bool, payload interface{}) pahomqtt.Token {
					inventory := &types.Inventory{}
					envelope, err := types.FromEnvelope(payload.([]byte), inventory)
					assert.NoError(t, err)
					assert.Equal(t, name, envelope.ActivityID)
					assert.True(t, envelope.Timestamp > 0)
					assert.Equal(t, testCurrentState, inventory)
					return mockToken
				})

			assert.NoError(t, updateAgentClient.SendCurrentState(name, testCurrentState))
		})
	}
}

func TestDomainAsTopic(t *testing.T) {
	t.Run("test_domainAsTopic_noSuffix", func(t *testing.T) {
		res := domainAsTopic("test")
		assert.Equal(t, "testupdate", res)
	})

	t.Run("test_domainAsTopic_withSuffix", func(t *testing.T) {
		res := domainAsTopic("testupdate")
		assert.Equal(t, "testupdate", res)
	})

	t.Run("test_domainAsTopic_noSuffix-withDash", func(t *testing.T) {
		res := domainAsTopic("test-withdash")
		assert.Equal(t, "testwithdashupdate", res)
	})

	t.Run("test_domainAsTopic_withSuffix-withDash", func(t *testing.T) {
		res := domainAsTopic("test-withdash-update")
		assert.Equal(t, "testwithdashupdate", res)
	})
}

func TestOnConnect(t *testing.T) {
	mockCtrl, mockPaho, mockToken := setupCommonMocks(t)
	defer mockCtrl.Finish()

	client := newInternalClient("test", mqttTestConfig, mockPaho)

	t.Run("test_onConnect_getCurrentState", func(t *testing.T) {
		mockHandler := mocks.NewMockUpdateAgentHandler(mockCtrl)
		mockHandler.EXPECT().HandleCurrentStateGet(gomock.Any(), gomock.Any())
		updateAgentClient := &updateAgentClient{
			mqttClient: client,
			domain:     "test",
			handler:    mockHandler,
		}
		topicsMap := map[string]byte{
			"testupdate/currentstate/get":     1,
			"testupdate/desiredstate":         1,
			"testupdate/desiredstate/command": 1,
		}
		mockPaho.EXPECT().SubscribeMultiple(topicsMap, gomock.Any()).Return(mockToken)
		setupMockToken(mockToken, mqttTestConfig.SubscribeTimeout, false)

		updateAgentClient.onConnect(nil)

		// Wait for go-routine to process event.
		time.Sleep(500 * time.Millisecond)
	})

}

func TestHandleDesiredStateMessage(t *testing.T) {
	tests := map[string]testCaseIncoming{
		"test_handle_desired_state_ok":         {domain: "testdomain", handlerError: nil, expectedJSONErr: false},
		"test_handle_desired_state_error":      {domain: "mydomain", handlerError: errors.New("handler error"), expectedJSONErr: false},
		"test_handle_desired_state_json_error": {domain: "testdomain", handlerError: nil, expectedJSONErr: true},
	}

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockMessage := mqttmocks.NewMockMessage(mockCtrl)

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			testDesiredState := &types.DesiredState{
				Domains: []*types.Domain{
					{ID: test.domain},
				},
			}
			testBytes, expectedCalls := testBytesToEnvelope(t, name, testDesiredState, test.expectedJSONErr)

			mockHandler := mocks.NewMockUpdateAgentHandler(mockCtrl)
			mockHandler.EXPECT().HandleDesiredState(name, gomock.Any(), testDesiredState).Times(expectedCalls).Return(test.handlerError)

			updateAgentClient := &updateAgentClient{
				mqttClient: newInternalClient(test.domain, mqttTestConfig, nil),
				domain:     test.domain,
				handler:    mockHandler,
			}
			mockMessage.EXPECT().Topic().Return(test.domain + "update/desiredstate")
			mockMessage.EXPECT().Payload().Return(testBytes)

			updateAgentClient.handleStateRequest(nil, mockMessage)
		})
	}
}

func TestHandleDesiredStateCommandMessage(t *testing.T) {
	tests := map[string]testCaseIncoming{
		"test_handle_desired_state_command_ok":         {domain: "testdomain", handlerError: nil, expectedJSONErr: false},
		"test_handle_desired_state_command_error":      {domain: "mydomain", handlerError: errors.New("handler error"), expectedJSONErr: false},
		"test_handle_desired_state_command_json_error": {domain: "testdomain", handlerError: nil, expectedJSONErr: true},
	}

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockMessage := mqttmocks.NewMockMessage(mockCtrl)

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			testDesiredStateCommand := &types.DesiredStateCommand{
				Baseline: test.domain,
				Command:  types.CommandUpdate,
			}
			testBytes, expectedCalls := testBytesToEnvelope(t, name, testDesiredStateCommand, test.expectedJSONErr)

			mockHandler := mocks.NewMockUpdateAgentHandler(mockCtrl)
			mockHandler.EXPECT().HandleDesiredStateCommand(name, gomock.Any(), testDesiredStateCommand).Times(expectedCalls).Return(test.handlerError)

			updateAgentClient := &updateAgentClient{
				mqttClient: newInternalClient(test.domain, mqttTestConfig, nil),
				domain:     test.domain,
				handler:    mockHandler,
			}
			mockMessage.EXPECT().Topic().Return(test.domain + "update/desiredstate/command")
			mockMessage.EXPECT().Payload().Return(testBytes)

			updateAgentClient.handleStateRequest(nil, mockMessage)
		})
	}
}

func TestHandleCurrentStateGetMessage(t *testing.T) {
	tests := map[string]testCaseIncoming{
		"test_handle_current_state_get_ok":         {domain: "testdomain", handlerError: nil, expectedJSONErr: false},
		"test_handle_current_state_get_error":      {domain: "mydomain", handlerError: errors.New("handler error"), expectedJSONErr: false},
		"test_handle_current_state_get_json_error": {domain: "testdomain", handlerError: nil, expectedJSONErr: true},
	}

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockMessage := mqttmocks.NewMockMessage(mockCtrl)

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			testBytes, expectedCalls := testBytesToEnvelope(t, name, nil, test.expectedJSONErr)

			mockHandler := mocks.NewMockUpdateAgentHandler(mockCtrl)
			mockHandler.EXPECT().HandleCurrentStateGet(name, gomock.Any()).Times(expectedCalls).Return(test.handlerError)

			updateAgentClient := &updateAgentClient{
				mqttClient: newInternalClient(test.domain, mqttTestConfig, nil),
				domain:     test.domain,
				handler:    mockHandler,
			}
			mockMessage.EXPECT().Topic().Return(test.domain + "update/currentstate/get")
			mockMessage.EXPECT().Payload().Return(testBytes)

			updateAgentClient.handleStateRequest(nil, mockMessage)
		})
	}
}
