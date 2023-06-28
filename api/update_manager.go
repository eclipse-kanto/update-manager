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

package api

import (
	"context"

	"github.com/eclipse-kanto/update-manager/api/types"
)

// UpdateAgent defines the interface for starting/stopping an update agent.
type UpdateAgent interface {
	Start(context.Context) error
	Stop() error
}

// UpdateManagerConfig holds configuration properties for an update manager.
type UpdateManagerConfig struct {
	Name           string
	RebootRequired bool   `json:"rebootRequired"`
	ReadTimeout    string `json:"readTimeout"`
}

// UpdateManager provides the orchestration management abstraction
type UpdateManager interface {
	Name() string

	SetCallback(callback UpdateManagerCallback)
	WatchEvents(ctx context.Context)

	Apply(ctx context.Context, activityID string, desiredState *types.DesiredState)
	Command(ctx context.Context, activityID string, command *types.DesiredStateCommand)
	Get(ctx context.Context, activityID string) (*types.Inventory, error)

	Dispose() error
}

// DesiredStateFeedbackHandler defines a callback for handling desired state feedback events
type DesiredStateFeedbackHandler interface {
	HandleDesiredStateFeedbackEvent(domain string, activityID string, baseline string, status types.StatusType, message string, actions []*types.Action)
}

// CurrentStateHandler defines a callback for handling current state events
type CurrentStateHandler interface {
	HandleCurrentStateEvent(domain string, activityID string, currentState *types.Inventory)
}

// UpdateManagerCallback defines a callback for event handling
type UpdateManagerCallback interface {
	DesiredStateFeedbackHandler
	CurrentStateHandler
}
