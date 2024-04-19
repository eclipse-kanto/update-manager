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

type ownerConsentClient struct {
	*mqttClient
	domain  string
	handler api.OwnerConsentHandler
}

// NewOwnerConsentClient instantiates a new client for triggering MQTT requests.
func NewOwnerConsentClient(domain string, updateAgent api.UpdateAgentClient) (api.OwnerConsentClient, error) {
	mqttClient, err := getMQTTClient(updateAgent)
	if err != nil {
		return nil, err
	}
	return &ownerConsentClient{
		mqttClient: newInternalClient(domain, mqttClient.mqttConfig, mqttClient.pahoClient),
		domain:     domain,
	}, nil
}

func (client *ownerConsentClient) Domain() string {
	return client.domain
}

// Start makes a client subscription to the MQTT broker for the MQTT topics for consent.
func (client *ownerConsentClient) Start(consentHandler api.OwnerConsentHandler) error {
	client.handler = consentHandler
	if err := client.subscribe(); err != nil {
		client.handler = nil
		return fmt.Errorf("[%s] error subscribing for OwnerConsent messages: %w", client.Domain(), err)
	}
	logger.Debug("[%s] subscribed for OwnerConsent messages", client.Domain())
	return nil
}

// Stop removes the client subscription to the MQTT broker for the MQTT topics for owner consent.
func (client *ownerConsentClient) Stop() error {
	if err := client.unsubscribe(); err != nil {
		return fmt.Errorf("[%s] error unsubscribing for OwnerConsent messages: %w", client.Domain(), err)
	}
	logger.Debug("[%s] unsubscribed for OwnerConsent messages", client.Domain())
	client.handler = nil
	return nil
}

func (client *ownerConsentClient) subscribe() error {
	logger.Debug("subscribing for '%v' topic", client.topicOwnerConsent)
	token := client.pahoClient.Subscribe(client.topicOwnerConsent, 1, client.handleMessage)
	if !token.WaitTimeout(client.mqttConfig.SubscribeTimeout) {
		return fmt.Errorf("cannot subscribe for topic '%s' in '%v'", client.topicOwnerConsent, client.mqttConfig.SubscribeTimeout)
	}
	return token.Error()
}

func (client *ownerConsentClient) unsubscribe() error {
	logger.Debug("unsubscribing from '%s' topic", client.topicOwnerConsent)
	token := client.pahoClient.Unsubscribe(client.topicOwnerConsent)
	if !token.WaitTimeout(client.mqttConfig.UnsubscribeTimeout) {
		return fmt.Errorf("cannot unsubscribe from topic '%s' in '%v'", client.topicOwnerConsent, client.mqttConfig.UnsubscribeTimeout)
	}
	return token.Error()
}

func (client *ownerConsentClient) handleMessage(mqttClient pahomqtt.Client, message pahomqtt.Message) {
	topic := message.Topic()
	logger.Debug("[%s] received %s message", client.Domain(), topic)
	if topic == client.topicOwnerConsent {
		ownerConsent := &types.OwnerConsent{}
		envelope, err := types.FromEnvelope(message.Payload(), ownerConsent)
		if err != nil {
			logger.ErrorErr(err, "[%s] cannot parse owner consent message", client.Domain())
			return
		}
		if err := client.handler.HandleOwnerConsent(envelope.ActivityID, envelope.Timestamp, ownerConsent); err != nil {
			logger.ErrorErr(err, "[%s] error processing owner consent message", client.Domain())
		}
	}
}

func (client *ownerConsentClient) SendOwnerConsentGet(activityID string, desiredState *types.DesiredState) error {
	logger.Debug("publishing to topic '%s'", client.topicOwnerConsentGet)
	desiredStateBytes, err := types.ToEnvelope(activityID, desiredState)
	if err != nil {
		return errors.Wrapf(err, "cannot marshal owner consent get message for activity-id %s", activityID)
	}
	token := client.pahoClient.Publish(client.topicOwnerConsentGet, 1, false, desiredStateBytes)
	if !token.WaitTimeout(client.mqttConfig.AcknowledgeTimeout) {
		return fmt.Errorf("cannot publish to topic '%s' in '%v'", client.topicOwnerConsentGet, client.mqttConfig.AcknowledgeTimeout)
	}
	return token.Error()
}
