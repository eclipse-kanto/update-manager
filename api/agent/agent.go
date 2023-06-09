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

	"github.com/eclipse-kanto/update-manager/api"
	"github.com/eclipse-kanto/update-manager/api/types"
	"github.com/eclipse-kanto/update-manager/logger"

	"github.com/pkg/errors"
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
	if err := agent.client.Start(agent); err != nil {
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

	if err := agent.manager.Dispose(); err != nil {
		return err
	}
	if err := agent.client.Stop(); err != nil {
		return err
	}
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

func (agent *updateAgent) getCurrentState(ctx context.Context, activityID string) (*types.Inventory, error) {
	if strings.HasPrefix(activityID, prefixInitCurrentStateID) {
		agent.manager.WatchEvents(agent.ctx)
	}
	return agent.manager.Get(ctx, activityID)
}

func (agent *updateAgent) HandleDesiredState(activityID string, timestamp int64, desiredState *types.DesiredState) error {
	logger.Debug("Received desired state request: activity-id=%s, timestamp=%d", activityID, timestamp)
	go agent.applyDesiredState(activityID, desiredState)
	return nil
}

func (agent *updateAgent) HandleDesiredStateCommand(activityID string, timestamp int64, desiredStateCommand *types.DesiredStateCommand) error {
	logger.Debug("Received desired state command request: activity-id=%s, timestamp=%d, command=%s, baseline=%s", activityID, timestamp, desiredStateCommand.Command, desiredStateCommand.Baseline)
	go agent.commandDesiredState(activityID, desiredStateCommand)
	return nil
}

func (agent *updateAgent) HandleCurrentStateGet(activityID string, timestamp int64) error {
	logger.Debug("Received current state get request: activity-id=%s, timestamp=%s", activityID, timestamp)
	currentState, err := agent.getCurrentState(agent.ctx, activityID)
	if err != nil {
		return err
	}
	agent.stopCurrentStateStateNotifier()

	if err := agent.client.SendCurrentState(activityID, currentState); err != nil {
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
