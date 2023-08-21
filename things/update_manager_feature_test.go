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

package things

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/eclipse-kanto/update-manager/api/types"
	"github.com/eclipse-kanto/update-manager/test/mocks"
	"github.com/eclipse/ditto-clients-golang/model"
	"github.com/eclipse/ditto-clients-golang/protocol"
	"github.com/eclipse/ditto-clients-golang/protocol/things"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

const (
	testDomain     = "testDomain"
	testActivityID = "testActivityID"
	outboxPathFmt  = "/features/UpdateManager/outbox/messages/%s"
)

var (
	tesThingID = model.NewNamespacedIDFrom("namespace:testDevice")
	testWG     = &sync.WaitGroup{}
	errTest    = fmt.Errorf("test error")
)

func TestActivate(t *testing.T) {
	tests := map[string]struct {
		feature       *updateManagerFeature
		mockExecution func(*mocks.MockClient) error
	}{
		"test_activate_ok": {
			feature: &updateManagerFeature{thingID: tesThingID, domain: testDomain},
			mockExecution: func(mockDittoClient *mocks.MockClient) error {
				mockDittoClient.EXPECT().Subscribe(gomock.Any())
				mockDittoClient.EXPECT().Send(gomock.AssignableToTypeOf(&protocol.Envelope{})).DoAndReturn(func(message *protocol.Envelope) error {
					assert.False(t, message.Headers.IsResponseRequired())
					assertTwinCommandTopic(t, *tesThingID, message.Topic)
					assert.Equal(t, "/features/UpdateManager", message.Path)
					feature := message.Value.(*model.Feature)
					assert.Equal(t, updateManagerFeatureDefinition, feature.Definition[0].String())
					assert.Equal(t, testDomain, feature.Properties["domain"])
					return nil
				})
				return nil
			},
		},
		"test_activate_already_activated": {
			feature: &updateManagerFeature{active: true},
			mockExecution: func(_ *mocks.MockClient) error {
				return nil
			},
		},
		"test_activate_error": {
			feature: &updateManagerFeature{thingID: tesThingID, domain: testDomain},
			mockExecution: func(mockDittoClient *mocks.MockClient) error {
				mockDittoClient.EXPECT().Subscribe(gomock.Any())
				mockDittoClient.EXPECT().Send(gomock.AssignableToTypeOf(&protocol.Envelope{})).Return(errTest)
				mockDittoClient.EXPECT().Unsubscribe()
				return errTest
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			mockCtrl, mockDittoClient, _ := setupMocks(t, test.feature)
			defer mockCtrl.Finish()

			expectedError := test.mockExecution(mockDittoClient)
			actualError := test.feature.Activate()
			if expectedError != nil {
				assert.EqualError(t, actualError, expectedError.Error())
				assert.False(t, test.feature.active)
			} else {
				assert.Nil(t, actualError)
				assert.True(t, test.feature.active)
			}
		})
	}
}

func TestDeactivate(t *testing.T) {
	tests := map[string]struct {
		feature       *updateManagerFeature
		mockExecution func(*mocks.MockClient)
	}{
		"test_deactivate_ok": {
			feature: &updateManagerFeature{active: true},
			mockExecution: func(mockDittoClient *mocks.MockClient) {
				mockDittoClient.EXPECT().Unsubscribe()
			},
		},
		"test_deactivate_already_deactivated": {
			feature:       &updateManagerFeature{},
			mockExecution: func(_ *mocks.MockClient) {},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			mockCtrl, mockDittoClient, _ := setupMocks(t, test.feature)
			defer mockCtrl.Finish()

			test.mockExecution(mockDittoClient)
			test.feature.Deactivate()

			assert.False(t, test.feature.active)
		})
	}
}

func TestSetState(t *testing.T) {
	testInventory := &types.Inventory{}
	tests := map[string]struct {
		feature       *updateManagerFeature
		mockExecution func(*mocks.MockClient) error
	}{
		"test_set_state_ok": {
			feature: &updateManagerFeature{active: true, thingID: tesThingID, domain: testDomain},
			mockExecution: func(mockDittoClient *mocks.MockClient) error {
				mockDittoClient.EXPECT().Send(gomock.AssignableToTypeOf(&protocol.Envelope{})).DoAndReturn(func(message *protocol.Envelope) error {
					assert.False(t, message.Headers.IsResponseRequired())
					assertTwinCommandTopic(t, *tesThingID, message.Topic)
					assert.Equal(t, "/features/UpdateManager/properties", message.Path)
					properties := message.Value.(*updateManagerProperties)
					assert.Equal(t, testDomain, properties.Domain)
					assert.Equal(t, testActivityID, properties.ActivityID)
					assert.Equal(t, testInventory, properties.Inventory)
					assert.True(t, properties.Timestamp > 0)
					return nil
				})
				return nil
			},
		},
		"test_set_state_not_active": {
			feature: &updateManagerFeature{active: false},
			mockExecution: func(_ *mocks.MockClient) error {
				return nil
			},
		},
		"test_set_state_error": {
			feature: &updateManagerFeature{active: true, thingID: tesThingID, domain: testDomain},
			mockExecution: func(mockDittoClient *mocks.MockClient) error {
				mockDittoClient.EXPECT().Send(gomock.AssignableToTypeOf(&protocol.Envelope{})).Return(errTest)
				return errTest
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			mockCtrl, mockDittoClient, _ := setupMocks(t, test.feature)
			defer mockCtrl.Finish()

			expectedError := test.mockExecution(mockDittoClient)
			actualError := test.feature.SetState(testActivityID, testInventory)
			if expectedError != nil {
				assert.EqualError(t, actualError, expectedError.Error())
			} else {
				assert.Nil(t, actualError)
			}
		})
	}
}

func TestSendFeedback(t *testing.T) {
	testFeedback := &types.DesiredStateFeedback{}
	tests := map[string]struct {
		feature       *updateManagerFeature
		mockExecution func(*mocks.MockClient) error
	}{
		"test_send_feedback_ok": {
			feature: &updateManagerFeature{active: true, thingID: tesThingID, domain: testDomain},
			mockExecution: func(mockDittoClient *mocks.MockClient) error {
				mockDittoClient.EXPECT().Send(gomock.AssignableToTypeOf(&protocol.Envelope{})).DoAndReturn(func(message *protocol.Envelope) error {
					assert.False(t, message.Headers.IsResponseRequired())
					assertLiveMessageTopic(t, *tesThingID, updateManagerFeatureMessageFeedback, message.Topic)
					assert.Equal(t, fmt.Sprintf(outboxPathFmt, updateManagerFeatureMessageFeedback), message.Path)
					feedback := message.Value.(*feedback)
					assert.Equal(t, testActivityID, feedback.ActivityID)
					assert.Equal(t, testFeedback, feedback.DesiredStateFeedback)
					assert.True(t, feedback.Timestamp > 0)
					return nil
				})
				return nil
			},
		},
		"test_send_feedback_not_active": {
			feature: &updateManagerFeature{active: false},
			mockExecution: func(_ *mocks.MockClient) error {
				return nil
			},
		},
		"test_send_feedback_error": {
			feature: &updateManagerFeature{active: true, thingID: tesThingID, domain: testDomain},
			mockExecution: func(mockDittoClient *mocks.MockClient) error {
				mockDittoClient.EXPECT().Send(gomock.AssignableToTypeOf(&protocol.Envelope{})).Return(errTest)
				return errTest
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			mockCtrl, mockDittoClient, _ := setupMocks(t, test.feature)
			defer mockCtrl.Finish()

			expectedError := test.mockExecution(mockDittoClient)
			actualError := test.feature.SendFeedback(testActivityID, testFeedback)
			if expectedError != nil {
				assert.EqualError(t, actualError, expectedError.Error())
			} else {
				assert.Nil(t, actualError)
			}
		})
	}
}

func TestMessageHandler(t *testing.T) {
	testDesiredState := &types.DesiredState{}
	testRequestID := "testRequestID"
	mockThingExecution := func(operation string) func(*mocks.MockClient, *mocks.MockUpdateAgentHandler) {
		return func(mockDittoClient *mocks.MockClient, mockHandler *mocks.MockUpdateAgentHandler) {
			mockDittoClient.EXPECT().Reply(testRequestID, gomock.AssignableToTypeOf(&protocol.Envelope{})).DoAndReturn(
				func(_ string, message *protocol.Envelope) error {
					assert.False(t, message.Headers.IsResponseRequired())
					assert.Equal(t, 204, message.Status)
					assertLiveMessageTopic(t, *tesThingID, protocol.TopicAction(operation), message.Topic)
					assert.Equal(t, fmt.Sprintf(outboxPathFmt, operation), message.Path)
					assert.Nil(t, message.Value)
					return nil
				})
			testWG.Add(1)
			switch operation {
			case updateManagerFeatureOperationRefresh:
				mockHandler.EXPECT().HandleCurrentStateGet(testActivityID, gomock.Any()).DoAndReturn(func(activityID string, timestamp int64) error {
					testWG.Done()
					return nil
				})
			case updateManagerFeatureOperationApply:
				mockHandler.EXPECT().HandleDesiredState(testActivityID, gomock.Any(), gomock.Any()).DoAndReturn(func(activityID string, timestamp int64, ds *types.DesiredState) error {
					assert.Equal(t, testDesiredState, ds)
					testWG.Done()
					return nil
				})
			default:
				testWG.Done()
			}
		}
	}

	mockThingErrorExecution := func(operation string) func(*mocks.MockClient, *mocks.MockUpdateAgentHandler) {
		return func(mockDittoClient *mocks.MockClient, _ *mocks.MockUpdateAgentHandler) {
			mockDittoClient.EXPECT().Reply(testRequestID, gomock.AssignableToTypeOf(&protocol.Envelope{})).DoAndReturn(
				func(_ string, message *protocol.Envelope) error {
					assert.False(t, message.Headers.IsResponseRequired())
					assert.Equal(t, responseStatusBadRequest, message.Status)
					assertLiveMessageTopic(t, *tesThingID, protocol.TopicAction(operation), message.Topic)
					assert.Equal(t, fmt.Sprintf(outboxPathFmt, operation), message.Path)
					thingError := message.Value.(*thingError)
					assert.Equal(t, messagesParameterInvalid, thingError.ErrorCode)
					assert.Equal(t, responseStatusBadRequest, thingError.Status)
					assert.NotEmpty(t, thingError.Message)
					return nil
				})
		}
	}

	tests := map[string]struct {
		feature       *updateManagerFeature
		envelope      *protocol.Envelope
		mockExecution func(*mocks.MockClient, *mocks.MockUpdateAgentHandler)
	}{
		"test_message_handler_not_active": {
			feature:       &updateManagerFeature{},
			mockExecution: func(_ *mocks.MockClient, _ *mocks.MockUpdateAgentHandler) {},
		},
		"test_message_handler_unexpected_command": {
			feature:       &updateManagerFeature{active: true, thingID: tesThingID},
			envelope:      things.NewMessage(tesThingID).Feature(updateManagerFeatureID).Inbox("unexpected").Envelope(),
			mockExecution: func(_ *mocks.MockClient, _ *mocks.MockUpdateAgentHandler) {},
		},
		"test_message_handler_unexpected_thing_id": {
			feature:       &updateManagerFeature{active: true, thingID: tesThingID},
			envelope:      things.NewMessage(model.NewNamespacedIDFrom("ns:unexpected")).Feature(updateManagerFeatureID).Inbox("unexpected").Envelope(),
			mockExecution: func(_ *mocks.MockClient, _ *mocks.MockUpdateAgentHandler) {},
		},
		"test_message_handler_refresh_ok": {
			feature: &updateManagerFeature{active: true, thingID: tesThingID},
			envelope: things.NewMessage(tesThingID).Feature(updateManagerFeatureID).Inbox(updateManagerFeatureOperationRefresh).WithPayload(&base{ActivityID: testActivityID}).
				Envelope(protocol.WithResponseRequired(true)),
			mockExecution: mockThingExecution(updateManagerFeatureOperationRefresh),
		},
		"test_message_handler_refresh_error": {
			feature: &updateManagerFeature{active: true, thingID: tesThingID},
			envelope: things.NewMessage(tesThingID).Feature(updateManagerFeatureID).Inbox(updateManagerFeatureOperationRefresh).WithPayload("invalid payload").
				Envelope(protocol.WithResponseRequired(true)),
			mockExecution: mockThingErrorExecution(updateManagerFeatureOperationRefresh),
		},
		"test_message_handler_apply_ok": {
			feature: &updateManagerFeature{active: true, thingID: tesThingID},
			envelope: things.NewMessage(tesThingID).Feature(updateManagerFeatureID).Inbox(updateManagerFeatureOperationApply).WithPayload(&applyArgs{base: base{ActivityID: testActivityID}, DesiredState: &types.DesiredState{}}).
				Envelope(protocol.WithResponseRequired(true)),
			mockExecution: mockThingExecution(updateManagerFeatureOperationApply),
		},
		"test_message_handler_apply_error": {
			feature: &updateManagerFeature{active: true, thingID: tesThingID},
			envelope: things.NewMessage(tesThingID).Feature(updateManagerFeatureID).Inbox(updateManagerFeatureOperationApply).WithPayload("invalid payload").
				Envelope(protocol.WithResponseRequired(true)),
			mockExecution: mockThingErrorExecution(updateManagerFeatureOperationApply),
		},
		"test_message_handler_apply_nil_desire_state_error": {
			feature: &updateManagerFeature{active: true, thingID: tesThingID},
			envelope: things.NewMessage(tesThingID).Feature(updateManagerFeatureID).Inbox(updateManagerFeatureOperationApply).WithPayload(&applyArgs{base: base{ActivityID: testActivityID}}).
				Envelope(protocol.WithResponseRequired(true)),
			mockExecution: mockThingErrorExecution(updateManagerFeatureOperationApply),
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			mockCtrl, mockDittoClient, mockHandler := setupMocks(t, test.feature)
			defer mockCtrl.Finish()

			test.mockExecution(mockDittoClient, mockHandler)
			test.feature.messagesHandler(testRequestID, test.envelope)

			testWaitChan := make(chan struct{})
			testTimeout := 2 * time.Second
			go func() {
				defer close(testWaitChan)
				testWG.Wait()
			}()
			select {
			case <-testWaitChan:
				return // completed normally
			case <-time.After(testTimeout):
				t.Fatal("timed out waiting for ", testTimeout)
			}
		})
	}
}

func assertTwinCommandTopic(t *testing.T, tesThingID model.NamespacedID, topic *protocol.Topic) {
	expectedTopic := (&protocol.Topic{}).
		WithNamespace(tesThingID.Namespace).
		WithEntityName(tesThingID.Name).
		WithGroup(protocol.GroupThings).
		WithChannel(protocol.ChannelTwin).
		WithCriterion(protocol.CriterionCommands).
		WithAction(protocol.ActionModify)

	assert.Equal(t, expectedTopic, topic)
}

func assertLiveMessageTopic(t *testing.T, tesThingID model.NamespacedID, operation protocol.TopicAction, topic *protocol.Topic) {
	expectedTopic := (&protocol.Topic{}).
		WithNamespace(tesThingID.Namespace).
		WithEntityName(tesThingID.Name).
		WithGroup(protocol.GroupThings).
		WithChannel(protocol.ChannelLive).
		WithCriterion(protocol.CriterionMessages).
		WithAction(operation)

	assert.Equal(t, expectedTopic, topic)
}

func setupMocks(t *testing.T, feature *updateManagerFeature) (*gomock.Controller, *mocks.MockClient, *mocks.MockUpdateAgentHandler) {
	mockCtrl := gomock.NewController(t)
	mockDittoClient := mocks.NewMockClient(mockCtrl)
	mockHandler := mocks.NewMockUpdateAgentHandler(mockCtrl)
	feature.dittoClient = mockDittoClient
	feature.handler = mockHandler
	return mockCtrl, mockDittoClient, mockHandler
}
