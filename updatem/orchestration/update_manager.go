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
	"context"
	"sync"
	"time"

	"github.com/eclipse-kanto/update-manager/api"
	"github.com/eclipse-kanto/update-manager/api/types"
	"github.com/eclipse-kanto/update-manager/api/util"
	"github.com/eclipse-kanto/update-manager/config"
	"github.com/eclipse-kanto/update-manager/logger"
	"github.com/eclipse-kanto/update-manager/mqtt"
	"github.com/eclipse-kanto/update-manager/updatem/domain"
)

type aggregatedUpdateManager struct {
	applyLock sync.Mutex
	eventLock sync.Mutex

	name    string
	version string
	cfg     *config.Config

	updateOrchestrator api.UpdateOrchestrator
	domainsInventory   map[string]*types.Inventory

	rebootManager RebootManager
	domainAgents  map[string]api.UpdateManager
	eventCallback api.UpdateManagerCallback
}

// NewUpdateManager instantiates a new Kanto update manager
func NewUpdateManager(version string, cfg *config.Config, updateAgentClient api.UpdateAgentClient, updateOrchestrator api.UpdateOrchestrator) api.UpdateManager {
	domainAgents := make(map[string]api.UpdateManager)

	for domainName, agentConfig := range cfg.Agents {
		agentConfig.Name = domainName
		desiredStateClient := mqtt.NewDesiredStateClient(domainName, updateAgentClient)
		domainAgents[domainName] = domain.NewUpdateManager(desiredStateClient, agentConfig)
	}
	updateManager := &aggregatedUpdateManager{
		name:               cfg.Domain,
		version:            version,
		cfg:                cfg,
		domainsInventory:   map[string]*types.Inventory{},
		updateOrchestrator: updateOrchestrator,
		rebootManager:      &rebootManager{},
		domainAgents:       domainAgents,
	}
	for _, domainAgent := range domainAgents {
		domainAgent.SetCallback(updateManager)
	}

	return updateManager
}

func (updateManager *aggregatedUpdateManager) Name() string {
	return updateManager.name
}

func (updateManager *aggregatedUpdateManager) Apply(ctx context.Context, activityID string, desiredState *types.DesiredState) {
	updateManager.applyLock.Lock()
	defer updateManager.applyLock.Unlock()

	logger.Debug("processing desired state specification - start")
	rebootRequired := updateManager.updateOrchestrator.Apply(ctx, updateManager.domainAgents, activityID, desiredState, updateManager.eventCallback)
	logger.Debug("processing desired state specification - done")

	if inventory, err := updateManager.Get(ctx, activityID); err == nil {
		updateManager.eventCallback.HandleCurrentStateEvent(updateManager.Name(), activityID, inventory)
	} else {
		logger.Error(err.Error())
	}

	if rebootRequired {
		if updateManager.cfg.RebootEnabled {
			timeout := util.ParseDuration("reboot-after", updateManager.cfg.RebootAfter, 30*time.Second, 30*time.Second)
			if err := updateManager.rebootManager.Reboot(timeout); err != nil {
				logger.Error(err.Error())
			}
		} else {
			logger.Warn("reboot required but automatic rebooting is disabled")
		}
	}
}

func (updateManager *aggregatedUpdateManager) Command(ctx context.Context, activityID string, command *types.DesiredStateCommand) {
	// noop - agregated update manager does not handle commands
}

func (updateManager *aggregatedUpdateManager) Get(ctx context.Context, activityID string) (*types.Inventory, error) {
	logger.Trace("getting current state from update agents....")
	wg := &sync.WaitGroup{}
	domainsInventory := map[string]*types.Inventory{}
	for key, value := range updateManager.domainsInventory {
		domainsInventory[key] = value
	}
	for _, agent := range updateManager.domainAgents {
		wg.Add(1)
		go updateInventoryForDomain(ctx, wg, activityID, agent, domainsInventory)
	}
	wg.Wait()
	logger.Debug("got current state from update agents.")
	return toFullInventory(updateManager.asSoftwareNode(), domainsInventory), nil
}

func (updateManager *aggregatedUpdateManager) Dispose() error {
	logger.Debug("disposing update agents...")
	for _, agent := range updateManager.domainAgents {
		agent.Dispose()
	}
	logger.Debug("disposed update agents.")
	return nil
}

func (updateManager *aggregatedUpdateManager) WatchEvents(ctx context.Context) {
	logger.Debug("starting watching events from update agents...")
	for _, agent := range updateManager.domainAgents {
		agent.WatchEvents(ctx)
	}
	logger.Debug("started watching events from update agents.")
}

func (updateManager *aggregatedUpdateManager) SetCallback(callback api.UpdateManagerCallback) {
	updateManager.eventCallback = callback
}
