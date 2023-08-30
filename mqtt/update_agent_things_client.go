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
	"encoding/json"
	"fmt"

	"github.com/eclipse-kanto/update-manager/api"
	"github.com/eclipse-kanto/update-manager/api/types"
	"github.com/eclipse-kanto/update-manager/logger"
	"github.com/eclipse-kanto/update-manager/things"

	"github.com/eclipse/ditto-clients-golang"
	pahomqtt "github.com/eclipse/paho.mqtt.golang"
)

const (
	edgeResponseTopic = "edge/thing/response"
	edgeRequestTopic  = "edge/thing/request"
)

// edgeConfiguration represents local Edge Thing configuration. Its device, tenant and policy identifiers.
type edgeConfiguration struct {
	DeviceID string `json:"deviceId"`
	TenantID string `json:"tenantId"`
	PolicyID string `json:"policyId"`
}

type updateAgentThingsClient struct {
	*updateAgentClient
	dittoClient ditto.Client
	edgeConfig  *edgeConfiguration
	umFeature   things.UpdateManagerFeature
}

// NewUpdateAgentThingsClient instantiates a new UpdateAgentClient instance using the provided configuration options.
func NewUpdateAgentThingsClient(domain string, config *ConnectionConfig) api.UpdateAgentClient {
	internalConfig := newInternalConnectionConfig(config)
	client := &updateAgentThingsClient{
		updateAgentClient: &updateAgentClient{
			mqttClient: newInternalClient(domain, internalConfig, nil),
			domain:     domain,
		},
	}
	client.pahoClient = newClient(internalConfig, client.onConnect)
	return client
}

// Domain returns the name of the domain that is handled by this client.
func (client *updateAgentThingsClient) Domain() string {
	return client.domain
}

// Start connects the client to the MQTT broker and gets the edge configuration.
func (client *updateAgentThingsClient) Start(handler api.UpdateAgentHandler) error {
	client.handler = handler
	token := client.pahoClient.Connect()
	if !token.WaitTimeout(client.mqttConfig.ConnectTimeout) {
		return fmt.Errorf("[%s] connect timed out", client.Domain())
	}
	return token.Error()
}

func (client *updateAgentThingsClient) handleEdgeResponse(_ pahomqtt.Client, message pahomqtt.Message) {
	var (
		localCfg = &edgeConfiguration{}
		err      error
	)

	if err = json.Unmarshal(message.Payload(), localCfg); err != nil {
		logger.ErrorErr(err, "[%s] could not unmarshal edge configuration: %v", client.Domain(), message)
		return
	}
	if client.edgeConfig == nil || *localCfg != *client.edgeConfig {
		logger.Info("[%s] applying edge configuration: %v", client.Domain(), localCfg)
		if client.edgeConfig != nil {
			client.umFeature.Deactivate()
			client.dittoClient.Disconnect()
		}

		dittoConfig := ditto.NewConfiguration().
			WithAcknowledgeTimeout(client.mqttConfig.AcknowledgeTimeout).
			WithSubscribeTimeout(client.mqttConfig.SubscribeTimeout).
			WithUnsubscribeTimeout(client.mqttConfig.UnsubscribeTimeout).
			WithConnectHandler(func(dittoClient ditto.Client) {
				if err = client.umFeature.Activate(); err != nil {
					logger.ErrorErr(err, "[%s] could not activate update manager feature", client.Domain())
				} else {
					go getAndPublishCurrentState(client.Domain(), client.handler.HandleCurrentStateGet)
				}
			})

		if client.dittoClient, err = ditto.NewClientMQTT(client.pahoClient, dittoConfig); err != nil {
			logger.ErrorErr(err, "[%s] could not create ditto client", client.Domain())
			return
		}
		client.umFeature = things.NewUpdateManagerFeature(client.Domain(), localCfg.DeviceID, client.dittoClient, client.handler)

		if err = client.dittoClient.Connect(); err != nil {
			logger.ErrorErr(err, "[%s] could not connect to ditto endpoint", client.Domain())
			return
		}
		client.edgeConfig = localCfg
		logger.Info("[%s] edge configuration applied [TenantID: %s, DeviceID: %s, PolicyID: %s]", client.Domain(), localCfg.TenantID, localCfg.DeviceID, localCfg.PolicyID)
	}
}

// Stop disconnects the client from the MQTT broker.
func (client *updateAgentThingsClient) Stop() error {
	token := client.pahoClient.Unsubscribe(edgeResponseTopic)
	if !token.WaitTimeout(client.mqttConfig.UnsubscribeTimeout) {
		logger.Warn("[%s] cannot unsubscribe for topic '%s' in '%v'", client.Domain(), edgeResponseTopic, client.mqttConfig.UnsubscribeTimeout)
	} else if err := token.Error(); err != nil {
		logger.WarnErr(err, "[%s] error unsubscribing for topic '%s", client.Domain(), edgeResponseTopic)
	}
	if client.umFeature != nil {
		client.umFeature.Deactivate()
	}
	if client.dittoClient != nil {
		client.dittoClient.Disconnect()
	}

	client.pahoClient.Disconnect(disconnectQuiesce)
	client.handler = nil
	return nil
}

func (client *updateAgentThingsClient) onConnect(_ pahomqtt.Client) {
	token := client.pahoClient.Subscribe(edgeResponseTopic, 1, client.handleEdgeResponse)
	if !token.WaitTimeout(client.mqttConfig.SubscribeTimeout) {
		logger.Error("[%s] cannot subscribe for topic '%s' in '%v'", client.Domain(), edgeResponseTopic, client.mqttConfig.SubscribeTimeout)
		return
	}
	if token.Error() != nil {
		logger.ErrorErr(token.Error(), "[%s] cannot subscribe for topic '%s'", client.Domain(), edgeResponseTopic)
		return
	}

	token = client.pahoClient.Publish(edgeRequestTopic, 1, false, "")
	if !token.WaitTimeout(client.mqttConfig.AcknowledgeTimeout) {
		logger.Error("[%s] cannot publish to topic '%s' in '%v'", client.Domain(), edgeRequestTopic, client.mqttConfig.AcknowledgeTimeout)
	}
	if token.Error() != nil {
		logger.ErrorErr(token.Error(), "[%s] cannot publish to topic '%s'", client.Domain(), edgeRequestTopic)
	}
}

// SendCurrentState makes the client create envelope with the given activityID and current state inventory and updates the current state property of the feature.
func (client *updateAgentThingsClient) SendCurrentState(activityID string, currentState *types.Inventory) error {

	logger.Debug("[%s] publishing current state...", client.Domain())

	return client.umFeature.SetState(activityID, currentState)
}

// SendDesiredStateFeedback makes the client create envelope with the given activityID and desired state feedback and send issues a desired state feedback message.
func (client *updateAgentThingsClient) SendDesiredStateFeedback(activityID string, desiredStateFeedback *types.DesiredStateFeedback) error {
	return client.umFeature.SendFeedback(activityID, desiredStateFeedback)
}
