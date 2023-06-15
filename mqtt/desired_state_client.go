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

	"github.com/eclipse-kanto/update-manager/api"
	"github.com/eclipse-kanto/update-manager/api/types"
	"github.com/eclipse-kanto/update-manager/logger"

	pahomqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/pkg/errors"
)

type desiredStateClient struct {
	*mqttClient
	domain       string
	stateHandler api.StateHandler
}

// NewDesiredStateClient instantiates a new client for triggering MQTT requests.
func NewDesiredStateClient(domain string, updateAgent api.UpdateAgentClient) api.DesiredStateClient {
	mqttClient := updateAgent.(*updateAgentClient).mqttClient
	return &desiredStateClient{
		mqttClient: newInternalClient(domain, mqttClient.mqttConfig, mqttClient.pahoClient),
		domain:     domain,
	}
}

func (client *desiredStateClient) Domain() string {
	return client.domain
}

// Subscribe makes a client subscription to the MQTT broker for the MQTT topics for desired state feedback and current state messages.
func (client *desiredStateClient) Start(stateHandler api.StateHandler) error {
	client.stateHandler = stateHandler
	if err := client.subscribe(); err != nil {
		client.stateHandler = nil
		return fmt.Errorf("[%s] error subscribing for CurrentState/DesiredStateFeedback messages: %w", client.Domain(), err)
	}
	logger.Debug("[%s] subscribed for CurrentState/DesiredStateFeedback messages", client.Domain())
	return nil
}

// Unsubscribe removes the client subscription to the MQTT broker for the MQTT topics for desired state feedback and current state messages.
func (client *desiredStateClient) Stop() error {
	if err := client.unsubscribe(); err != nil {
		return fmt.Errorf("[%s] error unsubscribing for DesiredStateFeedback/CurrentState messages: %w", client.Domain(), err)
	}
	logger.Debug("[%s] unsubscribed for DesiredStateFeedback/CurrentState messages", client.Domain())
	client.stateHandler = nil
	return nil
}

func (client *desiredStateClient) subscribe() error {
	topicFilters := make(map[string]byte)
	topicFilters[client.topicCurrentState] = 1
	topicFilters[client.topicDesiredStateFeedback] = 1
	logger.Debug("subscribing for '%v' topics", topicFilters)
	subscribeTimeout := convertToMilliseconds(client.mqttConfig.SubscribeTimeout)
	token := client.pahoClient.SubscribeMultiple(topicFilters, client.handleMessage)
	if !token.WaitTimeout(subscribeTimeout) {
		return fmt.Errorf("cannot subscribe for topics '%v' in '%v' seconds", topicFilters, subscribeTimeout)
	}
	return token.Error()
}

func (client *desiredStateClient) unsubscribe() error {
	logger.Debug("unsubscribing from '%s' & '%s' topics", client.topicCurrentState, client.topicDesiredStateFeedback)
	token := client.pahoClient.Unsubscribe(client.topicCurrentState, client.topicDesiredStateFeedback)
	unsubscribeTimeout := convertToMilliseconds(client.mqttConfig.UnsubscribeTimeout)
	if !token.WaitTimeout(unsubscribeTimeout) {
		return fmt.Errorf("cannot unsubscribe from topic '%s' & '%s' in '%v' seconds", client.topicCurrentState, client.topicDesiredStateFeedback, unsubscribeTimeout)
	}
	return token.Error()
}

func (client *desiredStateClient) handleMessage(mqttClient pahomqtt.Client, message pahomqtt.Message) {
	topic := message.Topic()
	logger.Debug("[%s] received %s message", client.Domain(), topic)
	if topic == client.topicCurrentState {
		currentState := &types.Inventory{}
		envelope, err := types.FromEnvelope(message.Payload(), currentState)
		if err != nil {
			logger.ErrorErr(err, "[%s] cannot parse current state message", client.Domain())
			return
		}
		if err := client.stateHandler.HandleCurrentState(envelope.ActivityID, envelope.Timestamp, currentState); err != nil {
			logger.ErrorErr(err, "[%s] error processing current state message", client.Domain())
		}
	} else if topic == client.topicDesiredStateFeedback {
		desiredstatefeedback := &types.DesiredStateFeedback{}
		envelope, err := types.FromEnvelope(message.Payload(), desiredstatefeedback)
		if err != nil {
			logger.ErrorErr(err, "[%s] cannot parse desired state feedback message", client.Domain())
			return
		}
		if err := client.stateHandler.HandleDesiredStateFeedback(envelope.ActivityID, envelope.Timestamp, desiredstatefeedback); err != nil {
			logger.ErrorErr(err, "[%s] error processing desired state feedback message", client.Domain())
		}
	}
}

func (client *desiredStateClient) SendDesiredState(activityID string, desiredState *types.DesiredState) error {
	logger.Debug("publishing desired state to topic '%s'", client.topicDesiredState)
	desiredStateBytes, err := types.ToEnvelope(activityID, desiredState)
	if err != nil {
		return errors.Wrapf(err, "cannot marshal desired state message for activity-id %s", activityID)
	}
	token := client.pahoClient.Publish(client.topicDesiredState, 1, false, desiredStateBytes)
	acknowledgeTimeout := convertToMilliseconds(client.mqttConfig.AcknowledgeTimeout)
	if !token.WaitTimeout(acknowledgeTimeout) {
		return fmt.Errorf("cannot publish to topic '%s' in '%v' seconds", client.topicDesiredState, acknowledgeTimeout)
	}
	return token.Error()
}

func (client *desiredStateClient) SendDesiredStateCommand(activityID string, desiredStateCommand *types.DesiredStateCommand) error {
	logger.Debug("publishing desired state command to topic '%s'", client.topicDesiredStateCommand)
	desiredStateCommandBytes, err := types.ToEnvelope(activityID, desiredStateCommand)
	if err != nil {
		return errors.Wrapf(err, "cannot marshal desired state command message for activity-id %s", activityID)
	}
	token := client.pahoClient.Publish(client.topicDesiredStateCommand, 1, false, desiredStateCommandBytes)
	acknowledgeTimeout := convertToMilliseconds(client.mqttConfig.AcknowledgeTimeout)
	if !token.WaitTimeout(acknowledgeTimeout) {
		return fmt.Errorf("cannot publish to topic '%s' in '%v' seconds", client.topicDesiredStateCommand, acknowledgeTimeout)
	}
	return token.Error()
}

func (client *desiredStateClient) SendCurrentStateGet(activityID string) error {
	logger.Debug("publishing get current state request to topic '%s'", client.topicCurrentStateGet)
	currentStateGetBytes, err := types.ToEnvelope(activityID, nil)
	if err != nil {
		return errors.Wrapf(err, "cannot marshal current state get message for activity-id %s", activityID)
	}
	token := client.pahoClient.Publish(client.topicCurrentStateGet, 1, false, currentStateGetBytes)
	acknowledgeTimeout := convertToMilliseconds(client.mqttConfig.AcknowledgeTimeout)
	if !token.WaitTimeout(acknowledgeTimeout) {
		return fmt.Errorf("cannot publish to topic '%s' in '%v' seconds", client.topicCurrentStateGet, acknowledgeTimeout)
	}
	return token.Error()
}
