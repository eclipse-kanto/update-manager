// Copyright (c) 2024 Contributors to the Eclipse Foundation
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

	"github.com/eclipse-kanto/update-manager/api/types"
	mqttmocks "github.com/eclipse-kanto/update-manager/mqtt/mocks"
	"github.com/eclipse-kanto/update-manager/test/mocks"

	pahomqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestOwnerConsentAgentClientStart(t *testing.T) {
	tests := map[string]testCaseOutgoing{
		"test_connect_ok":      {domain: "testdomain", isTimedOut: false},
		"test_connect_timeout": {domain: "mydomain", isTimedOut: true},
	}

	mockCtrl, mockPaho, mockToken := setupCommonMocks(t)
	defer mockCtrl.Finish()

	mockHandler := mocks.NewMockOwnerConsentAgentHandler(mockCtrl)

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			client := &ownerConsentAgentClient{
				domain:     test.domain,
				mqttClient: newInternalClient(test.domain, mqttTestConfig, mockPaho),
			}

			mockPaho.EXPECT().Connect().Return(mockToken)
			setupMockToken(mockToken, mqttTestConfig.ConnectTimeout, test.isTimedOut)

			assertOutgoingResult(t, test.isTimedOut, client.Start(mockHandler))
			assert.Equal(t, mockHandler, client.handler)
		})
	}
}

func TestOwnerConsentAgentClientStop(t *testing.T) {
	tests := map[string]testCaseOutgoing{
		//"test_disconnect_ok":      {domain: "testdomain", isTimedOut: false},
		"test_disconnect_timeout": {domain: "mydomain", isTimedOut: true},
	}

	mockCtrl, mockPaho, mockToken := setupCommonMocks(t)
	defer mockCtrl.Finish()

	mockHandler := mocks.NewMockOwnerConsentAgentHandler(mockCtrl)

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			client := &ownerConsentAgentClient{
				domain:     test.domain,
				mqttClient: newInternalClient(test.domain, mqttTestConfig, mockPaho),
				handler:    mockHandler,
			}

			mockPaho.EXPECT().Unsubscribe(test.domain + "update/ownerconsent").Return(mockToken)
			mockPaho.EXPECT().Disconnect(disconnectQuiesce)
			setupMockToken(mockToken, mqttTestConfig.UnsubscribeTimeout, test.isTimedOut)

			assert.NoError(t, client.Stop())
			assert.Nil(t, client.handler)
		})
	}
}

func TestSendOwnerConsentFeedback(t *testing.T) {
	tests := map[string]testCaseOutgoing{
		"test_send_owner_consent_feedback_ok":    {domain: "testdomain", isTimedOut: false},
		"test_send_owner_consent_feedback_error": {domain: "mydomain", isTimedOut: true},
	}

	mockCtrl, mockPaho, mockToken := setupCommonMocks(t)
	defer mockCtrl.Finish()

	testConsentFeedback := &types.OwnerConsentFeedback{
		Status: types.StatusApproved,
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			client := &ownerConsentAgentClient{
				domain:     test.domain,
				mqttClient: newInternalClient(test.domain, mqttTestConfig, mockPaho),
			}
			mockPaho.EXPECT().Publish(test.domain+"update/ownerconsentfeedback", uint8(1), false, gomock.Any()).DoAndReturn(
				func(topic string, qos byte, retained bool, payload interface{}) pahomqtt.Token {
					consentFeedback := &types.OwnerConsentFeedback{}
					envelope, err := types.FromEnvelope(payload.([]byte), consentFeedback)
					assert.NoError(t, err)
					assert.Equal(t, name, envelope.ActivityID)
					assert.True(t, envelope.Timestamp > 0)
					assert.Equal(t, testConsentFeedback, consentFeedback)
					return mockToken
				})
			setupMockToken(mockToken, mqttTestConfig.AcknowledgeTimeout, false)

			assert.NoError(t, client.SendOwnerConsentFeedback(name, testConsentFeedback))
		})
	}
}

func TestOwnerConsentOnConnect(t *testing.T) {
	mockCtrl, mockPaho, mockToken := setupCommonMocks(t)
	defer mockCtrl.Finish()

	client := newInternalClient("test", mqttTestConfig, mockPaho)

	t.Run("test_onConnect", func(t *testing.T) {
		mockHandler := mocks.NewMockOwnerConsentAgentHandler(mockCtrl)
		client := &ownerConsentAgentClient{
			mqttClient: client,
			domain:     "test",
			handler:    mockHandler,
		}
		mockPaho.EXPECT().Subscribe("testupdate/ownerconsent", uint8(1), gomock.Any()).Return(mockToken)
		setupMockToken(mockToken, mqttTestConfig.SubscribeTimeout, false)

		client.onConnect(nil)
	})
}

func TestHandleOwnerConsentMessage(t *testing.T) {
	tests := map[string]testCaseIncoming{
		"test_handle_owner_consent_ok":         {domain: "testdomain", handlerError: nil, expectedJSONErr: false},
		"test_handle_owner_consent_error":      {domain: "mydomain", handlerError: errors.New("handler error"), expectedJSONErr: false},
		"test_handle_owner_consent_json_error": {domain: "testdomain", handlerError: nil, expectedJSONErr: true},
	}

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockMessage := mqttmocks.NewMockMessage(mockCtrl)

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			testConsent := &types.OwnerConsent{
				Command: types.CommandDownload,
			}
			testBytes, expectedCalls := testBytesToEnvelope(t, name, testConsent, test.expectedJSONErr)

			mockHandler := mocks.NewMockOwnerConsentAgentHandler(mockCtrl)
			mockHandler.EXPECT().HandleOwnerConsent(name, gomock.Any(), testConsent).Times(expectedCalls).Return(test.handlerError)

			client := &ownerConsentAgentClient{
				mqttClient: newInternalClient(test.domain, mqttTestConfig, nil),
				domain:     test.domain,
				handler:    mockHandler,
			}
			mockMessage.EXPECT().Topic().Return(test.domain + "update/ownerconsent")
			mockMessage.EXPECT().Payload().Return(testBytes)

			client.handleMessage(nil, mockMessage)
		})
	}
}
