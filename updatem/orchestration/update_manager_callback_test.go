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
	"github.com/eclipse-kanto/update-manager/test"
	"github.com/eclipse-kanto/update-manager/test/mocks"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestHandleDesiredStateFeedbackEvent(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	testActions := []*types.Action{
		{
			Status:  types.ActionStatusActivating,
			Message: "testMsg1",
			Component: &types.Component{
				ID: "testId",
			},
		},
	}

	mockUpdateOrchestrator := mocks.NewMockUpdateOrchestrator(mockCtrl)
	updateManager := createTestUpdateManager(nil, nil, nil, 0, nil, mockUpdateOrchestrator, nil, "development")

	mockUpdateOrchestrator.EXPECT().HandleDesiredStateFeedbackEvent("testDomain", "testActivityId", "testBaseline", types.BaselineStatusActivating, "testMsg", testActions)
	updateManager.HandleDesiredStateFeedbackEvent("testDomain", "testActivityId", "testBaseline", types.BaselineStatusActivating, "testMsg", testActions)
}

var expectedInventory = &types.Inventory{
	SoftwareNodes: []*types.SoftwareNode{
		test.MainInventoryNode,
		test.CreateSoftwareNode("testDomain", 1, "", "", types.SoftwareTypeApplication),
		test.CreateSoftwareNode("testDomain", 2, "", "", types.SoftwareTypeApplication),
	},
	Associations: []*types.Association{
		test.CreateAssociation("device-update-manager", "testDomain-test-1"),
		test.CreateAssociation("device-update-manager", "testDomain-test-1"),
		test.CreateAssociation("device-update-manager", "testDomain-test-1"),
		test.CreateAssociation("device-update-manager", "testDomain-test-2"),
	},
	HardwareNodes: test.SampleTestHardwareNode,
}

var givenInventory = &types.Inventory{
	SoftwareNodes: []*types.SoftwareNode{
		test.CreateSoftwareNode("testDomain", 1, "", "", types.SoftwareTypeApplication),
	},
	HardwareNodes: test.SampleTestHardwareNode,
	Associations: []*types.Association{
		test.CreateAssociation("device-update-manager", "testDomain-test-1"),
		test.CreateAssociation("device-update-manager", "testDomain-test-2"),
	},
}

func TestHandleCurrentStateEvent(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	defaultlyAddedInventory := map[string]*types.Inventory{
		"testDomainInventory1": {
			SoftwareNodes: []*types.SoftwareNode{
				test.CreateSoftwareNode("testDomain", 2, "", "", types.SoftwareTypeApplication),
			},
		},
	}
	t.Run("test_HandleCurrentStateEvent_activityId_nil", func(t *testing.T) {
		eventCallback := mocks.NewMockUpdateManagerCallback(mockCtrl)
		domainUpdateManager := mocks.NewMockUpdateManager(mockCtrl)
		domainUpdateManagers := map[string]api.UpdateManager{"testDomain1": domainUpdateManager}
		updateManager := createTestUpdateManager(eventCallback, domainUpdateManagers, nil, 0, nil, nil, defaultlyAddedInventory, "development")

		eventCallback.EXPECT().HandleCurrentStateEvent("device", "", gomock.Any()).DoAndReturn(
			func(name string, activityId string, invent *types.Inventory) {
				test.AssertInventoryWithoutElementsOrder(t, expectedInventory, invent)
			})

		updateManager.HandleCurrentStateEvent("testName", "", givenInventory)
		assert.Equal(t, givenInventory, updateManager.domainsInventory["testName"])
	})
	t.Run("test_HandleCurrentStateEvent_activityId_notNil", func(t *testing.T) {
		updateManager := createTestUpdateManager(nil, nil, nil, 0, nil, nil, defaultlyAddedInventory, "development")
		updateManager.HandleCurrentStateEvent("testName", "testActivityId", givenInventory)
		assert.Equal(t, givenInventory, updateManager.domainsInventory["testName"])
	})
}
