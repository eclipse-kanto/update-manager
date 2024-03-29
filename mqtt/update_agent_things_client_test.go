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
	"encoding/json"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/eclipse-kanto/update-manager/api/types"
	mqttmocks "github.com/eclipse-kanto/update-manager/mqtt/mocks"
	"github.com/eclipse-kanto/update-manager/test"
	"github.com/eclipse-kanto/update-manager/test/mocks"
	"github.com/golang/mock/gomock"

	"github.com/stretchr/testify/assert"
)

func TestUpdateAgentThingsClientStart(t *testing.T) {
	tests := map[string]testCaseOutgoing{
		"test_connect_ok":      {domain: test.Domain, isTimedOut: false},
		"test_connect_timeout": {domain: test.Domain, isTimedOut: true},
	}

	mockCtrl, mockPaho, mockToken := setupCommonMocks(t)
	defer mockCtrl.Finish()

	mockHandler := mocks.NewMockUpdateAgentHandler(mockCtrl)

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			updateAgentThingsClient := &updateAgentThingsClient{
				updateAgentClient: &updateAgentClient{domain: test.domain,
					mqttClient: newInternalClient(test.domain, mqttTestConfig, mockPaho),
				},
			}

			mockPaho.EXPECT().Connect().Return(mockToken)
			setupMockToken(mockToken, mqttTestConfig.ConnectTimeout, test.isTimedOut)

			assertOutgoingResult(t, test.isTimedOut, updateAgentThingsClient.Start(mockHandler))
			assert.Equal(t, mockHandler, updateAgentThingsClient.handler)
		})
	}
}

func TestUpdateAgentThingClientStop(t *testing.T) {
	tests := map[string]testCaseOutgoing{
		"test_disconnect_ok":      {domain: test.Domain, isTimedOut: false},
		"test_disconnect_timeout": {domain: test.Domain, isTimedOut: true},
	}

	mockCtrl, mockPaho, mockToken := setupCommonMocks(t)
	defer mockCtrl.Finish()

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			updateAgentClient := &updateAgentThingsClient{
				updateAgentClient: &updateAgentClient{
					domain:     test.domain,
					mqttClient: newInternalClient(test.domain, mqttTestConfig, mockPaho),
				},
			}

			mockPaho.EXPECT().Unsubscribe(edgeResponseTopic).Return(mockToken)
			mockPaho.EXPECT().Disconnect(disconnectQuiesce)
			setupMockToken(mockToken, mqttTestConfig.UnsubscribeTimeout, test.isTimedOut)

			assert.NoError(t, updateAgentClient.Stop())
			assert.Nil(t, updateAgentClient.handler)
		})
	}
}

func TestThingsSendDesiredStateFeedback(t *testing.T) {
	tests := map[string]struct {
		domain string
		err    error
	}{
		"test_things_send_desired_state_feedback_ok":    {domain: test.Domain},
		"test_things_send_desired_state_feedback_error": {domain: test.Domain, err: fmt.Errorf("test error")},
	}

	mockCtrl, mockPaho, _ := setupCommonMocks(t)
	mockFeature := mocks.NewMockUpdateManagerFeature(mockCtrl)
	defer mockCtrl.Finish()

	testFeedback := &types.DesiredStateFeedback{
		Status: types.StatusCompleted,
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			updateAgentClient := &updateAgentThingsClient{
				updateAgentClient: &updateAgentClient{
					domain:     test.domain,
					mqttClient: newInternalClient(test.domain, mqttTestConfig, mockPaho),
				},
				umFeature: mockFeature,
			}
			mockFeature.EXPECT().SendFeedback(name, gomock.Any()).DoAndReturn(
				func(activityID string, desiredStateFeedback *types.DesiredStateFeedback) error {
					assert.Equal(t, name, activityID)
					assert.Equal(t, testFeedback, desiredStateFeedback)
					return test.err
				})

			if test.err != nil {
				assert.Errorf(t, updateAgentClient.SendDesiredStateFeedback(name, testFeedback), test.err.Error())
			} else {
				assert.NoError(t, updateAgentClient.SendDesiredStateFeedback(name, testFeedback))
			}
		})
	}
}

func TestThingsSendCurrentState(t *testing.T) {
	tests := map[string]struct {
		domain string
		err    error
	}{
		"test_send_current_state_ok":    {domain: test.Domain},
		"test_send_current_state_error": {domain: test.Domain, err: fmt.Errorf("test error")},
	}

	mockCtrl, mockPaho, _ := setupCommonMocks(t)
	mockFeature := mocks.NewMockUpdateManagerFeature(mockCtrl)
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
			updateAgentClient := &updateAgentThingsClient{
				updateAgentClient: &updateAgentClient{
					domain:     test.domain,
					mqttClient: newInternalClient(test.domain, mqttTestConfig, mockPaho),
				},
				umFeature: mockFeature,
			}
			mockFeature.EXPECT().SetState(name, gomock.Any()).DoAndReturn(
				func(activityID string, inventory *types.Inventory) error {
					assert.Equal(t, name, activityID)
					assert.Equal(t, testCurrentState, inventory)
					return test.err
				})

			if test.err != nil {
				assert.Errorf(t, updateAgentClient.SendCurrentState(name, testCurrentState), test.err.Error())
			} else {
				assert.NoError(t, updateAgentClient.SendCurrentState(name, testCurrentState))
			}
		})
	}
}

func TestThingOnConnect(t *testing.T) {
	mockCtrl, mockPaho, mockToken := setupCommonMocks(t)
	defer mockCtrl.Finish()

	updateAgentThingsClient := &updateAgentThingsClient{
		updateAgentClient: &updateAgentClient{
			mqttClient: newInternalClient("test", mqttTestConfig, mockPaho),
			domain:     "test",
		},
	}
	t.Run("test_onConnect", func(t *testing.T) {
		mockPaho.EXPECT().Subscribe(edgeResponseTopic, byte(1), gomock.Any()).Return(mockToken)
		setupMockToken(mockToken, mqttTestConfig.SubscribeTimeout, false)
		mockPaho.EXPECT().Publish(edgeRequestTopic, byte(1), false, "").Return(mockToken)
		setupMockToken(mockToken, mqttTestConfig.AcknowledgeTimeout, false)

		updateAgentThingsClient.onConnect(nil)
	})
	t.Run("test_onConnect_timeout", func(t *testing.T) {
		mockPaho.EXPECT().Subscribe(edgeResponseTopic, byte(1), gomock.Any()).Return(mockToken)
		setupMockToken(mockToken, mqttTestConfig.SubscribeTimeout, true)

		updateAgentThingsClient.onConnect(nil)
	})

}

func TestHandleEdgeResponseThingsHandleDesiredStateMessage(t *testing.T) {

	mockCtrl, mockPaho, mockToken := setupCommonMocks(t)
	defer mockCtrl.Finish()

	mockMessage := mqttmocks.NewMockMessage(mockCtrl)
	mockHandler := mocks.NewMockUpdateAgentHandler(mockCtrl)

	testEdgeConfig := &edgeConfiguration{DeviceID: "namespace:testDevice", TenantID: "testTenant", PolicyID: "testPolicy"}
	testBytes, _ := json.Marshal(testEdgeConfig)
	testWG := &sync.WaitGroup{}
	testWG.Add(1)

	updateAgentThingsClient := &updateAgentThingsClient{
		updateAgentClient: &updateAgentClient{
			mqttClient: newInternalClient(test.Domain, mqttTestConfig, mockPaho),
			domain:     test.Domain,
			handler:    mockHandler,
		},
	}

	mockMessage.EXPECT().Payload().Return(testBytes)
	mockPaho.EXPECT().IsConnected().Return(true)
	mockPaho.EXPECT().Subscribe("command///req/#", uint8(1), gomock.Any()).Return(mockToken)
	setupMockToken(mockToken, mqttTestConfig.SubscribeTimeout, false)
	mockPaho.EXPECT().Publish("e", uint8(1), false, gomock.Any()).Return(mockToken)
	setupMockToken(mockToken, mqttTestConfig.AcknowledgeTimeout, false)
	mockHandler.EXPECT().HandleCurrentStateGet(gomock.Any(), gomock.Any()).DoAndReturn(func(string, int64) error {
		testWG.Done()
		return nil
	})

	updateAgentThingsClient.handleEdgeResponse(nil, mockMessage)
	assert.Equal(t, testEdgeConfig, updateAgentThingsClient.edgeConfig)
	assert.NotNil(t, updateAgentThingsClient.dittoClient)
	assert.NotNil(t, updateAgentThingsClient.umFeature)
	test.AssertWithTimeout(t, testWG, 2*time.Second)
}
