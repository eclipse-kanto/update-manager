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
	"testing"
	"time"

	"github.com/eclipse-kanto/update-manager/test/mocks"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestWithCurrentStateReportDelay(t *testing.T) {
	mockCtr := gomock.NewController(t)
	defer mockCtr.Finish()

	mockClient := mocks.NewMockUpdateAgentClient(mockCtr)
	mockUpdateManager := mocks.NewMockUpdateManager(mockCtr)

	delay := 1 * time.Second
	actualAgent := NewUpdateAgent(mockClient, mockUpdateManager, WithCurrentStateReportDelay(delay))
	expAgent := &updateAgent{
		client:                  mockClient,
		manager:                 mockUpdateManager,
		currentStateReportDelay: delay,
	}
	assert.Equal(t, expAgent, actualAgent)
}

func TestWithDesiredStateFeedbackReportInterval(t *testing.T) {
	mockCtr := gomock.NewController(t)
	defer mockCtr.Finish()

	mockClient := mocks.NewMockUpdateAgentClient(mockCtr)
	mockUpdateManager := mocks.NewMockUpdateManager(mockCtr)

	interval := 1 * time.Second
	actualAgent := NewUpdateAgent(mockClient, mockUpdateManager, WithDesiredStateFeedbackReportInterval(interval))
	expAgent := &updateAgent{
		client:                             mockClient,
		manager:                            mockUpdateManager,
		desiredStateFeedbackReportInterval: interval,
	}
	assert.Equal(t, expAgent, actualAgent)
}
