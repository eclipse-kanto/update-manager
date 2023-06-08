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
	"time"
)

// WithCurrentStateReportDelay defines option for update agent to delay the current state reporting for the given duration, e.g. if current state changes meanwhile, only the latest current state will be sent
func WithCurrentStateReportDelay(delay time.Duration) updateAgentOption {
	return func(agent *updateAgent) {
		agent.currentStateReportDelay = delay
	}
}

// WithDesiredStateFeedbackReportInterval defines option for update agent to delay the desired state feedback for the given duration, e.g. if there are newer desired state feedback updates, only the latest feedback will be sent
func WithDesiredStateFeedbackReportInterval(interval time.Duration) updateAgentOption {
	return func(agent *updateAgent) {
		agent.desiredStateFeedbackReportInterval = interval
	}
}
