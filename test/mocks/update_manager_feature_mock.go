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

// Code generated by MockGen. DO NOT EDIT.
// Source: ./things/update_manager_feature.go

// Package mocks is a generated GoMock package.
package mocks

import (
	reflect "reflect"

	types "github.com/eclipse-kanto/update-manager/api/types"
	gomock "github.com/golang/mock/gomock"
)

// MockUpdateManagerFeature is a mock of UpdateManagerFeature interface.
type MockUpdateManagerFeature struct {
	ctrl     *gomock.Controller
	recorder *MockUpdateManagerFeatureMockRecorder
}

// MockUpdateManagerFeatureMockRecorder is the mock recorder for MockUpdateManagerFeature.
type MockUpdateManagerFeatureMockRecorder struct {
	mock *MockUpdateManagerFeature
}

// NewMockUpdateManagerFeature creates a new mock instance.
func NewMockUpdateManagerFeature(ctrl *gomock.Controller) *MockUpdateManagerFeature {
	mock := &MockUpdateManagerFeature{ctrl: ctrl}
	mock.recorder = &MockUpdateManagerFeatureMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockUpdateManagerFeature) EXPECT() *MockUpdateManagerFeatureMockRecorder {
	return m.recorder
}

// Activate mocks base method.
func (m *MockUpdateManagerFeature) Activate() error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Activate")
	ret0, _ := ret[0].(error)
	return ret0
}

// Activate indicates an expected call of Activate.
func (mr *MockUpdateManagerFeatureMockRecorder) Activate() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Activate", reflect.TypeOf((*MockUpdateManagerFeature)(nil).Activate))
}

// Deactivate mocks base method.
func (m *MockUpdateManagerFeature) Deactivate() {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "Deactivate")
}

// Deactivate indicates an expected call of Deactivate.
func (mr *MockUpdateManagerFeatureMockRecorder) Deactivate() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Deactivate", reflect.TypeOf((*MockUpdateManagerFeature)(nil).Deactivate))
}

// SendFeedback mocks base method.
func (m *MockUpdateManagerFeature) SendFeedback(activityID string, desiredStateFeedback *types.DesiredStateFeedback) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SendFeedback", activityID, desiredStateFeedback)
	ret0, _ := ret[0].(error)
	return ret0
}

// SendFeedback indicates an expected call of SendFeedback.
func (mr *MockUpdateManagerFeatureMockRecorder) SendFeedback(activityID, desiredStateFeedback interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SendFeedback", reflect.TypeOf((*MockUpdateManagerFeature)(nil).SendFeedback), activityID, desiredStateFeedback)
}

// SetState mocks base method.
func (m *MockUpdateManagerFeature) SetState(activityID string, currentState *types.Inventory) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SetState", activityID, currentState)
	ret0, _ := ret[0].(error)
	return ret0
}

// SetState indicates an expected call of SetState.
func (mr *MockUpdateManagerFeatureMockRecorder) SetState(activityID, currentState interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetState", reflect.TypeOf((*MockUpdateManagerFeature)(nil).SetState), activityID, currentState)
}
