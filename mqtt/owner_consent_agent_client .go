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

	"github.com/eclipse-kanto/update-manager/api"
	"github.com/eclipse-kanto/update-manager/api/types"
	"github.com/eclipse-kanto/update-manager/logger"

	pahomqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/pkg/errors"
)

type ownerConsentAgentClient struct {
	*mqttClient
	domain  string
	handler api.OwnerConsentAgentHandler
}

// NewOwnerConsentAgentClient instantiates a new client for triggering MQTT requests.
func NewOwnerConsentAgentClient(domain string, config *ConnectionConfig) (api.OwnerConsentAgentClient, error) {
	client := &ownerConsentAgentClient{
		mqttClient: newInternalClient(domain, newInternalConnectionConfig(config), nil),
		domain:     domain,
	}
	pahoClient, err := newClient(client.mqttConfig, client.onConnect)
	if err == nil {
		client.pahoClient = pahoClient
	}
	return client, err
}

func (client *ownerConsentAgentClient) onConnect(_ pahomqtt.Client) {
	if err := client.subscribe(); err != nil {
		logger.ErrorErr(err, "[%s] error subscribing for OwnerConsentGet requests", client.Domain())
	} else {
		logger.Debug("[%s] subscribed for OwnerConsentGet requests", client.Domain())
	}
}

// Start connects the client to the MQTT broker.
func (client *ownerConsentAgentClient) Start(handler api.OwnerConsentAgentHandler) error {
	client.handler = handler
	token := client.pahoClient.Connect()
	if !token.WaitTimeout(client.mqttConfig.ConnectTimeout) {
		return fmt.Errorf("[%s] connect timed out", client.Domain())
	}
	return token.Error()
}

func (client *ownerConsentAgentClient) Domain() string {
	return client.domain
}

// Stop removes the client subscription to the MQTT broker for the MQTT topics for getting owner consent.
func (client *ownerConsentAgentClient) Stop() error {
	if err := client.unsubscribe(); err != nil {
		logger.WarnErr(err, "[%s] error unsubscribing for OwnerConsentGet requests", client.Domain())
	} else {
		logger.Debug("[%s] unsubscribed for OwnerConsentGet messages", client.Domain())
	}
	client.pahoClient.Disconnect(disconnectQuiesce)
	client.handler = nil
	return nil
}

func (client *ownerConsentAgentClient) subscribe() error {
	logger.Debug("subscribing for '%v' topic", client.topicOwnerConsentGet)
	token := client.pahoClient.Subscribe(client.topicOwnerConsentGet, 1, client.handleMessage)
	if !token.WaitTimeout(client.mqttConfig.SubscribeTimeout) {
		return fmt.Errorf("cannot subscribe for topic '%s' in '%v'", client.topicOwnerConsentGet, client.mqttConfig.SubscribeTimeout)
	}
	return token.Error()
}

func (client *ownerConsentAgentClient) unsubscribe() error {
	logger.Debug("unsubscribing from '%s' topic", client.topicOwnerConsentGet)
	token := client.pahoClient.Unsubscribe(client.topicOwnerConsentGet)
	if !token.WaitTimeout(client.mqttConfig.UnsubscribeTimeout) {
		return fmt.Errorf("cannot unsubscribe from topic '%s' in '%v'", client.topicOwnerConsentGet, client.mqttConfig.UnsubscribeTimeout)
	}
	return token.Error()
}

func (client *ownerConsentAgentClient) handleMessage(mqttClient pahomqtt.Client, message pahomqtt.Message) {
	topic := message.Topic()
	logger.Debug("[%s] received %s message", client.Domain(), topic)
	if topic == client.topicOwnerConsentGet {
		consent := &types.OwnerConsent{}
		envelope, err := types.FromEnvelope(message.Payload(), consent)
		if err != nil {
			logger.ErrorErr(err, "[%s] cannot parse owner conset get message", client.Domain())
			return
		}
		if err := client.handler.HandleOwnerConsentGet(envelope.ActivityID, envelope.Timestamp, consent); err != nil {
			logger.ErrorErr(err, "[%s] error processing owner consent get message", client.Domain())
		}
	}
}

func (client *ownerConsentAgentClient) SendOwnerConsent(activityID string, consent *types.OwnerConsent) error {
	logger.Debug("publishing to topic '%s'", client.topicOwnerConsent)
	desiredStateBytes, err := types.ToEnvelope(activityID, consent)
	if err != nil {
		return errors.Wrapf(err, "cannot marshal owner consent message for activity-id %s", activityID)
	}
	token := client.pahoClient.Publish(client.topicOwnerConsent, 1, false, desiredStateBytes)
	if !token.WaitTimeout(client.mqttConfig.AcknowledgeTimeout) {
		return fmt.Errorf("cannot publish to topic '%s' in '%v'", client.topicOwnerConsent, client.mqttConfig.AcknowledgeTimeout)
	}
	return token.Error()
}
