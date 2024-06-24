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
	"github.com/eclipse-kanto/update-manager/test"
	"github.com/eclipse-kanto/update-manager/test/mocks"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestNewOwnerConsentThingsClient(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockPaho := clientsmocks.NewMockClient(mockCtrl)
	mockClient := mocks.NewMockUpdateAgentClient(mockCtrl)

	tests := map[string]struct {
		client api.UpdateAgentClient
		err    string
	}{
		"test_update_agent_things_client": {
			client: &updateAgentThingsClient{
				updateAgentClient: &updateAgentClient{
					mqttClient: newInternalClient("testDomain", &internalConnectionConfig{}, mockPaho),
				},
			},
		},
		"test_update_agent_client": {
			client: &updateAgentClient{
				mqttClient: newInternalClient("testDomain", &internalConnectionConfig{}, mockPaho),
			},
			err: fmt.Sprintf("unexpected type: %T", mockClient),
		},
		"test_error": {
			client: mockClient,
			err:    fmt.Sprintf("unexpected type: %T", mockClient),
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			client, err := NewOwnerConsentThingsClient("testDomain", test.client)
			if test.err != "" {
				assert.EqualError(t, err, fmt.Sprintf("unexpected type: %T", test.client))
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, client)
			}
		})
	}
}

func TestOwnerConsentThingsClientStart(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockFeature := mocks.NewMockUpdateManagerFeature(mockCtrl)
	mockHandler := mocks.NewMockOwnerConsentHandler(mockCtrl)
	client := &ownerConsentThingsClient{
		updateAgentThingsClient: &updateAgentThingsClient{
			umFeature: mockFeature,
		},
	}
	mockFeature.EXPECT().SetConsentHandler(mockHandler)
	assert.Nil(t, client.Start(mockHandler))
}

func TestOwnerConsentThingsClientStop(t *testing.T) {

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockFeature := mocks.NewMockUpdateManagerFeature(mockCtrl)
	client := &ownerConsentThingsClient{
		updateAgentThingsClient: &updateAgentThingsClient{
			umFeature: mockFeature,
		},
	}
	mockFeature.EXPECT().SetConsentHandler(nil)
	assert.Nil(t, client.Stop())
}

func TestSendOwnerConsentThings(t *testing.T) {
	tests := map[string]struct {
		err error
	}{
		"test_send_owner_consent_ok":    {},
		"test_send_owner_consent_error": {err: fmt.Errorf("test error")},
	}

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockFeature := mocks.NewMockUpdateManagerFeature(mockCtrl)
	client := &ownerConsentThingsClient{
		updateAgentThingsClient: &updateAgentThingsClient{
			umFeature: mockFeature,
		},
	}
	testConsent := &types.OwnerConsent{
		Command: types.CommandDownload,
	}

	for name, testCase := range tests {
		t.Run(name, func(t *testing.T) {
			mockFeature.EXPECT().SendConsent(test.ActivityID, testConsent).Return(testCase.err)
			assert.Equal(t, testCase.err, client.SendOwnerConsent(test.ActivityID, testConsent))
		})
	}
}
