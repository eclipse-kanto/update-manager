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

package orchestration

import (
	"fmt"
	"sync"

	"github.com/eclipse-kanto/update-manager/api"
	"github.com/eclipse-kanto/update-manager/api/types"
	"github.com/eclipse-kanto/update-manager/logger"
)

type updateOperation struct {
	activityID string

	statusLock    sync.Mutex
	status        types.StatusType
	delayedStatus types.StatusType
	delayedErrMsg string

	domains map[string]types.StatusType
	actions map[string]map[string]*types.Action

	desiredState    *types.DesiredState
	statesPerDomain map[api.UpdateManager]*types.DesiredState

	commandChannels map[types.CommandType]chan bool
	done            chan bool

	errChan chan bool
	errMsg  string

	ownerConsented chan bool
	rollbackChan   chan bool

	rebootRequired bool

	desiredStateCallback api.DesiredStateFeedbackHandler
}

func newUpdateOperation(domainAgents map[string]api.UpdateManager, activityID string,
	desiredState *types.DesiredState, desiredStateCallback api.DesiredStateFeedbackHandler) (*updateOperation, error) {

	statesPerDomain := map[api.UpdateManager]*types.DesiredState{}
	domainStatuses := map[string]types.StatusType{}
	for domain, statePerDomain := range desiredState.SplitPerDomains() {
		updateManagerForDomain, ok := domainAgents[domain]
		if !ok {
			logger.Warn("Cannot find Update Agent for domain %s", domain)
			continue
			// TODO update agent for domain is missing, what to do
		}
		statesPerDomain[updateManagerForDomain] = statePerDomain
		domainStatuses[domain] = types.StatusIdentifying
	}
	if len(statesPerDomain) == 0 {
		return nil, fmt.Errorf("the desired state manifest does not contain any supported domain")
	}
	return &updateOperation{
		activityID: activityID,

		status: types.StatusIdentifying,

		domains: domainStatuses,
		actions: map[string]map[string]*types.Action{},

		statesPerDomain: statesPerDomain,
		desiredState:    desiredState,
		commandChannels: generateCommandChannels(),

		done: make(chan bool, 1),

		errChan:        make(chan bool, 1),
		ownerConsented: make(chan bool, 1),
		rollbackChan:   make(chan bool, 1),

		desiredStateCallback: desiredStateCallback,
	}, nil
}

func (operation *updateOperation) updateStatus(status types.StatusType) {
	operation.statusLock.Lock()
	defer operation.statusLock.Unlock()

	operation.status = status
}

func generateCommandChannels() map[types.CommandType]chan bool {
	commandChannels := make(map[types.CommandType]chan bool, len(orderedCommands))
	for _, command := range orderedCommands {
		commandChannels[command] = make(chan bool, 1)
	}
	return commandChannels
}
