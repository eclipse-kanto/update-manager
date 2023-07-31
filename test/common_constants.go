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
	"time"

	"github.com/eclipse-kanto/update-manager/api/types"
)

//ActivityID test constant
const ActivityID = "testActivityId"

//Interval test constant
const Interval = 1 * time.Second

//Inventory test constant
var Inventory = &types.Inventory{
	SoftwareNodes: []*types.SoftwareNode{
		MainInventoryNode,
	},
}

//MainInventoryNode test constant
var MainInventoryNode = &types.SoftwareNode{
	InventoryNode: types.InventoryNode{
		ID:      "device-update-manager",
		Version: "development",
		Name:    "Update Manager",
	},
	Type: types.SoftwareTypeApplication,
}

//SampleTestHardwareNode test constant
var SampleTestHardwareNode = []*types.HardwareNode{
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
