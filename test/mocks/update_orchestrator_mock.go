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
// Source: api/update_orchestrator.go

// Package mocks is a generated GoMock package.
package mocks

import (
	context "context"
	reflect "reflect"

	api "github.com/eclipse-kanto/update-manager/api"
	types "github.com/eclipse-kanto/update-manager/api/types"
	gomock "github.com/golang/mock/gomock"
)

// MockUpdateOrchestrator is a mock of UpdateOrchestrator interface.
type MockUpdateOrchestrator struct {
	ctrl     *gomock.Controller
	recorder *MockUpdateOrchestratorMockRecorder
}

// MockUpdateOrchestratorMockRecorder is the mock recorder for MockUpdateOrchestrator.
type MockUpdateOrchestratorMockRecorder struct {
	mock *MockUpdateOrchestrator
}

// NewMockUpdateOrchestrator creates a new mock instance.
func NewMockUpdateOrchestrator(ctrl *gomock.Controller) *MockUpdateOrchestrator {
	mock := &MockUpdateOrchestrator{ctrl: ctrl}
	mock.recorder = &MockUpdateOrchestratorMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockUpdateOrchestrator) EXPECT() *MockUpdateOrchestratorMockRecorder {
	return m.recorder
}

// Apply mocks base method.
func (m *MockUpdateOrchestrator) Apply(arg0 context.Context, arg1 map[string]api.UpdateManager, arg2 string, arg3 *types.DesiredState, arg4 api.DesiredStateFeedbackHandler) bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Apply", arg0, arg1, arg2, arg3, arg4)
	ret0, _ := ret[0].(bool)
	return ret0
}

// Apply indicates an expected call of Apply.
func (mr *MockUpdateOrchestratorMockRecorder) Apply(arg0, arg1, arg2, arg3, arg4 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Apply", reflect.TypeOf((*MockUpdateOrchestrator)(nil).Apply), arg0, arg1, arg2, arg3, arg4)
}

// HandleDesiredStateFeedbackEvent mocks base method.
func (m *MockUpdateOrchestrator) HandleDesiredStateFeedbackEvent(domain, activityID, baseline string, status types.StatusType, message string, actions []*types.Action) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "HandleDesiredStateFeedbackEvent", domain, activityID, baseline, status, message, actions)
}

// HandleDesiredStateFeedbackEvent indicates an expected call of HandleDesiredStateFeedbackEvent.
func (mr *MockUpdateOrchestratorMockRecorder) HandleDesiredStateFeedbackEvent(domain, activityID, baseline, status, message, actions interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "HandleDesiredStateFeedbackEvent", reflect.TypeOf((*MockUpdateOrchestrator)(nil).HandleDesiredStateFeedbackEvent), domain, activityID, baseline, status, message, actions)
}

// HandleOwnerConsentFeedback mocks base method.
func (m *MockUpdateOrchestrator) HandleOwnerConsentFeedback(arg0 string, arg1 int64, arg2 *types.OwnerConsentFeedback) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "HandleOwnerConsent", arg0, arg1, arg2)
	ret0, _ := ret[0].(error)
	return ret0
}

// HandleOwnerConsentFeedback indicates an expected call of HandleOwnerConsentFeedback.
func (mr *MockUpdateOrchestratorMockRecorder) HandleOwnerConsentFeedback(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "HandleOwnerConsentFeedback", reflect.TypeOf((*MockUpdateOrchestrator)(nil).HandleOwnerConsentFeedback), arg0, arg1, arg2)
}
