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
	"encoding/json"
	"fmt"
	"sync"

	"github.com/eclipse-kanto/update-manager/api"
	"github.com/eclipse-kanto/update-manager/api/types"
	"github.com/eclipse-kanto/update-manager/logger"
	"github.com/eclipse/ditto-clients-golang"
	"github.com/eclipse/ditto-clients-golang/model"
	"github.com/eclipse/ditto-clients-golang/protocol"
	"github.com/eclipse/ditto-clients-golang/protocol/things"
)

const (
	// UpdateManagerFeatureID is the feature ID of the update manager
	updateManagerFeatureID = "UpdateManager"
	// updateManagerFeatureDefinition is the feature definition of the update manager
	updateManagerFeatureDefinition = "com.bosch.iot.suite.edge.update:UpdateManager:1.0.0"
	// incoming operations
	updateManagerFeatureOperationApply   = "apply"
	updateManagerFeatureOperationReport  = "report"
	updateManagerFeatureOperationCommand = "command"
	// outgoing messages
	updateManagerFeatureMessageFeedback = "feedback"
	// properties
	updateManagerFeaturePropertyState  = "state"
	updateManagerFeaturePropertyDomain = "domain"

	jsonContent = "application/json"
)

// UpdateManagerFeature describes the update manager feature representation.
type UpdateManagerFeature interface {
	Activate() error
	Deactivate()
	SetCurrentState(envelope *types.Envelope) error
	SendDesiredStateFeedback(envelope *types.Envelope) error
}

type updateManagerFeature struct {
	sync.Mutex
	active      bool
	thingID     *model.NamespacedID
	dittoClient ditto.Client
	domain      string
	handler     api.UpdateAgentHandler
}

// NewUpdateManagerFeature creates a new update manager feature representation.
func NewUpdateManagerFeature(domain string, deviceID string, dittoClient ditto.Client, handler api.UpdateAgentHandler) UpdateManagerFeature {
	return &updateManagerFeature{
		domain:      domain,
		thingID:     model.NewNamespacedIDFrom(deviceID),
		dittoClient: dittoClient,
		handler:     handler,
	}
}

// Activate subscribes for incoming Ditto messages and registers the UpdateManager feature.
func (um *updateManagerFeature) Activate() error {
	um.Lock()
	defer um.Unlock()

	if um.active {
		return nil
	}

	feature := (&model.Feature{}).
		WithDefinition(model.NewDefinitionIDFrom(updateManagerFeatureDefinition)).
		WithProperty(updateManagerFeaturePropertyDomain, um.domain)
	event := things.NewCommand(um.thingID).Feature(updateManagerFeatureID).Modify(feature).Twin()

	// Add the UpdateManager feature.
	um.dittoClient.Subscribe(um.messagesHandler)
	if err := um.dittoClient.Send(event.Envelope(protocol.WithResponseRequired(false))); err != nil {
		um.dittoClient.Unsubscribe()
		return err
	}
	um.active = true
	return nil
}

// Deactivate unsubscribes from incoming Ditto messages.
func (um *updateManagerFeature) Deactivate() {
	um.Lock()
	defer um.Unlock()
	if !um.active {
		return
	}

	um.dittoClient.Unsubscribe()
	um.active = false
}

// SetCurrentState modifies the current state property of th e feature.
func (um *updateManagerFeature) SetCurrentState(envelope *types.Envelope) error {
	um.Lock()
	defer um.Unlock()

	if !um.active {
		return nil
	}
	cmd := things.NewCommand(um.thingID).FeatureProperty(updateManagerFeatureID, updateManagerFeaturePropertyState).Modify(envelope).Twin()
	return um.dittoClient.Send(cmd.Envelope(protocol.WithResponseRequired(false), protocol.WithContentType(jsonContent)))
}

// SendDesiredStateFeedback issues a desired feedback message to the cloud.
func (um *updateManagerFeature) SendDesiredStateFeedback(envelope *types.Envelope) error {
	um.Lock()
	defer um.Unlock()

	if !um.active {
		return nil
	}
	message := things.NewMessage(um.thingID).Feature(updateManagerFeatureID).Outbox(updateManagerFeatureMessageFeedback).WithPayload(envelope)
	return um.dittoClient.Send(message.Envelope(protocol.WithResponseRequired(false), protocol.WithContentType(jsonContent)))
}

func (um *updateManagerFeature) messagesHandler(requestID string, msg *protocol.Envelope) {
	um.Lock()
	defer um.Unlock()

	if !um.active {
		return
	}
	logger.Trace("[%s][%s] received message with request id '%s': %v", updateManagerFeatureID, um.domain, requestID, msg)
	if msg.Topic.Namespace == um.thingID.Namespace && msg.Topic.EntityName == um.thingID.Name {
		if msg.Path == fmt.Sprintf("/features/%s/inbox/messages/%s", updateManagerFeatureID, updateManagerFeatureOperationApply) {
			um.processApply(requestID, msg)
		} else if msg.Path == fmt.Sprintf("/features/%s/inbox/messages/%s", updateManagerFeatureID, updateManagerFeatureOperationReport) {
			um.processReport(requestID, msg)
		} else if msg.Path == fmt.Sprintf("/features/%s/inbox/messages/%s", updateManagerFeatureID, updateManagerFeatureOperationCommand) {
			um.processCommand(requestID, msg)
		} else {
			logger.Debug("There is no handler for a message - skipping processing")
		}
	} else {
		logger.Debug("[%s][%s] skipping processing of unexpected message with request id '%s': %v", updateManagerFeatureID, um.domain, requestID, msg)
	}
}

func (um *updateManagerFeature) processApply(requestID string, msg *protocol.Envelope) {
	ds := &types.DesiredState{}
	envelope, prepared := um.prepare(requestID, msg, updateManagerFeatureOperationApply, ds)
	if !prepared {
		return
	}
	go func(handler api.UpdateAgentHandler) {
		logger.Trace("[%s][%s] processing apply operation", updateManagerFeatureID, um.domain)
		if err := um.handler.HandleDesiredState(envelope.ActivityID, envelope.Timestamp, ds); err != nil {
			logger.ErrorErr(err, "[%s][%s] error processing apply operation", updateManagerFeatureID, um.domain)
		}
	}(um.handler)
}

func (um *updateManagerFeature) processReport(requestID string, msg *protocol.Envelope) {
	envelope, prepared := um.prepare(requestID, msg, updateManagerFeatureOperationReport, nil)
	if !prepared {
		return
	}
	go func(handler api.UpdateAgentHandler) {
		logger.Trace("[%s][%s] processing report operation", updateManagerFeatureID, um.domain)
		if err := um.handler.HandleCurrentStateGet(envelope.ActivityID, envelope.Timestamp); err != nil {
			logger.ErrorErr(err, "[%s][%s] error processing report operation", updateManagerFeatureID, um.domain)
		}
	}(um.handler)
}

func (um *updateManagerFeature) processCommand(requestID string, msg *protocol.Envelope) {
	command := &types.DesiredStateCommand{}
	envelope, prepared := um.prepare(requestID, msg, updateManagerFeatureOperationCommand, command)
	if !prepared {
		return
	}
	go func(handler api.UpdateAgentHandler) {
		logger.Trace("[%s][%s] processing command '%s'", updateManagerFeatureID, um.domain, command.Command)
		if err := handler.HandleDesiredStateCommand(envelope.ActivityID, envelope.Timestamp, command); err != nil {
			logger.ErrorErr(err, "[%s][%s] error processing command '%s'", updateManagerFeatureID, um.domain, command.Command)
		}
	}(um.handler)

}

func (um *updateManagerFeature) prepare(requestID string, msg *protocol.Envelope, operation string, to interface{}) (*types.Envelope, bool) {
	logger.Trace("[%s][%s] parse message value: %v", updateManagerFeatureID, um.domain, msg.Value)

	var (
		envelope *types.Envelope
		respReq  = msg.Headers.IsResponseRequired()
	)
	bytes, err := json.Marshal(msg.Value)
	if err == nil {
		envelope, err = types.FromEnvelope(bytes, to)
		if err == nil {
			if respReq {
				um.reply(requestID, msg.Headers.CorrelationID(), operation, 204, nil)
			}
			logger.Debug("[%s][%s] execute '%s' operation with correlation id '%s'", updateManagerFeatureID, um.domain, operation, msg.Headers.CorrelationID())
			return envelope, true
		}
	}
	thingErr := newMessagesParameterInvalidError(err.Error())
	logger.ErrorErr(thingErr, "[%s][%s] failed to parse message value", updateManagerFeatureID, um.domain)
	if respReq {
		um.reply(requestID, msg.Headers.CorrelationID(), operation, thingErr.Status, thingErr)
	}
	return nil, false
}

func (um *updateManagerFeature) reply(requestID string, cid string, cmd string, status int, payload interface{}) {
	bHeadersOpts := [3]protocol.HeaderOpt{protocol.WithCorrelationID(cid), protocol.WithResponseRequired(false)}
	headerOpts := bHeadersOpts[:2]
	response := things.NewMessage(um.thingID).Feature(updateManagerFeatureID).Outbox(cmd)
	if payload != nil {
		response.WithPayload(payload)
		headerOpts = append(headerOpts, protocol.WithContentType(jsonContent))
	}
	responseMsg := response.Envelope(headerOpts...)
	responseMsg.Status = status

	if err := um.dittoClient.Reply(requestID, responseMsg); err != nil {
		logger.ErrorErr(err, "[%s][%s] failed to send error response for request id '%s'", updateManagerFeatureID, um.domain, requestID)
	} else {
		logger.Debug("[%s][%s] sent reply for request id '%s': %v", updateManagerFeatureID, um.domain, requestID, responseMsg)
	}
}
