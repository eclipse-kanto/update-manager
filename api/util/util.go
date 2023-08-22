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
	"time"

	"github.com/eclipse-kanto/update-manager/api/types"
	"github.com/eclipse-kanto/update-manager/logger"
)

// ParseDuration converts the given string as time duration value, or returns defValue in case of an error or emptyValue if property is not set.
func ParseDuration(property, strValue string, defValue, emptyValue time.Duration) time.Duration {
	if strValue == "" {
		logger.Warn("Duration for property '%s' not set, using value %v", property, emptyValue)
		return emptyValue
	}
	duration, err := time.ParseDuration(strValue)
	if err != nil {
		logger.Warn("Cannot parse duration for property '%s': %v, using default value %v", property, strValue, defValue)
		return defValue
	}
	if duration < 0 {
		logger.Warn("Negative duration for property '%s': %v, using default value %v", property, strValue, defValue)
		return defValue
	}
	return duration
}

// FixIncompleteInconsistentStatus replaces StatusIncompleteInconsistent with StatusIncomplete, just a temporary workaround.
func FixIncompleteInconsistentStatus(status types.StatusType) types.StatusType {
	switch status {
	case types.StatusIncompleteInconsistent:
		return types.StatusIncomplete
	default:
		return status
	}
}

// FixActivationActionStatus replaces ActionStatusActivating, ActionStatusActivationSuccess, ActionStatusActivationFailure with ActionStatusUpdating, ActionStatusUpdateSuccess, ActionStatusUpdateFailure, just a temporary workaround.
func FixActivationActionStatus(action *types.Action) *types.Action {
	var actionStatus types.ActionStatusType
	switch action.Status {
	case types.ActionStatusActivating:
		actionStatus = types.ActionStatusUpdating
	case types.ActionStatusActivationSuccess:
		actionStatus = types.ActionStatusUpdateSuccess
	case types.ActionStatusActivationFailure:
		actionStatus = types.ActionStatusUpdateFailure
	default:
		return action
	}
	return &types.Action{
		Component: action.Component,
		Status:    actionStatus,
		Progress:  action.Progress,
		Message:   action.Message,
	}
}
