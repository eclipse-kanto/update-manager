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
	"github.com/eclipse-kanto/update-manager/api/types"
	"github.com/eclipse-kanto/update-manager/logger"
)

func (agent *updateAgent) HandleCurrentStateEvent(name string, activityID string, currentState *types.Inventory) {
	agent.currentStateLock.Lock()
	defer agent.currentStateLock.Unlock()

	logger.Debug("handle current state event for domain and activityId '%s' - '%s'", name, activityID)

	currentStateBytes, err := types.ToCurrentStateBytes(activityID, currentState)
	if err != nil {
		logger.ErrorErr(err, "cannot serialize current state.")
		return
	}

	if agent.currentStateReportDelay == 0 {
		agent.publishCurrentState(currentStateBytes)
		return
	}

	if activityID != "" {
		if agent.currentStateNotifier != nil {
			agent.currentStateNotifier.stop()
			agent.currentStateNotifier = nil
		}
		agent.publishCurrentState(currentStateBytes)
		return
	}
	if agent.currentStateNotifier == nil {
		agent.currentStateNotifier = newCurrentStateNotifier(agent.currentStateReportDelay, agent)
	}
	agent.currentStateNotifier.set(currentStateBytes)
}

func (agent *updateAgent) publishCurrentState(currentStateBytes []byte) {
	err := agent.client.PublishCurrentState(currentStateBytes)
	if err != nil {
		logger.ErrorErr(err, "cannot publish current state.")
	}
}
