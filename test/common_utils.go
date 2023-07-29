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

package test

import (
	"fmt"
	"reflect"
	"strconv"
	"testing"

	"github.com/eclipse-kanto/update-manager/api"
	"github.com/eclipse-kanto/update-manager/api/types"
	"github.com/eclipse-kanto/update-manager/config"

	"github.com/stretchr/testify/assert"
)

func AssertInventoryWithoutElementsOrder(t *testing.T, inv1, inv2 *types.Inventory) {
	if inv1 == nil || inv2 == nil {
		assert.True(t, inv1 == inv2)
	}
	assert.True(t, compareSlicesWithoutOrder(inv1.HardwareNodes, inv2.HardwareNodes), "HardwereNodes not equal")
	assert.True(t, compareSlicesWithoutOrder(inv1.SoftwareNodes, inv2.SoftwareNodes), "SoftwareNodes not equal")
	assert.True(t, compareSlicesWithoutOrder(inv1.Associations, inv2.Associations), "Associations not equal")
}

func CreateSoftwareNode(domain string, number int, key string, value string, swType types.SoftwareType) *types.SoftwareNode {
	num := strconv.FormatInt(int64(number), 10)

	return &types.SoftwareNode{
		InventoryNode: types.InventoryNode{
			ID:      fmt.Sprintf("%s-test:%d", domain, number),
			Version: "1.0.0",
			Name:    "testName" + num,
			Parameters: []*types.KeyValuePair{
				{
					Key:   key,
					Value: value,
				},
			},
		},
		Type: swType,
	}
}

func CreateTestConfig(rebootRequired, rebootEnabled bool) *config.Config {
	agents := make(map[string]*api.UpdateManagerConfig)

	for i := 1; i < 4; i++ {
		agents[fmt.Sprintf("testDomain%d", i)] = &api.UpdateManagerConfig{
			Name:           fmt.Sprintf("testDomain%d", i),
			RebootRequired: rebootRequired,
			ReadTimeout:    "1s",
		}
	}

	return &config.Config{
		BaseConfig: &config.BaseConfig{
			Domain: "device",
		},
		Agents:        agents,
		RebootEnabled: rebootEnabled,
	}
}

func CreateAssociation(sourceID, targetID string) *types.Association {
	return &types.Association{
		SourceID: sourceID,
		TargetID: targetID,
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
