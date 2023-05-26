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
	"github.com/eclipse-kanto/update-manager/logger"

	pahomqtt "github.com/eclipse/paho.mqtt.golang"
)

type desiredStateClient struct {
	*mqttClient
	domain       string
	stateHandler api.StateHandler
}

// NewDesiredStateClient instantiates a new client for triggering mqtt requests.
func NewDesiredStateClient(domain string, updateAgent api.UpdateAgentClient) api.DesiredStateClient {
	client := updateAgent.(*updateAgentClient)
	return &desiredStateClient{
		mqttClient: &mqttClient{
			mqttPrefix: domainAsTopic(domain),
			mqttConfig: client.mqttConfig,
			pahoClient: client.pahoClient,
		},
		domain: domain,
	}
}

func (client *desiredStateClient) topic(topicSuffix string) string {
	return client.mqttPrefix + topicSuffix
}

func (client *desiredStateClient) Domain() string {
	return client.domain
}

// Connect connects the client to the MQTT broker.
func (client *desiredStateClient) Subscribe(stateHandler api.StateHandler) error {
	client.stateHandler = stateHandler
	if err := client.subscribe(); err != nil {
		logger.ErrorErr(err, "[%s] error subscribing for CurrentState/DesiredStateFeedback responses", client.Domain())
		client.stateHandler = nil
		return err
	}
	logger.Debug("[%s] subscribed for CurrentState/DesiredStateFeedback responses", client.Domain())
	return nil
}

// Disconnect disconnects the client from the MQTT broker.
func (client *desiredStateClient) Unsubscribe() {
	if err := client.unsubscribe(); err != nil {
		logger.WarnErr(err, "[%s] error unsubscribing for DesiredStateFeedback/CurrentState requests", client.Domain())
	} else {
		logger.Debug("[%s] unsubscribed for DesiredStateFeedback/CurrentState requests", client.Domain())
		client.stateHandler = nil
	}
}

func (client *desiredStateClient) subscribe() error {
	topicFilters := make(map[string]byte)
	topicFilters[client.topic(suffixCurrentState)] = 1
	topicFilters[client.topic(suffixDesiredStateFeedback)] = 1
	logger.Debug("subscribing for '%v' topics", topicFilters)
	subscribeTimeout := convertToMilliseconds(client.mqttConfig.SubscribeTimeout)
	token := client.pahoClient.SubscribeMultiple(topicFilters, client.handleMessage)
	if !token.WaitTimeout(subscribeTimeout) {
		return fmt.Errorf("cannot subscribe for topics '%v' in '%v' seconds", topicFilters, subscribeTimeout)
	}
	return token.Error()
}

func (client *desiredStateClient) unsubscribe() error {
	topicCurrentState := client.topic(suffixCurrentState)
	topicDesiredStateFeedback := client.topic(suffixDesiredStateFeedback)
	logger.Debug("unsubscribing from '%s' & '%s' topics", topicCurrentState, topicDesiredStateFeedback)
	token := client.pahoClient.Unsubscribe(topicCurrentState, topicDesiredStateFeedback)
	unsubscribeTimeout := convertToMilliseconds(client.mqttConfig.UnsubscribeTimeout)
	if !token.WaitTimeout(unsubscribeTimeout) {
		return fmt.Errorf("cannot unsubscribe from topic '%s' & '%s' in '%v' seconds", topicCurrentState, topicDesiredStateFeedback, unsubscribeTimeout)
	}
	return token.Error()
}

func (client *desiredStateClient) handleMessage(mqttClient pahomqtt.Client, message pahomqtt.Message) {
	topic := message.Topic()
	logger.Debug("[%s] received %s message", client.Domain(), topic)
	if topic == client.topic(suffixCurrentState) {
		currentState := message.Payload()
		if err := client.stateHandler.HandleCurrentState(currentState); err != nil {
			logger.ErrorErr(err, "[%s] error processing current state message", client.Domain())
		}
	} else if topic == client.topic(suffixDesiredStateFeedback) {
		desiredStateFeedback := message.Payload()
		if err := client.stateHandler.HandleDesiredStateFeedback(desiredStateFeedback); err != nil {
			logger.ErrorErr(err, "[%s] error processing desired state feedback message", client.Domain())
		}
	}
}

func (client *desiredStateClient) PublishDesiredState(desiredState []byte) error {
	topicDesiredState := client.topic(suffixDesiredState)
	logger.Debug("publishing desired state to topic '%s'", topicDesiredState)
	token := client.pahoClient.Publish(topicDesiredState, 1, false, desiredState)
	logger.Debug("published desired state to topic '%s'", topicDesiredState)
	acknowledgeTimeout := convertToMilliseconds(client.mqttConfig.AcknowledgeTimeout)
	if !token.WaitTimeout(acknowledgeTimeout) {
		return fmt.Errorf("cannot publish to topic '%s' in '%v' seconds", topicDesiredState, acknowledgeTimeout)
	}
	return token.Error()
}

func (client *desiredStateClient) PublishDesiredStateCommand(desiredStateCommand []byte) error {
	topicDesiredStateCommand := client.topic(suffixDesiredStateCommand)
	logger.Debug("publishing desired state command to topic '%s'", topicDesiredStateCommand)
	token := client.pahoClient.Publish(topicDesiredStateCommand, 1, false, desiredStateCommand)
	logger.Debug("published desired state command to topic '%s'", topicDesiredStateCommand)
	acknowledgeTimeout := convertToMilliseconds(client.mqttConfig.AcknowledgeTimeout)
	if !token.WaitTimeout(acknowledgeTimeout) {
		return fmt.Errorf("cannot publish to topic '%s' in '%v' seconds", topicDesiredStateCommand, acknowledgeTimeout)
	}
	return token.Error()
}

func (client *desiredStateClient) PublishGetCurrentState(currentState []byte) error {
	topicCurrentStateGet := client.topic(suffixCurrentStateGet)
	logger.Debug("publishing get current state request to topic '%s'", topicCurrentStateGet)
	token := client.pahoClient.Publish(topicCurrentStateGet, 1, false, currentState)
	logger.Debug("published get current state request to topic '%s'", topicCurrentStateGet)
	acknowledgeTimeout := convertToMilliseconds(client.mqttConfig.AcknowledgeTimeout)
	if !token.WaitTimeout(acknowledgeTimeout) {
		return fmt.Errorf("cannot publish to topic '%s' in '%v' seconds", topicCurrentStateGet, acknowledgeTimeout)
	}
	return token.Error()
}
