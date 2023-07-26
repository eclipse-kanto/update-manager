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
	"testing"

	"github.com/eclipse-kanto/update-manager/api"
	"github.com/eclipse-kanto/update-manager/api/types"
	"github.com/eclipse-kanto/update-manager/test/mocks"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

type TestDesiredStateFeedbackHandler struct {
}

func (h *TestDesiredStateFeedbackHandler) HandleDesiredStateFeedbackEvent(domain string, activityID string, baseline string, status types.StatusType, message string, actions []*types.Action) {
}
func TestNewUpdateOperation(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	domainAgents := map[string]api.UpdateManager{
		"domain1": mocks.NewMockUpdateManager(mockCtrl),
		"domain2": mocks.NewMockUpdateManager(mockCtrl),
	}
	activityID := "activity-123"
	desiredState := &types.DesiredState{
		Domains: []*types.Domain{
			{
				ID: "domain1",
			},
			{
				ID: "domain2",
			},
		},
	}

	handler := &TestDesiredStateFeedbackHandler{}

	t.Run("test-valid-scenario", func(t *testing.T) {
		testOp, err := newUpdateOperation(domainAgents, activityID, desiredState, handler)
		assert.Nil(t, err)
		assert.NotNil(t, testOp)
		assert.Equal(t, activityID, testOp.activityID)
		assert.Equal(t, types.StatusIdentifying, testOp.status)
		assert.Empty(t, testOp.delayedStatus)
		assert.Len(t, testOp.domains, 2)
		assert.Equal(t, types.StatusIdentifying, testOp.domains["domain1"])
		assert.Equal(t, types.StatusIdentifying, testOp.domains["domain2"])
		assert.NotNil(t, testOp.actions)
		assert.NotNil(t, testOp.statesPerDomain)
		assert.NotNil(t, testOp.done)
		assert.NotNil(t, testOp.errChan)
		assert.NotNil(t, testOp.identDone)
		assert.NotNil(t, testOp.identErrChan)
		assert.False(t, testOp.rebootRequired)
		assert.Equal(t, handler, testOp.desiredStateCallback)
	})

	t.Run("test-missing-domain-agents", func(t *testing.T) {
		domainAgents := map[string]api.UpdateManager{}
		handler := &TestDesiredStateFeedbackHandler{}

		testOp, err := newUpdateOperation(domainAgents, activityID, desiredState, handler)
		assert.Error(t, err)
		assert.Nil(t, testOp)
		assert.Equal(t, "the desired state manifest does not contain any supported domain", err.Error())
	})

	t.Run("test-empty-desired-state", func(t *testing.T) {
		domainAgents := map[string]api.UpdateManager{
			"domain1": mocks.NewMockUpdateManager(mockCtrl),
			"domain2": mocks.NewMockUpdateManager(mockCtrl),
		}
		desiredState := &types.DesiredState{}
		handler := &TestDesiredStateFeedbackHandler{}

		testOp, err := newUpdateOperation(domainAgents, activityID, desiredState, handler)
		assert.Error(t, err)
		assert.Nil(t, testOp)
		assert.Equal(t, "the desired state manifest does not contain any supported domain", err.Error())
	})
	t.Run("test-one-missing-desired-state", func(t *testing.T) {
		domainAgents := map[string]api.UpdateManager{
			"domain1": mocks.NewMockUpdateManager(mockCtrl),
			"domain2": mocks.NewMockUpdateManager(mockCtrl),
		}
		desiredState := &types.DesiredState{
			Domains: []*types.Domain{
				{
					ID: "domain1",
				},
			},
		}
		handler := &TestDesiredStateFeedbackHandler{}

		testOp, err := newUpdateOperation(domainAgents, activityID, desiredState, handler)

		assert.Nil(t, err)
		assert.NotNil(t, testOp)
		assert.Equal(t, activityID, testOp.activityID)
		assert.Equal(t, types.StatusIdentifying, testOp.status)
		assert.Len(t, testOp.domains, 1)
		assert.Equal(t, types.StatusIdentifying, testOp.domains["domain1"])
		assert.Equal(t, types.StatusType(""), testOp.domains["domain2"])
		assert.Equal(t, handler, testOp.desiredStateCallback)

	})
}
