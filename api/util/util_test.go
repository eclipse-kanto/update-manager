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

package util

import (
	"testing"
	"time"

	"github.com/eclipse-kanto/update-manager/api/types"

	"github.com/stretchr/testify/assert"
)

func TestParseDuration(t *testing.T) {
	defDuration := time.Duration(123) * time.Second
	emptyValue := time.Duration(400)

	tests := []struct {
		property string
		value    string
		expected time.Duration
	}{
		{"testValidDuration", "15m", time.Duration(15) * time.Minute},
		{"testValidDurationZero", "0s", time.Duration(0)},
		{"testValidDurationMillis", "10ms", time.Duration(10) * time.Millisecond},
		{"testInvalidDuration", "OneSecond", defDuration},
		{"testNegativeDuration", "-10s", defDuration},
		{"testEmptyDuration", "", emptyValue},
	}

	for _, test := range tests {
		result := ParseDuration(test.property, test.value, defDuration, emptyValue)
		assert.Equal(t, test.expected, result, test.property)
	}
}

func TestFixIncompleteInconsistentStatus(t *testing.T) {

	tests := []struct {
		status   types.StatusType
		expected types.StatusType
	}{
		{types.StatusIdentifying, types.StatusIdentifying},
		{types.StatusIdentified, types.StatusIdentified},
		{types.StatusIdentificationFailed, types.StatusIdentificationFailed},
		{types.StatusRunning, types.StatusRunning},
		{types.StatusDownloadCompleted, types.StatusDownloadCompleted},
		{types.StatusDownloadFailed, types.StatusDownloadFailed},
		{types.StatusUpdateCompleted, types.StatusUpdateCompleted},
		{types.StatusUpdateFailed, types.StatusUpdateFailed},
		{types.StatusActivationCompleted, types.StatusActivationCompleted},
		{types.StatusActivationFailed, types.StatusActivationFailed},
		{types.StatusCompleted, types.StatusCompleted},
		{types.StatusIncomplete, types.StatusIncomplete},
		{types.StatusIncompleteInconsistent, types.StatusIncomplete},
		{types.StatusSuperseded, types.StatusSuperseded},
	}

	for _, test := range tests {
		assert.Equal(t, test.expected, FixIncompleteInconsistentStatus(test.status))
	}
}

func TestFixActivationActionStatus(t *testing.T) {

	tests := []struct {
		status   types.ActionStatusType
		expected types.ActionStatusType
	}{
		{types.ActionStatusIdentified, types.ActionStatusIdentified},
		{types.ActionStatusDownloading, types.ActionStatusDownloading},
		{types.ActionStatusDownloadFailure, types.ActionStatusDownloadFailure},
		{types.ActionStatusDownloadSuccess, types.ActionStatusDownloadSuccess},
		{types.ActionStatusUpdating, types.ActionStatusUpdating},
		{types.ActionStatusUpdateFailure, types.ActionStatusUpdateFailure},
		{types.ActionStatusUpdateSuccess, types.ActionStatusUpdateSuccess},
		{types.ActionStatusRemoving, types.ActionStatusRemoving},
		{types.ActionStatusRemovalFailure, types.ActionStatusRemovalFailure},
		{types.ActionStatusRemovalSuccess, types.ActionStatusRemovalSuccess},
		{types.ActionStatusActivating, types.ActionStatusUpdating},
		{types.ActionStatusActivationFailure, types.ActionStatusUpdateFailure},
		{types.ActionStatusActivationSuccess, types.ActionStatusUpdateSuccess},
	}

	for i, test := range tests {
		testAction := &types.Action{
			Status: test.status,
			Component: &types.Component{
				ID:      "test-component",
				Version: "1.2.3",
			},
			Progress: uint8(i),
			Message:  string(test.status),
		}
		resultAction := FixActivationActionStatus(testAction)
		assert.Equal(t, test.expected, resultAction.Status)
		if test.status == test.expected {
			assert.Equal(t, testAction, resultAction)
		}
	}

}
