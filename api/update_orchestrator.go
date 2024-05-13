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

// UpdateOrchestrator defines an interface for controlling the update process and applying of the desired state
type UpdateOrchestrator interface {
	Apply(context.Context, map[string]UpdateManager, string, *types.DesiredState, DesiredStateFeedbackHandler) bool

	DesiredStateFeedbackHandler
	OwnerConsentHandler
}
