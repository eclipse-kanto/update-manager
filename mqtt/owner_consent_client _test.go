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
	"fmt"
	"testing"

	"github.com/eclipse-kanto/update-manager/api"
	"github.com/eclipse-kanto/update-manager/api/types"
	clientsmocks "github.com/eclipse-kanto/update-manager/mqtt/mocks"
	"github.com/eclipse-kanto/update-manager/test/mocks"
	pahomqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/golang/mock/gomock"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func TestNewOwnerConsentClient(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockPaho := clientsmocks.NewMockClient(mockCtrl)
	mockClient := mocks.NewMockUpdateAgentClient(mockCtrl)

	tests := map[string]struct {
		client api.UpdateAgentClient
		err    string
	}{
		"test_update_agent_client": {
			client: &updateAgentClient{
				mqttClient: newInternalClient("testDomain", &internalConnectionConfig{}, mockPaho),
			},
		},
		"test_update_agent_things_client": {
			client: &updateAgentThingsClient{
				updateAgentClient: &updateAgentClient{
					mqttClient: newInternalClient("testDomain", &internalConnectionConfig{}, mockPaho),
				},
			},
		},
		"test_error": {
			client: mockClient,
			err:    fmt.Sprintf("unexpected type: %T", mockClient),
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			client, err := NewDesiredStateClient("testDomain", test.client)
			if test.err != "" {
				assert.EqualError(t, err, fmt.Sprintf("unexpected type: %T", test.client))
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, client)
			}
		})
	}
}

func TestOwnerConsentClientStart(t *testing.T) {
	tests := map[string]testCaseOutgoing{
		"test_subscribe_ok":      {domain: "testdomain", isTimedOut: false},
		"test_subscribe_timeout": {domain: "mydomain", isTimedOut: true},
	}

	mockCtrl, mockPaho, mockToken := setupCommonMocks(t)
	defer mockCtrl.Finish()

	mockHandler := mocks.NewMockOwnerConsentHandler(mockCtrl)

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			client := &ownerConsentClient{
				mqttClient: newInternalClient(test.domain, mqttTestConfig, mockPaho),
				domain:     test.domain,
			}
			mockPaho.EXPECT().Subscribe(test.domain+"update/ownerconsent", uint8(1), gomock.Any()).Return(mockToken)
			setupMockToken(mockToken, mqttTestConfig.SubscribeTimeout, test.isTimedOut)

			assertOutgoingResult(t, test.isTimedOut, client.Start(mockHandler))
			if test.isTimedOut {
				assert.Nil(t, client.handler)
			} else {
				assert.Equal(t, mockHandler, client.handler)
			}
		})
	}
}

func TestOwnerConsentClientStop(t *testing.T) {
	tests := map[string]testCaseOutgoing{
		"test_unsubscribe_ok":      {domain: "testdomain", isTimedOut: false},
		"test_unsubscribe_timeout": {domain: "mydomain", isTimedOut: true},
	}

	mockCtrl, mockPaho, mockToken := setupCommonMocks(t)
	defer mockCtrl.Finish()

	mockHandler := mocks.NewMockOwnerConsentHandler(mockCtrl)

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			client := &ownerConsentClient{
				mqttClient: newInternalClient(test.domain, mqttTestConfig, mockPaho),
				domain:     test.domain,
				handler:    mockHandler,
			}
			mockPaho.EXPECT().Unsubscribe(test.domain + "update/ownerconsent").Return(mockToken)
			setupMockToken(mockToken, mqttTestConfig.UnsubscribeTimeout, test.isTimedOut)

			assertOutgoingResult(t, test.isTimedOut, client.Stop())
			if test.isTimedOut {
				assert.Equal(t, mockHandler, client.handler)
			} else {
				assert.Nil(t, client.handler)
			}
		})
	}
}

func TestSendOwnerConsentGet(t *testing.T) {
	tests := map[string]testCaseOutgoing{
		"test_send_owner_consent_get_ok":    {domain: "testdomain", isTimedOut: false},
		"test_send_owner_consent_get_error": {domain: "mydomain", isTimedOut: true},
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
			client, _ := NewOwnerConsentClient(test.domain, &updateAgentClient{
				mqttClient: newInternalClient("testDomain", mqttTestConfig, mockPaho),
			})
			mockPaho.EXPECT().Publish(test.domain+"update/ownerconsent/get", uint8(1), false, gomock.Any()).DoAndReturn(
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

			assertOutgoingResult(t, test.isTimedOut, client.SendOwnerConsentGet(name, testDesiredState))
		})
	}
}

func TestHandleOwnerConsentMessage(t *testing.T) {
	tests := map[string]testCaseIncoming{
		"test_handle_owner_consent_ok":         {domain: "testdomain", handlerError: nil, expectedJSONErr: false},
		"test_handle_owner_consent_error":      {domain: "mydomain", handlerError: errors.New("handler error"), expectedJSONErr: false},
		"test_handle_owner_consent_json_error": {domain: "testdomain", handlerError: nil, expectedJSONErr: true},
	}

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockMessage := clientsmocks.NewMockMessage(mockCtrl)

	testConsent := &types.OwnerConsent{
		Status: types.StatusApproved,
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			testBytes, expectedCalls := testBytesToEnvelope(t, name, testConsent, test.expectedJSONErr)

			handler := mocks.NewMockOwnerConsentHandler(mockCtrl)
			handler.EXPECT().HandleOwnerConsent(name, gomock.Any(), testConsent).Times(expectedCalls).Return(test.handlerError)

			client := &ownerConsentClient{
				mqttClient: newInternalClient(test.domain, &internalConnectionConfig{}, nil),
				domain:     test.domain,
				handler:    handler,
			}
			mockMessage.EXPECT().Topic().Return(test.domain + "update/ownerconsent")
			mockMessage.EXPECT().Payload().Return(testBytes)

			client.handleMessage(nil, mockMessage)
		})
	}
}
