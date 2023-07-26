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
	"reflect"
	"strconv"
	"sync"
	"testing"

	"github.com/eclipse-kanto/update-manager/api/types"
	"github.com/eclipse-kanto/update-manager/test/mocks"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestAggregateDomainsInventory(t *testing.T) {
	var sampleTestHardwareNode = []*types.HardwareNode{
		{
			InventoryNode: types.InventoryNode{
				ID:      "testId",
				Version: "3.0.0",
				Name:    "testName",
				Parameters: []*types.KeyValuePair{
					{
						Key:   "k1",
						Value: "v1",
					},
				},
			},
		},
	}
	var mainInventoryNode = &types.SoftwareNode{
		InventoryNode: types.InventoryNode{
			ID:      "device-update-manager",
			Version: "2.0.0",
			Name:    "Update Manager",
		},
		Type: types.SoftwareTypeApplication,
	}

	testCases := map[string]struct {
		agrVersion       string
		domainsInventory map[string]*types.Inventory
		expected         *types.Inventory
	}{
		"test_empty_domainsInventory": {
			agrVersion: "2.0.0",
			expected: &types.Inventory{
				SoftwareNodes: []*types.SoftwareNode{
					mainInventoryNode,
				},
			},
		},
		"test_empty_SoftwareNodes_rootNode_nil": {
			agrVersion: "2.0.0",
			domainsInventory: map[string]*types.Inventory{
				"safety-domain": {
					HardwareNodes: sampleTestHardwareNode,
				},
			},
			expected: &types.Inventory{
				SoftwareNodes: []*types.SoftwareNode{
					mainInventoryNode,
				},
				HardwareNodes: sampleTestHardwareNode,
			},
		},
		"test_software_type_not_application": {
			agrVersion: "2.0.0",
			domainsInventory: map[string]*types.Inventory{
				"safety-domain": {
					HardwareNodes: sampleTestHardwareNode,
					SoftwareNodes: []*types.SoftwareNode{
						{
							InventoryNode: createTestInventoryNode("safety-domain", 2, "domain", "safety-domain"),
							Type:          types.SoftwareTypeContainer,
						},
						{
							InventoryNode: createTestInventoryNode("safety-domain", 3, "domain", "safety-domain"),
							Type:          types.SoftwareTypeContainer,
						},
					},
				},
			},
			expected: &types.Inventory{
				HardwareNodes: sampleTestHardwareNode,
				SoftwareNodes: []*types.SoftwareNode{
					mainInventoryNode,
					{
						InventoryNode: createTestInventoryNode("safety-domain", 2, "domain", "safety-domain"),
						Type:          types.SoftwareTypeContainer,
					},
					{
						InventoryNode: createTestInventoryNode("safety-domain", 3, "domain", "safety-domain"),
						Type:          types.SoftwareTypeContainer,
					},
				},
				Associations: []*types.Association{
					{
						SourceID: "device-update-manager",
						TargetID: "safety-domain-test:2",
					},
				},
			},
		},
		"test_multiple_softwaretype_first_not_application_second_application": {
			agrVersion: "2.0.0",
			domainsInventory: map[string]*types.Inventory{
				"safety-domain": {
					HardwareNodes: sampleTestHardwareNode,
					SoftwareNodes: []*types.SoftwareNode{
						{
							InventoryNode: createTestInventoryNode("safety-domain", 1, "domain", "safety-domain"),
							Type:          types.SoftwareTypeContainer,
						},
					},
				},
				"containers": {
					HardwareNodes: sampleTestHardwareNode,
					SoftwareNodes: []*types.SoftwareNode{
						{
							InventoryNode: createTestInventoryNode("containers", 2, "domain", "containers"),
							Type:          types.SoftwareTypeApplication,
						},
					},
				},
			},
			expected: &types.Inventory{
				HardwareNodes: []*types.HardwareNode{
					sampleTestHardwareNode[0],
					sampleTestHardwareNode[0],
				},
				SoftwareNodes: []*types.SoftwareNode{
					mainInventoryNode,
					{
						InventoryNode: createTestInventoryNode("safety-domain", 1, "domain", "safety-domain"),
						Type:          types.SoftwareTypeContainer,
					},
					{
						InventoryNode: createTestInventoryNode("containers", 2, "domain", "containers"),
						Type:          types.SoftwareTypeApplication,
					},
				},
				Associations: []*types.Association{
					{
						SourceID: "device-update-manager",
						TargetID: "safety-domain-test:1",
					},
					{
						SourceID: "device-update-manager",
						TargetID: "containers-test:2",
					},
				},
			},
		},
		"test_parameter_key_not_domain": {
			agrVersion: "2.0.0",
			domainsInventory: map[string]*types.Inventory{
				"safety-domain": {
					HardwareNodes: sampleTestHardwareNode,
					SoftwareNodes: []*types.SoftwareNode{
						{
							InventoryNode: createTestInventoryNode("safety-domain", 2, "NOTdomain", "safety-domain"),
							Type:          types.SoftwareTypeApplication,
						},
						{
							InventoryNode: createTestInventoryNode("safety-domain", 3, "NOTdomain", "safety-domain"),
							Type:          types.SoftwareTypeApplication,
						},
					},
				},
			},
			expected: &types.Inventory{
				HardwareNodes: sampleTestHardwareNode,
				SoftwareNodes: []*types.SoftwareNode{
					mainInventoryNode,
					{
						InventoryNode: createTestInventoryNode("safety-domain", 2, "NOTdomain", "safety-domain"),
						Type:          types.SoftwareTypeApplication,
					},
					{
						InventoryNode: createTestInventoryNode("safety-domain", 3, "NOTdomain", "safety-domain"),
						Type:          types.SoftwareTypeApplication,
					},
				},
				Associations: []*types.Association{
					{
						SourceID: "device-update-manager",
						TargetID: "safety-domain-test:2",
					},
				},
			},
		},
		"test_parameter_key_domain": {
			agrVersion: "2.0.0",
			domainsInventory: map[string]*types.Inventory{
				"safety-domain": {
					HardwareNodes: sampleTestHardwareNode,
					SoftwareNodes: []*types.SoftwareNode{
						{
							InventoryNode: createTestInventoryNode("safety-domain", 2, "domain", "safety-domain"),
							Type:          types.SoftwareTypeApplication,
						},
						{
							InventoryNode: createTestInventoryNode("safety-domain", 3, "", ""),
							Type:          types.SoftwareTypeApplication,
						},
					},
					Associations: []*types.Association{
						{
							SourceID: "safety-domain-test:2",
							TargetID: "safety-domain-test:3",
						},
					},
				},
			},
			expected: &types.Inventory{
				HardwareNodes: sampleTestHardwareNode,
				SoftwareNodes: []*types.SoftwareNode{
					mainInventoryNode,
					{
						InventoryNode: createTestInventoryNode("safety-domain", 2, "domain", "safety-domain"),
						Type:          types.SoftwareTypeApplication,
					},
					{
						InventoryNode: createTestInventoryNode("safety-domain", 3, "", ""),
						Type:          types.SoftwareTypeApplication,
					},
				},
				Associations: []*types.Association{
					{
						SourceID: "device-update-manager",
						TargetID: "safety-domain-test:2",
					},
					{
						SourceID: "safety-domain-test:2",
						TargetID: "safety-domain-test:3",
					},
				},
			},
		},
		"test_parameter_key_domain_on_multiple_domainsInventory": {
			agrVersion: "2.0.0",
			domainsInventory: map[string]*types.Inventory{
				"safety-domain": {
					HardwareNodes: sampleTestHardwareNode,
					SoftwareNodes: []*types.SoftwareNode{
						{
							InventoryNode: createTestInventoryNode("safety-domain", 2, "domain", "safety-domain"),
							Type:          types.SoftwareTypeApplication,
						},
						{
							InventoryNode: createTestInventoryNode("safety-domain", 3, "", ""),
							Type:          types.SoftwareTypeApplication,
						},
					},
					Associations: []*types.Association{
						{
							SourceID: "safety-domain-test:2",
							TargetID: "safety-domain-test:3",
						},
					},
				},
				"containers": {
					HardwareNodes: sampleTestHardwareNode,
					SoftwareNodes: []*types.SoftwareNode{
						{
							InventoryNode: createTestInventoryNode("containers", 2, "domain", "containers"),
							Type:          types.SoftwareTypeApplication,
						},
						{
							InventoryNode: createTestInventoryNode("containers", 3, "", ""),
							Type:          types.SoftwareTypeApplication,
						},
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
					sampleTestHardwareNode[0],
					sampleTestHardwareNode[0],
				},
				SoftwareNodes: []*types.SoftwareNode{
					mainInventoryNode,
					{
						InventoryNode: createTestInventoryNode("safety-domain", 2, "domain", "safety-domain"),
						Type:          types.SoftwareTypeApplication,
					},
					{
						InventoryNode: createTestInventoryNode("safety-domain", 3, "", ""),
						Type:          types.SoftwareTypeApplication,
					},
					{
						InventoryNode: createTestInventoryNode("containers", 2, "domain", "containers"),
						Type:          types.SoftwareTypeApplication,
					},
					{
						InventoryNode: createTestInventoryNode("containers", 3, "", ""),
						Type:          types.SoftwareTypeApplication,
					},
				},
				Associations: []*types.Association{
					{
						SourceID: "device-update-manager",
						TargetID: "safety-domain-test:2",
					},
					{
						SourceID: "safety-domain-test:2",
						TargetID: "safety-domain-test:3",
					},
					{
						SourceID: "device-update-manager",
						TargetID: "containers-test:2",
					},
					{
						SourceID: "containers-test:2",
						TargetID: "containers-test:3",
					},
				},
			},
		},
		"test_parameter_value_not_equal_domain": {
			agrVersion: "2.0.0",
			domainsInventory: map[string]*types.Inventory{
				"safety-domain": {
					HardwareNodes: sampleTestHardwareNode,
					SoftwareNodes: []*types.SoftwareNode{
						{
							InventoryNode: createTestInventoryNode("safety-domain", 2, "domain", "containers"),
							Type:          types.SoftwareTypeApplication,
						},
						{
							InventoryNode: createTestInventoryNode("safety-domain", 3, "domain", "containers"),
							Type:          types.SoftwareTypeApplication,
						},
					},
				},
			},
			expected: &types.Inventory{
				HardwareNodes: sampleTestHardwareNode,
				SoftwareNodes: []*types.SoftwareNode{
					mainInventoryNode,
					{
						InventoryNode: createTestInventoryNode("safety-domain", 2, "domain", "containers"),
						Type:          types.SoftwareTypeApplication,
					},
					{
						InventoryNode: createTestInventoryNode("safety-domain", 3, "domain", "containers"),
						Type:          types.SoftwareTypeApplication,
					},
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
			agrVersion: "2.0.0",
			domainsInventory: map[string]*types.Inventory{
				"safety-domain": {
					HardwareNodes: sampleTestHardwareNode,
					SoftwareNodes: []*types.SoftwareNode{
						{
							InventoryNode: createTestInventoryNode("safety-domain", 2, "domain", "safety-domain"),
							Type:          types.SoftwareTypeApplication,
						},
						{
							InventoryNode: createTestInventoryNode("safety-domain", 3, "", ""),
							Type:          types.SoftwareTypeApplication,
						},
						{
							InventoryNode: createTestInventoryNode("safety-domain", 4, "", ""),
							Type:          types.SoftwareTypeApplication,
						},
					},
					Associations: []*types.Association{
						{
							SourceID: "safety-domain-test:2",
							TargetID: "safety-domain-test:3",
						},
						{
							SourceID: "safety-domain-test:3",
							TargetID: "safety-domain-test:4",
						},
					},
				},
			},
			expected: &types.Inventory{
				HardwareNodes: sampleTestHardwareNode,
				SoftwareNodes: []*types.SoftwareNode{
					mainInventoryNode,
					{
						InventoryNode: createTestInventoryNode("safety-domain", 2, "domain", "safety-domain"),
						Type:          types.SoftwareTypeApplication,
					},
					{
						InventoryNode: createTestInventoryNode("safety-domain", 3, "", ""),
						Type:          types.SoftwareTypeApplication,
					},
					{
						InventoryNode: createTestInventoryNode("safety-domain", 4, "", ""),
						Type:          types.SoftwareTypeApplication,
					},
				},
				Associations: []*types.Association{
					{
						SourceID: "device-update-manager",
						TargetID: "safety-domain-test:2",
					},
					{
						SourceID: "safety-domain-test:2",
						TargetID: "safety-domain-test:3",
					},
					{
						SourceID: "safety-domain-test:3",
						TargetID: "safety-domain-test:4",
					},
				},
			},
		},
	}

	for testName, testCase := range testCases {
		t.Run(testName, func(t *testing.T) {
			agrUpdMgr := aggregatedUpdateManager{
				name: "device",
			}
			agrUpdMgr.version = testCase.agrVersion

			ret := toFullInventory(agrUpdMgr.asSoftwareNode(), testCase.domainsInventory)

			assert.True(t, compareSlicesWithoutOrder(testCase.expected.HardwareNodes, ret.HardwareNodes), "HardwereNodes not equal")
			assert.True(t, compareSlicesWithoutOrder(testCase.expected.SoftwareNodes, ret.SoftwareNodes), "SoftwareNodes not equal")
			assert.True(t, compareSlicesWithoutOrder(testCase.expected.Associations, ret.Associations), "Associations not equal")

		})
	}
}

func TestUpdateInventoryForDomain(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	ctx := context.Background()
	activityID := "activity123"
	agent := mocks.NewMockUpdateManager(mockCtrl)

	t.Run("test_successful_Get_err_nil_inventory_not_nil", func(t *testing.T) {
		var wg sync.WaitGroup
		domainsInventory := make(map[string]*types.Inventory)

		testInventory := &types.Inventory{
			Associations: []*types.Association{
				{
					SourceID: "testSourceId",
					TargetID: "testTargetId",
				},
			},
		}
		agent.EXPECT().Get(ctx, activityID).Return(testInventory, nil)
		agent.EXPECT().Name().Return("testName").Times(2)

		wg.Add(1)
		updateInventoryForDomain(ctx, &wg, activityID, agent, domainsInventory)
		wg.Wait()
		assert.Equal(t, testInventory, domainsInventory["testName"])
	})

	t.Run("test_Get_err_notNil_testInventory_nil", func(t *testing.T) {
		var wg sync.WaitGroup
		domainsInventory := make(map[string]*types.Inventory)

		agent.EXPECT().Get(ctx, activityID).Return(nil, fmt.Errorf("errNotNil"))
		agent.EXPECT().Name().Return("testName").Times(1)

		wg.Add(1)
		updateInventoryForDomain(ctx, &wg, activityID, agent, domainsInventory)
		wg.Wait()

		assert.Nil(t, domainsInventory["testName"])
		assert.Empty(t, domainsInventory)
	})

	t.Run("test_Get_err_nil_inventory_nil", func(t *testing.T) {
		var wg sync.WaitGroup
		domainsInventory := make(map[string]*types.Inventory)

		agent.EXPECT().Get(ctx, activityID).Return(nil, nil)
		agent.EXPECT().Name().Return("testName").Times(1)

		wg.Add(1)
		updateInventoryForDomain(ctx, &wg, activityID, agent, domainsInventory)
		wg.Wait()

		assert.Nil(t, domainsInventory["testName"])
	})

	t.Run("test_Get_err_notNil_intentory_notNil", func(t *testing.T) {
		var wg sync.WaitGroup
		domainsInventory := make(map[string]*types.Inventory)
		testInventory := &types.Inventory{
			Associations: []*types.Association{
				{
					SourceID: "testSourceId",
				},
			},
		}

		agent.EXPECT().Get(ctx, activityID).Return(testInventory, fmt.Errorf("errNotNil"))
		agent.EXPECT().Name().Return("testName").Times(2)

		wg.Add(1)
		updateInventoryForDomain(ctx, &wg, activityID, agent, domainsInventory)
		wg.Wait()

		assert.Equal(t, testInventory, domainsInventory["testName"])
	})
}

func createTestInventoryNode(domain string, number int, key string, value string) types.InventoryNode {
	num := strconv.FormatInt(int64(number), 10)

	return types.InventoryNode{
		ID:      domain + "-test:" + num,
		Version: "1.0.0",
		Name:    "testName" + num,
		Parameters: []*types.KeyValuePair{
			{
				Key:   key,
				Value: value,
			},
		},
	}
}

func compareSlicesWithoutOrder(expected, actual interface{}) bool {
	expectedValue := reflect.ValueOf(expected)
	actualValue := reflect.ValueOf(actual)

	if expectedValue.Len() != actualValue.Len() {
		return false
	}
	for a := 0; a < expectedValue.Len(); a++ {
		present := false
		for b := 0; b < actualValue.Len(); b++ {
			if reflect.DeepEqual(expectedValue.Index(a).Interface(), actualValue.Index(b).Interface()) {
				present = true
				break
			}
		}
		if !present {
			return false
		}
	}
	return true
}
