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

package domain

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/pkg/errors"

	"github.com/eclipse-kanto/update-manager/api"
	"github.com/eclipse-kanto/update-manager/api/types"
	"github.com/eclipse-kanto/update-manager/api/util"
	"github.com/eclipse-kanto/update-manager/logger"
)

type domainUpdateManager struct {
	managerLock      sync.Mutex
	desiredStateLock sync.Mutex
	currentStateLock sync.Mutex

	desiredStateClient api.DesiredStateClient
	eventCallback      api.UpdateManagerCallback

	readTimeout time.Duration

	updateOperation *updateOperation
	currentState    *internalCurrentState
}

type internalCurrentState struct {
	expActivityID string
	receiveChan   chan bool
	inventory     *types.Inventory
	timestamp     int64
}

type updateOperation struct {
	activityID string
}

func newUpdateOperation(activityID string) *updateOperation {
	return &updateOperation{
		activityID: activityID,
	}
}

// NewUpdateManager instantiates a new instance of a domain specific update manager
func NewUpdateManager(desiredStateClient api.DesiredStateClient, updateConfig *api.UpdateManagerConfig) api.UpdateManager {
	return &domainUpdateManager{
		desiredStateClient: desiredStateClient,
		readTimeout:        util.ParseDuration(updateConfig.Name+"-read-timeout", updateConfig.ReadTimeout, time.Minute, time.Minute),
		currentState:       &internalCurrentState{},
	}
}

func (updateManager *domainUpdateManager) Apply(ctx context.Context, activityID string, desiredState *types.DesiredState) {
	updateManager.managerLock.Lock()
	defer updateManager.managerLock.Unlock()

	domainName := updateManager.Name()

	var errMessage string
	updateManager.updateOperation = newUpdateOperation(activityID)
	logger.Debug("[%s] processing desired state specification - start", domainName)

	desiredStateBytes, err := types.ToEnvelope(activityID, desiredState)
	if err != nil {
		errMessage := fmt.Sprintf("invalid desired state manifest for domain %s", domainName)
		updateManager.eventCallback.HandleDesiredStateFeedbackEvent(domainName, activityID, "", types.StatusIncomplete, errMessage, []*types.Action{})
		return
	}

	if err := updateManager.desiredStateClient.PublishDesiredState(desiredStateBytes); err != nil {
		errMessage = fmt.Sprintf("%s. cannot send desired state manifest to domain %s", err.Error(), domainName)
		updateManager.eventCallback.HandleDesiredStateFeedbackEvent(domainName, activityID, "", types.StatusIncomplete, errMessage, []*types.Action{})
		return
	}

	logger.Debug("[%s] processing desired state specification - done", domainName)
}

func (updateManager *domainUpdateManager) Command(ctx context.Context, activityID string, command *types.DesiredStateCommand) {
	updateManager.managerLock.Lock()
	defer updateManager.managerLock.Unlock()

	domainName := updateManager.Name()

	logger.Debug("[%s] processing desired state command request '%v' with activityId '%s'", domainName, command, activityID)

	if updateManager.updateOperation == nil {
		logger.Warn("[%s] the desired state command will be skipped, no active desired state apply operation.", domainName)
		return
	}
	if updateManager.updateOperation.activityID != activityID {
		logger.Warn("[%s] activity id mismatch for desired state command request - expecting %s, received %s", domainName, updateManager.updateOperation.activityID, activityID)
		return
	}
	commandBytes, err := types.ToDesiredStateCommandBytes(activityID, command)
	if err != nil {
		logger.Error(err.Error())
		return
	}
	if err := updateManager.desiredStateClient.PublishDesiredStateCommand(commandBytes); err != nil {
		errMessage := fmt.Sprintf("%s. cannot send desired state command to domain %s", err.Error(), domainName)
		updateManager.eventCallback.HandleDesiredStateFeedbackEvent(domainName, activityID, "", types.StatusIncomplete, errMessage, []*types.Action{})
		return
	}
	logger.Debug("[%s] processing desired state command '%v' with activityId '%s' - done", domainName, command, activityID)
}

func (updateManager *domainUpdateManager) Name() string {
	return updateManager.desiredStateClient.Domain()
}

func (updateManager *domainUpdateManager) Get(ctx context.Context, activityID string) (*types.Inventory, error) {
	updateManager.managerLock.Lock()
	defer updateManager.managerLock.Unlock()

	domainName := updateManager.Name()
	logger.Debug("[%s] processing get current state for activityID '%s'", domainName, activityID)

	updateManager.setCurrentStateActivityID(activityID)
	defer updateManager.setCurrentStateActivityID("")

	var bytes []byte
	var err error
	if bytes, err = types.ToEnvelope(activityID, nil); err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("cannot serialize envelope with activityID '%s' for domain %s", activityID, domainName))
	}
	if err := updateManager.desiredStateClient.PublishGetCurrentState(bytes); err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("cannot send get current state request for domain %s", domainName))
	}
	select {
	case <-updateManager.currentState.receiveChan:
		return updateManager.currentState.inventory, nil
	case <-time.After(updateManager.readTimeout):
		return updateManager.currentState.inventory, fmt.Errorf("cannot get current state by activityID '%s' in '%s' for domain %s", activityID, updateManager.readTimeout, domainName)
	case <-ctx.Done():
		return nil, fmt.Errorf("the Update Manager instance is already terminated")
	}
}

func (updateManager *domainUpdateManager) setCurrentStateActivityID(activityID string) {
	updateManager.currentStateLock.Lock()
	defer updateManager.currentStateLock.Unlock()

	updateManager.currentState.expActivityID = activityID
	if activityID == "" {
		updateManager.currentState.receiveChan = nil
	} else {
		updateManager.currentState.receiveChan = make(chan bool, 1)
	}
}

func (updateManager *domainUpdateManager) Dispose() error {
	updateManager.desiredStateClient.Unsubscribe()
	return nil
}

func (updateManager *domainUpdateManager) WatchEvents(ctx context.Context) {
	err := updateManager.desiredStateClient.Subscribe(updateManager)
	if err != nil {
		logger.ErrorErr(err, "[%s] cannot subscribe for events", updateManager.Name())
	}
}

func (updateManager *domainUpdateManager) SetCallback(callback api.UpdateManagerCallback) {
	updateManager.eventCallback = callback
}
