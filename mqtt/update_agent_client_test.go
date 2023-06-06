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
	"fmt"
	"testing"
	"time"

	mqttmocks "github.com/eclipse-kanto/update-manager/mqtt/mock"
	"github.com/eclipse-kanto/update-manager/test/mocks"

	gomock "github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

const testDomain = "test-domain"

func TestConnect(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockPaho := mqttmocks.NewMockClient(mockCtrl)
	mockToken := mqttmocks.NewMockToken(mockCtrl)

	client := &mqttClient{
		pahoClient: mockPaho,
		mqttConfig: &ConnectionConfig{
			ConnectTimeout: 5,
		},
	}

	t.Run("test_Connect_waitTimeout_true_handleDesiredState_and_getCurrentState_not_nil", func(t *testing.T) {
		updateAgentClient := &updateAgentClient{
			domain:     testDomain,
			mqttClient: client,
		}

		mockHandler := mocks.NewMockUpdateAgentHandler(mockCtrl)

		mockPaho.EXPECT().Connect().Return(mockToken).Times(1)
		mockToken.EXPECT().WaitTimeout(time.Duration(5000000)).Return(true).Times(1)
		mockToken.EXPECT().Error().Times(1)

		connectErr := updateAgentClient.Connect(mockHandler)

		assert.NotNil(t, updateAgentClient.handler)
		assert.Equal(t, nil, connectErr)
	})

	t.Run("test_Connect_waitTimeout_false_handleDesiredState_and_getCurrentState_not_nil", func(t *testing.T) {
		updateAgentClient := &updateAgentClient{
			domain:     testDomain,
			mqttClient: client,
		}

		mockHandler := mocks.NewMockUpdateAgentHandler(mockCtrl)

		mockPaho.EXPECT().Connect().Return(mockToken).Times(1)
		mockToken.EXPECT().WaitTimeout(time.Duration(5000000)).Return(false).Times(1)

		connectErr := updateAgentClient.Connect(mockHandler)

		assert.NotNil(t, updateAgentClient.handler)
		assert.Equal(t, fmt.Errorf("[test-domain] connect timed out"), connectErr)
	})
}

func TestDisconnect(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockPaho := mqttmocks.NewMockClient(mockCtrl)
	mockToken := mqttmocks.NewMockToken(mockCtrl)

	client := &mqttClient{
		mqttPrefix: "testTopic",
		pahoClient: mockPaho,
		mqttConfig: &ConnectionConfig{
			UnsubscribeTimeout: 5,
		},
	}

	t.Run("test_Disconnect_waitTimeout_false_assert_handleDesiredState_and_getCurrentState_nil", func(t *testing.T) {
		mockHandler := mocks.NewMockUpdateAgentHandler(mockCtrl)
		updateAgentClient := &updateAgentClient{
			mqttClient: client,
			handler:    mockHandler,
		}

		mockPaho.EXPECT().Unsubscribe("testTopic/desiredstate", "testTopic/desiredstate/command", "testTopic/currentstate/get").Return(mockToken).Times(1)
		mockToken.EXPECT().WaitTimeout(time.Duration(5000000)).Return(false)
		mockPaho.EXPECT().Disconnect(uint(10000))

		updateAgentClient.Disconnect()

		assert.Nil(t, updateAgentClient.handler)
	})

	t.Run("test_Disconnect_waitTimeout_true_assert_handleDesiredState_and_getCurrentState_nil", func(t *testing.T) {
		mockHandler := mocks.NewMockUpdateAgentHandler(mockCtrl)
		updateAgentClient := &updateAgentClient{
			mqttClient: client,
			handler:    mockHandler,
		}

		mockPaho.EXPECT().Unsubscribe("testTopic/desiredstate", "testTopic/desiredstate/command", "testTopic/currentstate/get").Return(mockToken).Times(1)
		mockToken.EXPECT().WaitTimeout(time.Duration(5000000)).Return(true)
		mockPaho.EXPECT().Disconnect(uint(10000))
		mockToken.EXPECT().Error()

		updateAgentClient.Disconnect()

		assert.Nil(t, updateAgentClient.handler)
	})
}

func TestPublishDesiredStateFeedback(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockPaho := mqttmocks.NewMockClient(mockCtrl)
	mockToken := mqttmocks.NewMockToken(mockCtrl)

	client := &mqttClient{
		mqttPrefix: "testTopic",
		pahoClient: mockPaho,
		mqttConfig: &ConnectionConfig{
			AcknowledgeTimeout: 5,
		},
	}

	updateAgentClient := &updateAgentClient{
		mqttClient: client,
	}

	mockPaho.EXPECT().Publish("testTopic/desiredstatefeedback", uint8(1), false, []byte("testdesiredstate")).Return(mockToken).Times(1)
	publishDesiredStateFeedbackErr := updateAgentClient.PublishDesiredStateFeedback([]byte("testdesiredstate"))

	assert.Equal(t, nil, publishDesiredStateFeedbackErr)
}

func TestPublishCurrentState(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockPaho := mqttmocks.NewMockClient(mockCtrl)
	mockToken := mqttmocks.NewMockToken(mockCtrl)

	client := &mqttClient{
		mqttPrefix: "testTopic",
		pahoClient: mockPaho,
		mqttConfig: &ConnectionConfig{
			AcknowledgeTimeout: 5,
		},
	}

	updateAgentClient := &updateAgentClient{
		mqttClient: client,
	}
	mockPaho.EXPECT().Publish("testTopic/currentstate", uint8(1), true, []byte("testdesiredstate")).Return(mockToken).Times(1)
	publishCurrentStateErr := updateAgentClient.PublishCurrentState([]byte("testdesiredstate"))
	assert.Equal(t, nil, publishCurrentStateErr)
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
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockPaho := mqttmocks.NewMockClient(mockCtrl)
	mockToken := mqttmocks.NewMockToken(mockCtrl)

	client := &mqttClient{
		mqttPrefix: "testTopic",
		pahoClient: mockPaho,
		mqttConfig: &ConnectionConfig{
			SubscribeTimeout:   5,
			AcknowledgeTimeout: 5,
		},
	}

	t.Run("test_onConnect_getCurrentState", func(t *testing.T) {
		mockHandler := mocks.NewMockUpdateAgentHandler(mockCtrl)
		mockHandler.EXPECT().HandleCurrentStateGet(gomock.Any())
		updateAgentClient := &updateAgentClient{
			mqttClient: client,
			handler:    mockHandler,
		}
		topicsMap := map[string]byte{
			"testTopic/currentstate/get":     1,
			"testTopic/desiredstate":         1,
			"testTopic/desiredstate/command": 1,
		}
		mockPaho.EXPECT().SubscribeMultiple(topicsMap, gomock.Any()).Return(mockToken)
		mockToken.EXPECT().WaitTimeout(time.Duration(5000000)).Return(false).Times(1)

		updateAgentClient.onConnect(nil)

		// Wait for go-routine to process event.
		time.Sleep(500 * time.Millisecond)
	})

}

func TestHandleDesiredStateRequest(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockPaho := mqttmocks.NewMockClient(mockCtrl)
	mockMessage := mqttmocks.NewMockMessage(mockCtrl)

	client := &mqttClient{
		mqttPrefix: "testTopic",
		pahoClient: mockPaho,
		mqttConfig: &ConnectionConfig{},
	}

	t.Run("test_handleDesiredStateRequest_handleDesiredStateErr_nil", func(t *testing.T) {
		mockHandler := mocks.NewMockUpdateAgentHandler(mockCtrl)
		mockHandler.EXPECT().HandleDesiredState([]byte("testDesiredStateCall"))
		updateAgentClient := &updateAgentClient{
			mqttClient: client,
			handler:    mockHandler,
		}
		mockMessage.EXPECT().Payload().Return([]byte("testDesiredStateCall")).Times(1)
		mockMessage.EXPECT().Topic().Return("testTopic/desiredstate").Times(1)

		updateAgentClient.handleStateRequest(mockPaho, mockMessage)
	})

	t.Run("test_handleDesiredStateRequest_handleDesiredStateErr_notNil", func(t *testing.T) {
		mockHandler := mocks.NewMockUpdateAgentHandler(mockCtrl)
		mockHandler.EXPECT().HandleDesiredState([]byte("testDesiredStateCall")).Return(fmt.Errorf("errNotNil"))
		updateAgentClient := &updateAgentClient{
			mqttClient: client,
			handler:    mockHandler,
		}
		mockMessage.EXPECT().Payload().Return([]byte("testDesiredStateCall")).Times(1)
		mockMessage.EXPECT().Topic().Return("testTopic/desiredstate").Times(1)

		updateAgentClient.handleStateRequest(mockPaho, mockMessage)
	})
}
