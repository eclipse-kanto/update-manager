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
	"strconv"
	"strings"
	"time"

	"github.com/eclipse-kanto/update-manager/api"
	"github.com/eclipse-kanto/update-manager/api/types"
	"github.com/eclipse-kanto/update-manager/logger"

	pahomqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/google/uuid"
	"github.com/pkg/errors"
)

const prefixInitCurrentStateID = "initial-current-state-"

const (
	suffixDesiredState         = "/desiredstate"
	suffixDesiredStateCommand  = "/desiredstate/command"
	suffixCurrentState         = "/currentstate"
	suffixCurrentStateGet      = "/currentstate/get"
	suffixDesiredStateFeedback = "/desiredstatefeedback"

	disconnectQuiesce uint = 10000
)

type mqttClient struct {
	mqttConfig *ConnectionConfig
	pahoClient pahomqtt.Client

	// incoming topics
	topicCurrentState         string
	topicDesiredStateFeedback string
	// outgoing topics
	topicDesiredState        string
	topicDesiredStateCommand string
	topicCurrentStateGet     string
}

func newInternalClient(domain string, config *ConnectionConfig, pahoClient pahomqtt.Client) *mqttClient {
	mqttPrefix := domainAsTopic(domain)
	return &mqttClient{
		mqttConfig: config,
		pahoClient: pahoClient,

		topicCurrentState:         mqttPrefix + suffixCurrentState,
		topicCurrentStateGet:      mqttPrefix + suffixCurrentStateGet,
		topicDesiredState:         mqttPrefix + suffixDesiredState,
		topicDesiredStateCommand:  mqttPrefix + suffixDesiredStateCommand,
		topicDesiredStateFeedback: mqttPrefix + suffixDesiredStateFeedback,
	}
}

type updateAgentClient struct {
	*mqttClient
	domain  string
	handler api.UpdateAgentHandler
}

// NewUpdateAgentClient instantiates a new UpdateAgentClient instance using the provided configuration options.
func NewUpdateAgentClient(domain string, config *ConnectionConfig) api.UpdateAgentClient {
	client := &updateAgentClient{
		mqttClient: newInternalClient(domain, config, nil),
		domain:     domain,
	}
	client.pahoClient = newClient(config, client.onConnect)
	return client
}

// Domain returns the name of the domain that is handled by this client.
func (client *updateAgentClient) Domain() string {
	return client.domain
}

// Start connects the client to the MQTT broker.
func (client *updateAgentClient) Start(handler api.UpdateAgentHandler) error {
	client.handler = handler
	connectTimeout := convertToMilliseconds(client.mqttConfig.ConnectTimeout)
	token := client.pahoClient.Connect()
	if !token.WaitTimeout(connectTimeout) {
		return fmt.Errorf("[%s] connect timed out", client.Domain())
	}
	return token.Error()
}

// Stop disconnects the client from the MQTT broker.
func (client *updateAgentClient) Stop() error {
	if err := client.unsubscribeStateTopics(); err != nil {
		logger.WarnErr(err, "[%s] error unsubscribing for DesiredState/DesiredStateCommand/CurrentStateGet requests", client.Domain())
	} else {
		logger.Debug("[%s] unsubscribed for DesiredState/DesiredStateCommand/CurrentStateGet requests", client.Domain())
	}
	client.pahoClient.Disconnect(disconnectQuiesce)
	client.handler = nil
	return nil
}

func (client *updateAgentClient) onConnect(mqttClient pahomqtt.Client) {
	go client.getAndPublishCurrentState()

	if err := client.subscribeStateTopics(); err != nil {
		logger.ErrorErr(err, "[%s] error subscribing for DesiredState/DesiredStateCommand/CurrentStateGet requests", client.Domain())
	} else {
		logger.Debug("[%s] subscribed for DesiredState/DesiredStateCommand/CurrentStateGet requests", client.Domain())
	}
}

func (client *updateAgentClient) subscribeStateTopics() error {
	topics := []string{client.topicDesiredState, client.topicDesiredStateCommand, client.topicCurrentStateGet}
	topicsMap := map[string]byte{
		client.topicDesiredState:        1,
		client.topicDesiredStateCommand: 1,
		client.topicCurrentStateGet:     1,
	}
	logger.Debug("subscribing for '%s' topics", topics)
	subscribeTimeout := convertToMilliseconds(client.mqttConfig.SubscribeTimeout)
	token := client.pahoClient.SubscribeMultiple(topicsMap, client.handleStateRequest)
	if !token.WaitTimeout(subscribeTimeout) {
		return fmt.Errorf("cannot subscribe for topics '%s,%s,%s' in '%v' seconds", client.topicDesiredState, client.topicDesiredStateCommand, client.topicCurrentStateGet, subscribeTimeout)
	}
	return token.Error()
}

func (client *updateAgentClient) unsubscribeStateTopics() error {
	topics := []string{client.topicDesiredState, client.topicDesiredStateCommand, client.topicCurrentStateGet}
	logger.Debug("unsubscribing from '%s' topics", client.Domain(), topics)
	token := client.pahoClient.Unsubscribe(topics...)
	unsubscribeTimeout := convertToMilliseconds(client.mqttConfig.UnsubscribeTimeout)
	if !token.WaitTimeout(unsubscribeTimeout) {
		return fmt.Errorf("cannot unsubscribe from topics '%s,%s,%s' in '%v' seconds", client.topicDesiredState, client.topicDesiredStateCommand, client.topicCurrentStateGet, unsubscribeTimeout)
	}
	return token.Error()
}

func (client *updateAgentClient) handleStateRequest(mqttClient pahomqtt.Client, message pahomqtt.Message) {
	topic := message.Topic()
	if topic == client.topicDesiredState {
		logger.Debug("[%s] received desired state request", client.Domain())
		desiredState := &types.DesiredState{}
		envelope, err := types.FromEnvelope(message.Payload(), desiredState)
		if err != nil {
			logger.ErrorErr(err, "[%s] cannot parse desired state message", client.Domain())
			return
		}
		if err := client.handler.HandleDesiredState(envelope.ActivityID, envelope.Timestamp, desiredState); err != nil {
			logger.ErrorErr(err, "[%s] error processing desired state request", client.Domain())
		}
		return
	}
	if topic == client.topicDesiredStateCommand {
		logger.Trace("[%s] received desired state command request", client.Domain())
		desiredStateCommand := &types.DesiredStateCommand{}
		envelope, err := types.FromEnvelope(message.Payload(), desiredStateCommand)
		if err != nil {
			logger.ErrorErr(err, "[%s] cannot parse desired state command message", client.Domain())
			return
		}
		if err := client.handler.HandleDesiredStateCommand(envelope.ActivityID, envelope.Timestamp, desiredStateCommand); err != nil {
			logger.ErrorErr(err, "[%s] error processing desired state command request", client.Domain())
		}
		return
	}
	logger.Trace("[%s] received current state get request", client.Domain())
	envelope, err := types.FromEnvelope(message.Payload(), nil)
	if err != nil {
		logger.ErrorErr(err, "[%s] cannot parse current state get message", client.Domain())
		return
	}
	if err := client.handler.HandleCurrentStateGet(envelope.ActivityID, envelope.Timestamp); err != nil {
		logger.ErrorErr(err, "[%s] error processing current state get request", client.Domain())
	}
}

func (client *updateAgentClient) getAndPublishCurrentState() {
	currentTime := time.Now().UnixNano() / int64(time.Millisecond)
	activityID := prefixInitCurrentStateID + strconv.FormatInt(int64(currentTime), 10)
	if err := client.handler.HandleCurrentStateGet(activityID, currentTime); err != nil {
		logger.ErrorErr(err, "[%s] error processing initial current state get request", client.Domain())
	} else {
		logger.Debug("[%s] initial current state get request successfully processed", client.Domain())
	}
}

// SendCurrentState makes the client create envelope raw bytes with the given activityID and current state inventory and send the raw bytes as current state message.
func (client *updateAgentClient) SendCurrentState(activityID string, currentState *types.Inventory) error {
	currentStateBytes, err := types.ToEnvelope(activityID, currentState)
	if err != nil {
		return errors.Wrapf(err, "[%s] cannot marshal current state message", client.Domain())
	}
	if logger.IsTraceEnabled() {
		logger.Trace("[%s] publishing current state '%s'....", client.Domain(), currentState)
	} else {
		logger.Debug("[%s] publishing current state...", client.Domain())
	}
	return client.publish(client.topicCurrentState, true, currentStateBytes)
}

// SendDesiredStateFeedback makes the client send the given raw bytes as desired state feedback message.
func (client *updateAgentClient) SendDesiredStateFeedback(activityID string, desiredStateFeedback *types.DesiredStateFeedback) error {
	desiredStateFeedbackBytes, err := types.ToEnvelope(activityID, desiredStateFeedback)
	if err != nil {
		return errors.Wrapf(err, "[%s] cannot marshal desired state feedback message", client.Domain())
	}
	logger.Debug("[%s] publishing desired state feedback '%s'", client.Domain(), desiredStateFeedbackBytes)
	return client.publish(client.topicDesiredStateFeedback, false, desiredStateFeedbackBytes)
}

func (client *updateAgentClient) publish(topic string, retained bool, message []byte) error {
	logger.Debug("publishing to topic '%s'", topic)
	client.pahoClient.Publish(topic, 1, retained, message)
	logger.Debug("publishing to topic '%s' sent", topic)
	return nil
}

func domainAsTopic(domain string) string {
	domain = strings.ReplaceAll(domain, "-", "")
	if strings.HasSuffix(domain, "update") {
		return domain
	}
	return domain + "update"
}

func convertToMilliseconds(value int64) time.Duration {
	return time.Duration(value) * time.Millisecond
}

func newClient(config *ConnectionConfig, onConnect pahomqtt.OnConnectHandler) pahomqtt.Client {
	clientOptions := pahomqtt.NewClientOptions().
		SetClientID(uuid.New().String()). // TODO fixed ClientID ???
		AddBroker(config.BrokerURL).
		SetKeepAlive(convertToMilliseconds(config.KeepAlive)).
		SetCleanSession(true).
		SetAutoReconnect(true).
		SetProtocolVersion(4).
		SetConnectTimeout(convertToMilliseconds(config.ConnectTimeout)).
		SetOnConnectHandler(onConnect).
		SetUsername(config.ClientUsername).
		SetPassword(config.ClientPassword)

	return pahomqtt.NewClient(clientOptions)
}
