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
	mqttPrefix string
	mqttConfig *ConnectionConfig
	pahoClient pahomqtt.Client
}

type updateAgentClient struct {
	*mqttClient
	domain  string
	handler api.UpdateAgentHandler
}

// NewUpdateAgentClient instantiates a new UpdateAgentClient instance using the provided configuration options.
func NewUpdateAgentClient(domain string, config *ConnectionConfig) api.UpdateAgentClient {
	client := &updateAgentClient{
		mqttClient: &mqttClient{
			mqttPrefix: domainAsTopic(domain),
			mqttConfig: config,
		},
		domain: domain,
	}
	client.pahoClient = newClient(config, client.onConnect)
	return client
}

func (client *updateAgentClient) topic(topicSuffix string) string {
	return client.mqttPrefix + topicSuffix
}

// Domain returns the name of the domain that is handled by this client.
func (client *updateAgentClient) Domain() string {
	return client.domain
}

// Connect connects the client to the MQTT broker.
func (client *updateAgentClient) Connect(handler api.UpdateAgentHandler) error {
	client.handler = handler
	connectTimeout := convertToMilliseconds(client.mqttConfig.ConnectTimeout)
	token := client.pahoClient.Connect()
	if !token.WaitTimeout(connectTimeout) {
		return fmt.Errorf("[%s] connect timed out", client.Domain())
	}
	return token.Error()
}

// Disconnect disconnects the client from the MQTT broker.
func (client *updateAgentClient) Disconnect() {
	if err := client.unsubscribeStateTopics(); err != nil {
		logger.WarnErr(err, "[%s] error unsubscribing for DesiredState/DesiredStateCommand/CurrentStateGet requests", client.Domain())
	} else {
		logger.Debug("[%s] unsubscribed for DesiredState/DesiredStateCommand/CurrentStateGet requests", client.Domain())
	}
	client.pahoClient.Disconnect(disconnectQuiesce)
	client.handler = nil
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
	topicDesiredState := client.topic(suffixDesiredState)
	topicDesiredStateCommand := client.topic(suffixDesiredStateCommand)
	topicCurrentStateGet := client.topic(suffixCurrentStateGet)
	topics := []string{topicDesiredState, topicDesiredStateCommand, topicCurrentStateGet}
	topicsMap := map[string]byte{
		topicDesiredState:        1,
		topicDesiredStateCommand: 1,
		topicCurrentStateGet:     1,
	}
	logger.Debug("subscribing for '%s' topics", topics)
	subscribeTimeout := convertToMilliseconds(client.mqttConfig.SubscribeTimeout)
	token := client.pahoClient.SubscribeMultiple(topicsMap, client.handleStateRequest)
	if !token.WaitTimeout(subscribeTimeout) {
		return fmt.Errorf("cannot subscribe for topics '%s,%s,%s' in '%v' seconds", topicDesiredState, topicDesiredStateCommand, topicCurrentStateGet, subscribeTimeout)
	}
	return token.Error()
}

func (client *updateAgentClient) unsubscribeStateTopics() error {
	topicDesiredState := client.topic(suffixDesiredState)
	topicDesiredStateCommand := client.topic(suffixDesiredStateCommand)
	topicCurrentStateGet := client.topic(suffixCurrentStateGet)
	topics := []string{topicDesiredState, topicDesiredStateCommand, topicCurrentStateGet}
	logger.Debug("unsubscribing from '%s' topics", client.Domain(), topics)
	token := client.pahoClient.Unsubscribe(topics...)
	unsubscribeTimeout := convertToMilliseconds(client.mqttConfig.UnsubscribeTimeout)
	if !token.WaitTimeout(unsubscribeTimeout) {
		return fmt.Errorf("cannot unsubscribe from topics '%s,%s,%s' in '%v' seconds", topicDesiredState, topicDesiredStateCommand, topicCurrentStateGet, unsubscribeTimeout)
	}
	return token.Error()
}

func (client *updateAgentClient) handleStateRequest(mqttClient pahomqtt.Client, message pahomqtt.Message) {
	topicDesiredState := client.topic(suffixDesiredState)
	topic := message.Topic()
	if topic == topicDesiredState {
		logger.Debug("[%s] received desired state request", client.Domain())
		desiredState := message.Payload()
		if err := client.handler.HandleDesiredState(desiredState); err != nil {
			logger.ErrorErr(err, "[%s] error processing desired state request", client.Domain())
		}
		return
	}
	topicDesiredStateCommand := client.topic(suffixDesiredStateCommand)
	if topic == topicDesiredStateCommand {
		logger.Trace("[%s] received desired state command request", client.Domain())
		if err := client.handler.HandleDesiredStateCommand(message.Payload()); err != nil {
			logger.ErrorErr(err, "[%s] error processing desired state command request", client.Domain())
		}
		return
	}
	logger.Trace("[%s] received current state get request", client.Domain())
	if err := client.handler.HandleCurrentStateGet(message.Payload()); err != nil {
		logger.ErrorErr(err, "[%s] error processing current state get request", client.Domain())
	}
}

func (client *updateAgentClient) getAndPublishCurrentState() {
	currentTime := time.Now().UnixNano() / int64(time.Millisecond)
	activityID := prefixInitCurrentStateID + strconv.FormatInt(int64(currentTime), 10)
	currentStateGet, err := types.ToCurrentStateGetBytes(activityID)
	if err != nil {
		logger.ErrorErr(err, "[%s] error getting initial current state", client.Domain())
		return
	}
	if err := client.handler.HandleCurrentStateGet(currentStateGet); err != nil {
		logger.ErrorErr(err, "[%s] error processing initial current state get request", client.Domain())
	} else {
		logger.Debug("[%s] initial current state get request successfully processed", client.Domain())
	}
}

// PublishCurrentState makes the client send the given raw bytes as current state message.
func (client *updateAgentClient) PublishCurrentState(currentState []byte) error {
	if logger.IsTraceEnabled() {
		logger.Trace("[%s] publishing current state '%s'....", client.Domain(), currentState)
	} else {
		logger.Debug("[%s] publishing current state...", client.Domain())
	}
	return client.publish(client.topic(suffixCurrentState), true, currentState)
}

// PublishCurrentState makes the client send the given raw bytes as desired state feedback message.
func (client *updateAgentClient) PublishDesiredStateFeedback(desiredStateFeedback []byte) error {
	logger.Debug("[%s] publishing desired state feedback '%s'", client.Domain(), desiredStateFeedback)
	return client.publish(client.topic(suffixDesiredStateFeedback), false, desiredStateFeedback)
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
