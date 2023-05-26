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

package agent

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"

	"github.com/eclipse-kanto/update-manager/api"
	"github.com/eclipse-kanto/update-manager/api/types"
	"github.com/eclipse-kanto/update-manager/logger"
)

const prefixInitCurrentStateID = "initial-current-state-"

type updateAgentOption = func(agent *updateAgent)

// UpdateAgent defines a struct for implementing an update agent.
type updateAgent struct {
	ctx     context.Context
	client  api.UpdateAgentClient
	manager api.UpdateManager

	desiredStateFeedbackReportInterval time.Duration
	currentStateReportDelay            time.Duration

	desiredStateFeedbackNotifier *desiredStateFeedbackNotifier
	currentStateNotifier         *currentStateNotifier

	clientLock               sync.Mutex
	desiredStateFeedbackLock sync.Mutex
	currentStateLock         sync.Mutex
}

// NewUpdateAgent instantiates an Update Agent instance.
func NewUpdateAgent(client api.UpdateAgentClient, manager api.UpdateManager, options ...updateAgentOption) api.UpdateAgent {
	updateAgent := &updateAgent{
		client:  client,
		manager: manager,
	}
	for _, option := range options {
		option(updateAgent)
	}
	return updateAgent
}

// Start method puts the Update Agent into operation.
// It will establish a connection to the MQTT broker and subscribe for incoming requests.
// It will also start monitoring the current state and report if upon changes.
func (agent *updateAgent) Start(ctx context.Context) error {
	agent.clientLock.Lock()
	defer agent.clientLock.Unlock()

	logger.Debug("starting update agent...")
	agent.manager.SetCallback(agent)

	agent.ctx = ctx
	err := agent.client.Connect(agent)
	if err != nil {
		return err
	}
	logger.Debug("started update agent.")
	return nil
}

// Stop method terminates the Update Agent operation.
func (agent *updateAgent) Stop() error {
	logger.Debug("stopping update agent...")
	agent.stopCurrentStateStateNotifier()
	agent.stopDesiredStateNotifier()

	agent.clientLock.Lock()
	defer agent.clientLock.Unlock()

	err := agent.manager.Dispose()
	if err != nil {
		return err
	}
	agent.client.Disconnect()
	logger.Debug("stopped update agent.")
	return nil
}

func (agent *updateAgent) stopDesiredStateNotifier() {
	agent.desiredStateFeedbackLock.Lock()
	defer agent.desiredStateFeedbackLock.Unlock()

	if agent.desiredStateFeedbackNotifier != nil {
		agent.desiredStateFeedbackNotifier.stop()
	}
}

func (agent *updateAgent) stopCurrentStateStateNotifier() {
	agent.currentStateLock.Lock()
	defer agent.currentStateLock.Unlock()

	if agent.currentStateNotifier != nil {
		agent.currentStateNotifier.stop()
	}
}

func (agent *updateAgent) GetCurrentState(ctx context.Context, activityID string) ([]byte, error) {
	if strings.HasPrefix(activityID, prefixInitCurrentStateID) {
		agent.manager.WatchEvents(agent.ctx)
	}
	var err error
	var inventory *types.Inventory
	if inventory, err = agent.manager.Get(ctx, activityID); err != nil {
		return nil, err
	}
	currentStateBytes, err := types.ToCurrentStateBytes(activityID, inventory)
	if err == nil {
		agent.stopCurrentStateStateNotifier()
	}
	return currentStateBytes, err
}

func (agent *updateAgent) HandleDesiredState(desiredStateBytes []byte) error {
	activityID, desiredState, err := types.FromDesiredStateBytes(desiredStateBytes)
	if err != nil {
		if activityID != "" {
			agent.HandleDesiredStateFeedbackEvent(agent.manager.Name(), activityID, "", types.StatusIdentificationFailed, err.Error(), []*types.Action{})
		}
		return err
	}
	logger.Debug("Received desired state request, activity-id=%s ", activityID)
	go agent.applyDesiredState(activityID, desiredState)
	return nil
}

func (agent *updateAgent) HandleDesiredStateCommand(desiredStateCommandBytes []byte) error {
	activityID, desiredStateCommand, err := types.FromDesiredStateCommandBytes(desiredStateCommandBytes)
	if err != nil {
		return err
	}
	logger.Debug("Received desired state command request, activity-id=%s, command=%s, baseline=%s", activityID, desiredStateCommand.Command, desiredStateCommand.Baseline)
	go agent.commandDesiredState(activityID, desiredStateCommand)
	return nil
}

func (agent *updateAgent) HandleCurrentStateGet(currentStateGetBytes []byte) error {
	activityID, err := types.FromCurrentStateGetBytes(currentStateGetBytes)
	if err != nil {
		return err
	}
	logger.Debug("Received current state get request, activity-id=%s, domain=%s", activityID, agent.manager.Name())
	currentStateBytes, err := agent.GetCurrentState(agent.ctx, activityID)
	if err != nil {
		return err
	}
	err = agent.client.PublishCurrentState(currentStateBytes)
	if err != nil {
		return errors.Wrap(err, "cannot publish current state.")
	}
	return nil
}

func (agent *updateAgent) applyDesiredState(activityID string, desiredState *types.DesiredState) {
	agent.clientLock.Lock()
	defer agent.clientLock.Unlock()

	logger.Trace("applying desired state...")
	agent.manager.Apply(agent.ctx, activityID, desiredState)
}

func (agent *updateAgent) commandDesiredState(activityID string, desiredStateCommand *types.DesiredStateCommand) {
	agent.clientLock.Lock()
	defer agent.clientLock.Unlock()

	logger.Trace("applying desired state command...")
	agent.manager.Command(agent.ctx, activityID, desiredStateCommand)
}
