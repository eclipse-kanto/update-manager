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
	"context"
	"fmt"
	"time"

	"sync"
	"testing"

	"github.com/eclipse-kanto/update-manager/api/types"
	"github.com/eclipse-kanto/update-manager/test"
	"github.com/eclipse-kanto/update-manager/test/mocks"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestAggregateDomainsInventory(t *testing.T) {
	testCases := map[string]struct {
		domainsInventory map[string]*types.Inventory
		expected         *types.Inventory
	}{
		"test_empty_domainsInventory": {
			expected: &types.Inventory{
				SoftwareNodes: []*types.SoftwareNode{
					test.MainInventoryNode,
				},
			},
		},
		"test_empty_SoftwareNodes_rootNode_nil": {
			domainsInventory: map[string]*types.Inventory{
				"safety-domain": {
					HardwareNodes: test.SampleTestHardwareNode,
				},
			},
			expected: &types.Inventory{
				SoftwareNodes: []*types.SoftwareNode{
					test.MainInventoryNode,
				},
				HardwareNodes: test.SampleTestHardwareNode,
			},
		},
		"test_software_type_not_application": {
			domainsInventory: map[string]*types.Inventory{
				"safety-domain": {
					HardwareNodes: test.SampleTestHardwareNode,
					SoftwareNodes: []*types.SoftwareNode{
						test.CreateSoftwareNode("safety-domain", 2, "domain", "safety-domain", types.SoftwareTypeContainer),
						test.CreateSoftwareNode("safety-domain", 3, "domain", "safety-domain", types.SoftwareTypeContainer),
					},
				},
			},
			expected: &types.Inventory{
				HardwareNodes: test.SampleTestHardwareNode,
				SoftwareNodes: []*types.SoftwareNode{
					test.MainInventoryNode,
					test.CreateSoftwareNode("safety-domain", 2, "domain", "safety-domain", types.SoftwareTypeContainer),
					test.CreateSoftwareNode("safety-domain", 3, "domain", "safety-domain", types.SoftwareTypeContainer),
				},
				Associations: []*types.Association{
					test.CreateAssociation("device-update-manager", "safety-domain-test:2"),
				},
			},
		},
		"test_multiple_softwaretype_first_not_application_second_application": {
			domainsInventory: map[string]*types.Inventory{
				"safety-domain": {
					HardwareNodes: test.SampleTestHardwareNode,
					SoftwareNodes: []*types.SoftwareNode{
						test.CreateSoftwareNode("safety-domain", 1, "domain", "safety-domain", types.SoftwareTypeContainer),
					},
				},
				"containers": {
					HardwareNodes: test.SampleTestHardwareNode,
					SoftwareNodes: []*types.SoftwareNode{
						test.CreateSoftwareNode("containers", 2, "domain", "containers", types.SoftwareTypeApplication),
					},
				},
			},
			expected: &types.Inventory{
				HardwareNodes: []*types.HardwareNode{
					test.SampleTestHardwareNode[0],
					test.SampleTestHardwareNode[0],
				},
				SoftwareNodes: []*types.SoftwareNode{
					test.MainInventoryNode,
					test.CreateSoftwareNode("safety-domain", 1, "domain", "safety-domain", types.SoftwareTypeContainer),
					test.CreateSoftwareNode("containers", 2, "domain", "containers", types.SoftwareTypeApplication),
				},
				Associations: []*types.Association{
					test.CreateAssociation("device-update-manager", "safety-domain-test:1"),
					test.CreateAssociation("device-update-manager", "containers-test:2"),
				},
			},
		},
		"test_parameter_key_not_domain": {
			domainsInventory: map[string]*types.Inventory{
				"safety-domain": {
					HardwareNodes: test.SampleTestHardwareNode,
					SoftwareNodes: []*types.SoftwareNode{
						test.CreateSoftwareNode("safety-domain", 2, "NOTdomain", "safety-domain", types.SoftwareTypeApplication),
						test.CreateSoftwareNode("safety-domain", 3, "NOTdomain", "safety-domain", types.SoftwareTypeApplication),
					},
				},
			},
			expected: &types.Inventory{
				HardwareNodes: test.SampleTestHardwareNode,
				SoftwareNodes: []*types.SoftwareNode{
					test.MainInventoryNode,
					test.CreateSoftwareNode("safety-domain", 2, "NOTdomain", "safety-domain", types.SoftwareTypeApplication),
					test.CreateSoftwareNode("safety-domain", 3, "NOTdomain", "safety-domain", types.SoftwareTypeApplication),
				},
				Associations: []*types.Association{
					test.CreateAssociation("device-update-manager", "safety-domain-test:2"),
				},
			},
		},
		"test_parameter_key_domain": {
			domainsInventory: map[string]*types.Inventory{
				"safety-domain": {
					HardwareNodes: test.SampleTestHardwareNode,
					SoftwareNodes: []*types.SoftwareNode{
						test.CreateSoftwareNode("safety-domain", 2, "domain", "safety-domain", types.SoftwareTypeApplication),
						test.CreateSoftwareNode("safety-domain", 3, "", "", types.SoftwareTypeApplication),
					},
					Associations: []*types.Association{
						test.CreateAssociation("safety-domain-test:2", "safety-domain-test:3"),
					},
				},
			},
			expected: &types.Inventory{
				HardwareNodes: test.SampleTestHardwareNode,
				SoftwareNodes: []*types.SoftwareNode{
					test.MainInventoryNode,
					test.CreateSoftwareNode("safety-domain", 2, "domain", "safety-domain", types.SoftwareTypeApplication),
					test.CreateSoftwareNode("safety-domain", 3, "", "", types.SoftwareTypeApplication),
				},
				Associations: []*types.Association{
					test.CreateAssociation("device-update-manager", "safety-domain-test:2"),
					test.CreateAssociation("safety-domain-test:2", "safety-domain-test:3"),
				},
			},
		},
		"test_parameter_key_domain_on_multiple_domainsInventory": {
			domainsInventory: map[string]*types.Inventory{
				"safety-domain": {
					HardwareNodes: test.SampleTestHardwareNode,
					SoftwareNodes: []*types.SoftwareNode{
						test.CreateSoftwareNode("safety-domain", 2, "domain", "safety-domain", types.SoftwareTypeApplication),
						test.CreateSoftwareNode("safety-domain", 3, "", "", types.SoftwareTypeApplication),
					},
					Associations: []*types.Association{
						{
							SourceID: "safety-domain-test:2",
							TargetID: "safety-domain-test:3",
						},
					},
				},
				"containers": {
					HardwareNodes: test.SampleTestHardwareNode,
					SoftwareNodes: []*types.SoftwareNode{
						test.CreateSoftwareNode("containers", 2, "domain", "containers", types.SoftwareTypeApplication),
						test.CreateSoftwareNode("containers", 3, "", "", types.SoftwareTypeApplication),
					},
					Associations: []*types.Association{
						{
							SourceID: "containers-test:2",
							TargetID: "containers-test:3",
						},
					},
				},
			},
			expected: &types.Inventory{
				HardwareNodes: []*types.HardwareNode{
					test.SampleTestHardwareNode[0],
					test.SampleTestHardwareNode[0],
				},
				SoftwareNodes: []*types.SoftwareNode{
					test.MainInventoryNode,
					test.CreateSoftwareNode("safety-domain", 2, "domain", "safety-domain", types.SoftwareTypeApplication),
					test.CreateSoftwareNode("safety-domain", 3, "", "", types.SoftwareTypeApplication),
					test.CreateSoftwareNode("containers", 2, "domain", "containers", types.SoftwareTypeApplication),
					test.CreateSoftwareNode("containers", 3, "", "", types.SoftwareTypeApplication),
				},
				Associations: []*types.Association{
					test.CreateAssociation("device-update-manager", "safety-domain-test:2"),
					test.CreateAssociation("safety-domain-test:2", "safety-domain-test:3"),
					test.CreateAssociation("device-update-manager", "safety-domain-test:2"),
					test.CreateAssociation("safety-domain-test:2", "safety-domain-test:3"),
				},
			},
		},
		"test_parameter_value_not_equal_domain": {
			domainsInventory: map[string]*types.Inventory{
				"safety-domain": {
					HardwareNodes: test.SampleTestHardwareNode,
					SoftwareNodes: []*types.SoftwareNode{
						test.CreateSoftwareNode("safety-domain", 2, "domain", "containers", types.SoftwareTypeApplication),
						test.CreateSoftwareNode("safety-domain", 3, "domain", "containers", types.SoftwareTypeApplication),
					},
				},
			},
			expected: &types.Inventory{
				HardwareNodes: test.SampleTestHardwareNode,
				SoftwareNodes: []*types.SoftwareNode{
					test.MainInventoryNode,
					test.CreateSoftwareNode("safety-domain", 2, "domain", "containers", types.SoftwareTypeApplication),
					test.CreateSoftwareNode("safety-domain", 3, "domain", "containers", types.SoftwareTypeApplication),
				},
				Associations: []*types.Association{
					{
						SourceID: "device-update-manager",
						TargetID: "safety-domain-test:2",
					},
				},
			},
		},
		"test_inventoryNode_multiple_layers": {
			domainsInventory: map[string]*types.Inventory{
				"safety-domain": {
					HardwareNodes: test.SampleTestHardwareNode,
					SoftwareNodes: []*types.SoftwareNode{
						test.CreateSoftwareNode("safety-domain", 2, "domain", "safety-domain", types.SoftwareTypeApplication),
						test.CreateSoftwareNode("safety-domain", 3, "", "", types.SoftwareTypeApplication),
						test.CreateSoftwareNode("safety-domain", 4, "", "", types.SoftwareTypeApplication),
					},
					Associations: []*types.Association{
						test.CreateAssociation("safety-domain-test:2", "safety-domain-test:3"),
						test.CreateAssociation("safety-domain-test:3", "safety-domain-test:4"),
					},
				},
			},
			expected: &types.Inventory{
				HardwareNodes: test.SampleTestHardwareNode,
				SoftwareNodes: []*types.SoftwareNode{
					test.MainInventoryNode,
					test.CreateSoftwareNode("safety-domain", 2, "domain", "safety-domain", types.SoftwareTypeApplication),
					test.CreateSoftwareNode("safety-domain", 3, "", "", types.SoftwareTypeApplication),
					test.CreateSoftwareNode("safety-domain", 4, "", "", types.SoftwareTypeApplication),
				},
				Associations: []*types.Association{
					test.CreateAssociation("device-update-manager", "safety-domain-test:2"),
					test.CreateAssociation("safety-domain-test:2", "safety-domain-test:3"),
					test.CreateAssociation("safety-domain-test:3", "safety-domain-test:4"),
				},
			},
		},
	}

	for testName, testCase := range testCases {
		t.Run(testName, func(t *testing.T) {
			agrUpdMgr := aggregatedUpdateManager{
				name:    "device",
				version: "development",
			}

			ret := toFullInventory(agrUpdMgr.asSoftwareNode(), testCase.domainsInventory)

			test.AssertInventoryWithoutElementsOrder(t, testCase.expected, ret)
		})
	}
}
func TestUpdateInventoryForDomain2(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	agent := mocks.NewMockUpdateManager(mockCtrl)

	testCases := map[string]struct {
		testInventory    *types.Inventory
		mockNameTimes    int
		mockGetReturnErr error
	}{
		"test_successful_Get_err_nil_inventory_not_nil": {
			testInventory: &types.Inventory{
				Associations: []*types.Association{
					test.CreateAssociation("safety-domain-test:1", "safety-domain-test:2"),
				},
			},
			mockNameTimes:    2,
			mockGetReturnErr: nil,
		},
		"test_Get_err_notNil_testInventory_nil": {
			testInventory:    nil,
			mockNameTimes:    1,
			mockGetReturnErr: fmt.Errorf("errNotNil"),
		},
		"test_Get_err_nil_inventory_nil": {
			testInventory:    nil,
			mockNameTimes:    1,
			mockGetReturnErr: nil,
		},
		"test_Get_err_notNil_intentory_notNil": {
			testInventory: &types.Inventory{
				Associations: []*types.Association{
					test.CreateAssociation("safety-domain-test:1", "safety-domain-test:2"),
				},
			},
			mockNameTimes:    2,
			mockGetReturnErr: fmt.Errorf("errNotNil"),
		},
	}
	for testName, testCase := range testCases {
		t.Log("testName:", testName)
		var wg sync.WaitGroup
		domainsInventory := make(map[string]*types.Inventory)

		agent.EXPECT().Get(context.Background(), test.ActivityID).Return(testCase.testInventory, testCase.mockGetReturnErr)
		agent.EXPECT().Name().Return("testName").Times(testCase.mockNameTimes)

		wg.Add(1)
		updateInventoryForDomain(context.Background(), &wg, test.ActivityID, agent, domainsInventory)
		test.AssertWithTimeout(t, &wg, 2*time.Second)
		assert.Equal(t, testCase.testInventory, domainsInventory["testName"])
	}
}
