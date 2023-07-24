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

	"github.com/eclipse-kanto/update-manager/api/types"
)

const testActivityID = "testActivityId"
const interval = 1 * time.Second

var inventory = &types.Inventory{
	SoftwareNodes: []*types.SoftwareNode{
		{
			InventoryNode: types.InventoryNode{
				ID:      "update-manager",
				Version: "development",
				Name:    "Update Manager",
			},
			Type: "APPLICATION",
		},
	},
}
