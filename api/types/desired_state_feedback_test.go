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

package types

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestToDesiredStateFeedbackBytes(t *testing.T) {
	feedback := &DesiredStateFeedback{
		Status:  StatusRunning,
		Message: "Applying desired state across 3 agents. Estimated time: about 5 minutes",
		Actions: []*Action{
			{
				Component: &Component{
					ID:      "containers:xyz",
					Version: "1",
				},
				Status:  ActionStatusUpdateSuccess,
				Message: "Container is healthy. Startup time: 7 seconds",
			},
			{
				Component: &Component{
					ID:      "safety-app:aap-app-1",
					Version: "4.3",
				},
				Status:   ActionStatusDownloading,
				Progress: 79,
				Message:  "Downloading 53.8 MiB",
			},
			{
				Component: &Component{
					ID:      "safety-ecu:powertrain",
					Version: "342.444.195",
				},
				Status:   ActionStatusUpdating,
				Progress: 35,
				Message:  "Writing firmware",
			},
		},
	}
	feedbackBytes, err := ToEnvelope("test-activity-id", feedback)
	assert.NoError(t, err)

	feedbackEnvelope := &Envelope{
		Payload: &DesiredStateFeedback{},
	}
	assert.NoError(t, json.Unmarshal(feedbackBytes, feedbackEnvelope))

	assert.Equal(t, "test-activity-id", feedbackEnvelope.ActivityID)
	assert.Equal(t, feedback, feedbackEnvelope.Payload)
}
