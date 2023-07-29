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
	"github.com/eclipse-kanto/update-manager/test"
	"github.com/eclipse-kanto/update-manager/test/mocks"

	"github.com/golang/mock/gomock"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func TestHandleCurrentStateEvent(t *testing.T) {
	mockCtr := gomock.NewController(t)
	defer mockCtr.Finish()

	t.Run("test_no_activity_id_without_delay", func(t *testing.T) {
		mockClient := mocks.NewMockUpdateAgentClient(mockCtr)
		mockClient.EXPECT().SendCurrentState("", test.Inventory)

		updAgent := &updateAgent{
			client: mockClient,
		}

		updAgent.HandleCurrentStateEvent("testDomain", "", test.Inventory)
	})

	t.Run("test_no_activity_id_with_delay", func(t *testing.T) {
		mockClient := mocks.NewMockUpdateAgentClient(mockCtr)
		updAgent := &updateAgent{
			client:                  mockClient,
			currentStateReportDelay: test.Interval,
		}

		ch := make(chan bool, 1)
		mockClient.EXPECT().SendCurrentState("", test.Inventory).DoAndReturn(
			func(activityID string, inventory *types.Inventory) error {
				ch <- true
				return nil
			})
		updAgent.HandleCurrentStateEvent("testDomain", "", test.Inventory)
		<-ch
	})

	t.Run("test_activity_id_not_empty", func(t *testing.T) {
		mockClient := mocks.NewMockUpdateAgentClient(mockCtr)
		mockClient.EXPECT().SendCurrentState(test.TestActivityID, test.Inventory)
		updAgent := &updateAgent{
			client:                  mockClient,
			currentStateReportDelay: test.Interval,
		}
		updAgent.HandleCurrentStateEvent("testDomain", test.TestActivityID, test.Inventory)
	})

	t.Run("test_current_state_notifier_not_nil", func(t *testing.T) {
		mockClient := mocks.NewMockUpdateAgentClient(mockCtr)
		mockClient.EXPECT().SendCurrentState(test.TestActivityID, test.Inventory).Return(nil)
		csNotifier := &currentStateNotifier{
			internalTimer: time.AfterFunc(time.Millisecond, nil),
		}
		updAgent := &updateAgent{
			client:                  mockClient,
			currentStateReportDelay: test.Interval,
			currentStateNotifier:    csNotifier,
		}
		updAgent.HandleCurrentStateEvent("testDomain", test.TestActivityID, test.Inventory)
		assert.Nil(t, updAgent.currentStateNotifier)
	})

	t.Run("test_current_state_send_error", func(t *testing.T) {
		mockClient := mocks.NewMockUpdateAgentClient(mockCtr)
		mockClient.EXPECT().SendCurrentState(test.TestActivityID, test.Inventory).Return(errors.New("send current state error"))

		updAgent := &updateAgent{
			client:                  mockClient,
			currentStateReportDelay: test.Interval,
		}
		updAgent.HandleCurrentStateEvent("testDomain", test.TestActivityID, test.Inventory)
	})
}
