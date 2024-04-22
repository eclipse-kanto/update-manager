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

import "github.com/eclipse-kanto/update-manager/api/types"

// UpdateAgentHandler defines functions for handling the desired state / current state requests
type UpdateAgentHandler interface {
	HandleDesiredState(string, int64, *types.DesiredState) error
	HandleDesiredStateCommand(string, int64, *types.DesiredStateCommand) error
	HandleCurrentStateGet(string, int64) error
}

// BaseClient defines a common interface for both UpdateAgentClient and DesiredStateClient
type BaseClient interface {
	Domain() string

	Stop() error
}

// UpdateAgentClient defines an interface for interacting with the UpdateAgent API
type UpdateAgentClient interface {
	BaseClient

	Start(UpdateAgentHandler) error

	SendCurrentState(string, *types.Inventory) error
	SendDesiredStateFeedback(string, *types.DesiredStateFeedback) error
}

// StateHandler defines functions for handling the desired state feedback / current state responses
type StateHandler interface {
	HandleDesiredStateFeedback(string, int64, *types.DesiredStateFeedback) error
	HandleCurrentState(string, int64, *types.Inventory) error
}

// DesiredStateClient defines an interface for triggering requests towards an Update Agent implementation
type DesiredStateClient interface {
	BaseClient

	Start(StateHandler) error

	SendDesiredState(string, *types.DesiredState) error
	SendDesiredStateCommand(string, *types.DesiredStateCommand) error
	SendCurrentStateGet(string) error
}

// OwnerConsentAgentHandler defines functions for handling the owner consent requests
type OwnerConsentAgentHandler interface {
	HandleOwnerConsentGet(string, int64, *types.OwnerConsent) error
}

// OwnerConsentAgentClient defines an interface for handling for owner consent requests
type OwnerConsentAgentClient interface {
	BaseClient

	Start(OwnerConsentAgentHandler) error
	SendOwnerConsent(string, *types.OwnerConsent) error
}

// OwnerConsentHandler defines functions for handling the owner consent
type OwnerConsentHandler interface {
	HandleOwnerConsent(string, int64, *types.OwnerConsent) error
}

// OwnerConsentClient defines an interface for triggering requests for owner consent
type OwnerConsentClient interface {
	BaseClient

	Start(OwnerConsentHandler) error
	SendOwnerConsentGet(string) error
}
