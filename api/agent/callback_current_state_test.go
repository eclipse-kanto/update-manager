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

	"github.com/eclipse-kanto/update-manager/api/types"
	"github.com/eclipse-kanto/update-manager/test/mocks"

	"github.com/golang/mock/gomock"
)

func TestHandleCurrentStateEvent(t *testing.T) {
	mockCtr := gomock.NewController(t)
	defer mockCtr.Finish()

	inventory := &types.Inventory{
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

	t.Run("test_no_activity_id_without_delay", func(t *testing.T) {
		mockClient := mocks.NewMockUpdateAgentClient(mockCtr)
		mockClient.EXPECT().SendCurrentState("", inventory)

		updAgent := &updateAgent{
			client: mockClient,
		}

		updAgent.HandleCurrentStateEvent("testDomain", "", inventory)
	})

	t.Run("test_no_activity_id_with_delay", func(t *testing.T) {
		mockClient := mocks.NewMockUpdateAgentClient(mockCtr)
		updAgent := &updateAgent{
			client:                  mockClient,
			currentStateReportDelay: time.Second,
		}

		ch := make(chan bool, 1)
		mockClient.EXPECT().SendCurrentState("", inventory).DoAndReturn(
			func(activityID string, inventory *types.Inventory) error {
				ch <- true
				return nil
			})
		updAgent.HandleCurrentStateEvent("testDomain", "", inventory)
		<-ch
	})

	t.Run("test_activity_id_not_empty", func(t *testing.T) {
		mockClient := mocks.NewMockUpdateAgentClient(mockCtr)
		mockClient.EXPECT().SendCurrentState(testActivityID, inventory)
		updAgent := &updateAgent{
			client:                  mockClient,
			currentStateReportDelay: time.Minute,
		}
		updAgent.HandleCurrentStateEvent("testDomain", testActivityID, inventory)
	})
}
