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
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/eclipse-kanto/update-manager/api"
	"github.com/eclipse-kanto/update-manager/api/types"
	"github.com/eclipse-kanto/update-manager/logger"
	"github.com/eclipse-kanto/update-manager/util/tls"

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
	suffixOwnerConsent         = "/ownerconsent"
	suffixOwnerConsentFeedback = "/ownerconsentfeedback"

	disconnectQuiesce uint = 10000
)

type internalConnectionConfig struct {
	Broker             string
	KeepAlive          time.Duration
	DisconnectTimeout  time.Duration
	Username           string
	Password           string
	ConnectTimeout     time.Duration
	AcknowledgeTimeout time.Duration
	SubscribeTimeout   time.Duration
	UnsubscribeTimeout time.Duration
	CACert             string
	Cert               string
	Key                string
}

func newInternalConnectionConfig(config *ConnectionConfig) *internalConnectionConfig {
	return &internalConnectionConfig{
		Broker:             config.Broker,
		KeepAlive:          parseDuration("mqtt-conn-keep-alive", config.KeepAlive, defaultKeepAlive),
		DisconnectTimeout:  parseDuration("mqtt-conn-disconnect-timeout", config.DisconnectTimeout, defaultDisconnectTimeout),
		Username:           config.Username,
		Password:           config.Password,
		ConnectTimeout:     parseDuration("mqtt-conn-connect-timeout", config.ConnectTimeout, defaultConnectTimeout),
		AcknowledgeTimeout: parseDuration("mqtt-conn-ack-timeout", config.AcknowledgeTimeout, defaultAcknowledgeTimeout),
		SubscribeTimeout:   parseDuration("mqtt-conn-sub-timeout", config.SubscribeTimeout, defaultSubscribeTimeout),
		UnsubscribeTimeout: parseDuration("mqtt-conn-unsub-timeout", config.UnsubscribeTimeout, defaultUnsubscribeTimeout),
		CACert:             config.CACert,
		Cert:               config.Cert,
		Key:                config.Key,
	}
}

type mqttClient struct {
	mqttConfig *internalConnectionConfig
	pahoClient pahomqtt.Client

	// UM incoming topics
	topicCurrentState         string
	topicDesiredStateFeedback string
	topicOwnerConsentFeedback string
	// UM outgoing topics
	topicDesiredState        string
	topicDesiredStateCommand string
	topicCurrentStateGet     string
	topicOwnerConsent        string
}

func newInternalClient(domain string, config *internalConnectionConfig, pahoClient pahomqtt.Client) *mqttClient {
	mqttPrefix := domainAsTopic(domain)
	return &mqttClient{
		mqttConfig: config,
		pahoClient: pahoClient,

		topicCurrentState:         mqttPrefix + suffixCurrentState,
		topicCurrentStateGet:      mqttPrefix + suffixCurrentStateGet,
		topicDesiredState:         mqttPrefix + suffixDesiredState,
		topicDesiredStateCommand:  mqttPrefix + suffixDesiredStateCommand,
		topicDesiredStateFeedback: mqttPrefix + suffixDesiredStateFeedback,
		topicOwnerConsent:         mqttPrefix + suffixOwnerConsent,
		topicOwnerConsentFeedback: mqttPrefix + suffixOwnerConsentFeedback,
	}
}

type updateAgentClient struct {
	*mqttClient
	domain  string
	handler api.UpdateAgentHandler
}

// NewUpdateAgentClient instantiates a new UpdateAgentClient instance using the provided configuration options.
func NewUpdateAgentClient(domain string, config *ConnectionConfig) (api.UpdateAgentClient, error) {
	client := &updateAgentClient{
		mqttClient: newInternalClient(domain, newInternalConnectionConfig(config), nil),
		domain:     domain,
	}
	pahoClient, err := newClient(client.mqttConfig, client.onConnect)
	if err == nil {
		client.pahoClient = pahoClient
	}
	return client, err
}

// Domain returns the name of the domain that is handled by this client.
func (client *updateAgentClient) Domain() string {
	return client.domain
}

// Start connects the client to the MQTT broker.
func (client *updateAgentClient) Start(handler api.UpdateAgentHandler) error {
	client.handler = handler
	token := client.pahoClient.Connect()
	if !token.WaitTimeout(client.mqttConfig.ConnectTimeout) {
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

func (client *updateAgentClient) onConnect(_ pahomqtt.Client) {
	go getAndPublishCurrentState(client.Domain(), client.handler.HandleCurrentStateGet)

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
	token := client.pahoClient.SubscribeMultiple(topicsMap, client.handleStateRequest)
	if !token.WaitTimeout(client.mqttConfig.SubscribeTimeout) {
		return fmt.Errorf("cannot subscribe for topics '%s,%s,%s' in '%v'", client.topicDesiredState, client.topicDesiredStateCommand, client.topicCurrentStateGet, client.mqttConfig.SubscribeTimeout)
	}
	return token.Error()
}

func (client *updateAgentClient) unsubscribeStateTopics() error {
	topics := []string{client.topicDesiredState, client.topicDesiredStateCommand, client.topicCurrentStateGet}
	logger.Debug("unsubscribing from '%s' topics", client.Domain(), topics)
	token := client.pahoClient.Unsubscribe(topics...)
	if !token.WaitTimeout(client.mqttConfig.UnsubscribeTimeout) {
		return fmt.Errorf("cannot unsubscribe from topics '%s,%s,%s' in '%v'", client.topicDesiredState, client.topicDesiredStateCommand, client.topicCurrentStateGet, client.mqttConfig.UnsubscribeTimeout)
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

// SendDesiredStateFeedback makes the client create envelope raw bytes with the given activityID and desired state feedback and send the raw bytes as desired state feedback message.
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

func newClient(config *internalConnectionConfig, onConnect pahomqtt.OnConnectHandler) (pahomqtt.Client, error) {
	clientOptions := pahomqtt.NewClientOptions().
		SetClientID(uuid.New().String()).
		AddBroker(config.Broker).
		SetKeepAlive(config.KeepAlive).
		SetCleanSession(true).
		SetAutoReconnect(true).
		SetProtocolVersion(4).
		SetConnectTimeout(config.ConnectTimeout).
		SetOnConnectHandler(onConnect).
		SetUsername(config.Username).
		SetPassword(config.Password)

	u, err := url.Parse(config.Broker)
	if err != nil {
		return nil, err
	}
	if isConnectionSecure(u.Scheme) {
		if len(config.CACert) == 0 {
			return nil, errors.New("connection is secure, but no TLS configuration is provided")
		}
		tlsConfig, err := tls.NewTLSConfig(config.CACert, config.Cert, config.Key)
		if err != nil {
			return nil, err
		}
		clientOptions.SetTLSConfig(tlsConfig)
	}

	return pahomqtt.NewClient(clientOptions), nil
}

func isConnectionSecure(schema string) bool {
	switch schema {
	case "wss", "ssl", "tls", "mqtts", "mqtt+ssl", "tcps":
		return true
	default:
	}
	return false
}

func getAndPublishCurrentState(domain string, currentStateGetHandler func(string, int64) error) {
	currentTime := time.Now().UnixNano() / int64(time.Millisecond)
	activityID := prefixInitCurrentStateID + strconv.FormatInt(int64(currentTime), 10)
	if err := currentStateGetHandler(activityID, currentTime); err != nil {
		logger.ErrorErr(err, "[%s] error processing initial current state get request", domain)
	} else {
		logger.Debug("[%s] initial current state get request successfully processed", domain)
	}
}
