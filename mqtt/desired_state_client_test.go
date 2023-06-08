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

	mocks "github.com/eclipse-kanto/update-manager/mqtt/mock"

	gomock "github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

type dummyStateHandler struct {
	feedbackPayload     []byte
	feedbackErr         error
	currentStatePayload []byte
	currentStateErr     error
}

func (s *dummyStateHandler) HandleDesiredStateFeedback(bytes []byte) error {
	s.feedbackPayload = bytes
	return s.feedbackErr
}

func (s *dummyStateHandler) HandleCurrentState(bytes []byte) error {
	s.currentStatePayload = bytes
	return s.currentStateErr
}

func (s *dummyStateHandler) setCurrentStateError(err error) {
	s.currentStateErr = err
}

func TestDomain(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockPaho := mocks.NewMockClient(mockCtrl)

	client := &mqttClient{
		pahoClient: mockPaho,
		mqttConfig: &ConnectionConfig{},
	}

	updateAgentClient := &updateAgentClient{
		mqttClient: client,
	}
	desiredStateClient := NewDesiredStateClient("testDomain", updateAgentClient)

	res := desiredStateClient.Domain()
	assert.Equal(t, "testDomain", res)
}

func TestSubscribe(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockPaho := mocks.NewMockClient(mockCtrl)
	mockToken := mocks.NewMockToken(mockCtrl)

	uacMqtt := &mqttClient{
		pahoClient: mockPaho,
		mqttConfig: &ConnectionConfig{
			SubscribeTimeout: 5,
		},
	}

	updateAgentClient := &updateAgentClient{
		mqttClient: uacMqtt,
	}

	desiredStateClient := &desiredStateClient{
		mqttClient: &mqttClient{
			mqttPrefix: domainAsTopic("testDomain"),
			mqttConfig: updateAgentClient.mqttConfig,
			pahoClient: updateAgentClient.pahoClient,
		},
		domain: "testDomain",
	}

	t.Run("test_Subscribe_error_token_false", func(t *testing.T) {
		mockPaho.EXPECT().SubscribeMultiple(map[string]byte{desiredStateClient.topic(suffixCurrentState): 1, desiredStateClient.topic(suffixDesiredStateFeedback): 1}, gomock.Any()).Return(mockToken).Times(1)
		mockToken.EXPECT().WaitTimeout(time.Duration(5000000)).Return(false).Times(1)
		mockToken.EXPECT().Error().Times(1)

		err := desiredStateClient.Subscribe(&dummyStateHandler{})
		assert.NotNil(t, err)
	})

	t.Run("test_Subscribe_correct_token_true", func(t *testing.T) {
		mockPaho.EXPECT().SubscribeMultiple(map[string]byte{desiredStateClient.topic(suffixCurrentState): 1, desiredStateClient.topic(suffixDesiredStateFeedback): 1}, gomock.Any()).Return(mockToken).Times(1)
		mockToken.EXPECT().WaitTimeout(time.Duration(5000000)).Return(true).Times(1)

		assert.Nil(t, desiredStateClient.stateHandler)
		assert.Nil(t, desiredStateClient.Subscribe(&dummyStateHandler{}))

		assert.NotNil(t, desiredStateClient.stateHandler)
	})
}

func TestUnsubscribe(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockPaho := mocks.NewMockClient(mockCtrl)
	mockToken := mocks.NewMockToken(mockCtrl)

	uacMqtt := &mqttClient{
		pahoClient: mockPaho,
		mqttConfig: &ConnectionConfig{
			UnsubscribeTimeout: 5,
		},
	}

	updateAgentClient := &updateAgentClient{
		mqttClient: uacMqtt,
	}

	desiredStateClient := &desiredStateClient{
		mqttClient: &mqttClient{
			mqttPrefix: domainAsTopic("testDomain"),
			mqttConfig: updateAgentClient.mqttConfig,
			pahoClient: updateAgentClient.pahoClient,
		},
		domain: "testDomain",
	}

	t.Run("test_Unsubscribe_error_token_false", func(t *testing.T) {
		mockPaho.EXPECT().Unsubscribe("testDomainupdate/currentstate", "testDomainupdate/desiredstatefeedback").Return(mockToken).Times(1)
		mockToken.EXPECT().WaitTimeout(time.Duration(5000000)).Return(false).Times(1)

		desiredStateClient.stateHandler = &dummyStateHandler{}
		assert.NotNil(t, desiredStateClient.Unsubscribe())

		assert.NotNil(t, desiredStateClient.stateHandler)
		desiredStateClient.stateHandler = nil
	})

	t.Run("test_Unsubscribe_correct_token_true", func(t *testing.T) {
		mockPaho.EXPECT().Unsubscribe("testDomainupdate/currentstate", "testDomainupdate/desiredstatefeedback").Return(mockToken).Times(1)
		mockToken.EXPECT().WaitTimeout(time.Duration(5000000)).Return(true).Times(1)
		mockToken.EXPECT().Error().Times(1)

		desiredStateClient.stateHandler = &dummyStateHandler{}
		assert.Nil(t, desiredStateClient.Unsubscribe())

		assert.Nil(t, desiredStateClient.stateHandler)
	})
}

func TestPublishDesiredState(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockPaho := mocks.NewMockClient(mockCtrl)
	mockToken := mocks.NewMockToken(mockCtrl)

	client := &mqttClient{
		pahoClient: mockPaho,
		mqttConfig: &ConnectionConfig{
			AcknowledgeTimeout: 5,
		},
	}

	updateAgentClient := &updateAgentClient{
		mqttClient: client,
	}
	desiredStateClient := NewDesiredStateClient("testDomain", updateAgentClient)

	t.Run("test_PublishDesiredState_correct_token_true", func(t *testing.T) {
		mockPaho.EXPECT().Publish("testDomainupdate/desiredstate", uint8(1), false, []byte("testdesiredstate")).Return(mockToken).Times(1)
		mockToken.EXPECT().WaitTimeout(time.Duration(5000000)).Return(true).Times(1)
		mockToken.EXPECT().Error().Times(1)

		err := desiredStateClient.PublishDesiredState([]byte("testdesiredstate"))
		assert.Equal(t, nil, err)
	})

	t.Run("test_PublishDesiredState_error_token_false", func(t *testing.T) {
		mockPaho.EXPECT().Publish("testDomainupdate/desiredstate", uint8(1), false, []byte("testdesiredstate")).Return(mockToken).Times(1)
		mockToken.EXPECT().WaitTimeout(time.Duration(5000000)).Return(false).Times(1)

		err := desiredStateClient.PublishDesiredState([]byte("testdesiredstate"))
		assert.Equal(t, fmt.Errorf("cannot publish to topic '%s' in '%vms' seconds", "testDomainupdate/desiredstate", 5), err)
	})
}

func TestHandleMessage(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockMessage := mocks.NewMockMessage(mockCtrl)

	t.Run("test_handleMessage_handleCurrentState_err_nil", func(t *testing.T) {
		stateHandler := &dummyStateHandler{}
		desiredStateClient := &desiredStateClient{
			mqttClient: &mqttClient{
				mqttPrefix: "testprefix",
			},
			stateHandler: stateHandler,
		}
		mockMessage.EXPECT().Topic().Return("testprefix/currentstate").Times(1)
		mockMessage.EXPECT().Payload().Return([]byte("testCurrStateCall")).Times(1)

		desiredStateClient.handleMessage(nil, mockMessage)
		assert.Equal(t, []byte("testCurrStateCall"), stateHandler.currentStatePayload)
	})

	t.Run("test_handleMessage_handleCurrentState_err_not_nil", func(t *testing.T) {
		stateHandler := &dummyStateHandler{}
		stateHandler.setCurrentStateError(fmt.Errorf("errNotNil"))

		desiredStateClient := &desiredStateClient{
			mqttClient: &mqttClient{
				mqttPrefix: "testprefix",
			},
			stateHandler: stateHandler,
		}
		mockMessage.EXPECT().Topic().Return("testprefix/currentstate").Times(1)
		mockMessage.EXPECT().Payload().Return([]byte("testCurrStateCall")).Times(1)

		desiredStateClient.handleMessage(nil, mockMessage)
		assert.Equal(t, []byte("testCurrStateCall"), stateHandler.currentStatePayload)
	})

	t.Run("test_handleMessage_handleDesiredStateFeedback_err_nil", func(t *testing.T) {
		stateHandler := &dummyStateHandler{}

		desiredStateClient := &desiredStateClient{
			mqttClient: &mqttClient{
				mqttPrefix: "testprefix",
			},
			stateHandler: stateHandler,
		}
		mockMessage.EXPECT().Topic().Return("testprefix/desiredstatefeedback").Times(1)
		mockMessage.EXPECT().Payload().Return([]byte("testDesiredStateCall")).Times(1)

		desiredStateClient.handleMessage(nil, mockMessage)
		assert.Equal(t, []byte("testDesiredStateCall"), stateHandler.feedbackPayload)
	})

	t.Run("test_handleMessage_handleDesiredStateFeedback_err_not_nil", func(t *testing.T) {
		stateHandler := &dummyStateHandler{}
		stateHandler.setCurrentStateError(fmt.Errorf("errNotNil"))

		desiredStateClient := &desiredStateClient{
			mqttClient: &mqttClient{
				mqttPrefix: "testprefix",
			},
			stateHandler: stateHandler,
		}
		mockMessage.EXPECT().Topic().Return("testprefix/desiredstatefeedback").Times(1)
		mockMessage.EXPECT().Payload().Return([]byte("testDesiredStateCall")).Times(1)

		desiredStateClient.handleMessage(nil, mockMessage)
		assert.Equal(t, []byte("testDesiredStateCall"), stateHandler.feedbackPayload)
	})
}
