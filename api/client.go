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

// UpdateAgentHandler defines functions for handling the desired state / current state requests
type UpdateAgentHandler interface {
	HandleDesiredState([]byte) error
	HandleDesiredStateCommand([]byte) error
	HandleCurrentStateGet([]byte) error
}

// UpdateAgentClient defines an interface for interacting with the UpdateAgent API
type UpdateAgentClient interface {
	Domain() string

	Connect(UpdateAgentHandler) error
	Disconnect()

	PublishCurrentState([]byte) error
	PublishDesiredStateFeedback([]byte) error
}

// StateHandler defines functions for handling the desired state feedback / current state responses
type StateHandler interface {
	HandleDesiredStateFeedback([]byte) error
	HandleCurrentState([]byte) error
}

// DesiredStateClient defines an interface for triggering requests towards an Update Agent implementation
type DesiredStateClient interface {
	Domain() string

	Subscribe(StateHandler) error
	Unsubscribe() error

	PublishDesiredState([]byte) error
	PublishDesiredStateCommand([]byte) error
	PublishGetCurrentState([]byte) error
}
