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
	"testing"

	"github.com/eclipse-kanto/update-manager/api"
	"github.com/eclipse-kanto/update-manager/api/types"
	"github.com/eclipse-kanto/update-manager/test"
	"github.com/eclipse-kanto/update-manager/test/mocks"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

var testOperationActions = map[string]map[string]*types.Action{
	"testDomain1": {
		"action": {
			Component: &types.Component{
				ID: "componentId",
			},
		},
	},
}
var testUpdateActions = []*types.Action{
	{
		Component: &types.Component{
			ID: "component-1",
		},
	},
}

func TestFeedbackHandleDesiredStateFeedbackEvent(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	eventCallback := mocks.NewMockUpdateManagerCallback(mockCtrl)
	mockUpdateManager := mocks.NewMockUpdateManager(mockCtrl)

	testCases := map[string]struct {
		handleStatus          types.StatusType
		updateOrchStatus      types.StatusType
		domains               map[string]types.StatusType
		expectedStatus        types.StatusType
		expectedDelayedStatus types.StatusType
		expectedDomainStatus  types.StatusType
		expectedIdentErrMsg   string
		testCode              func()
	}{
		"test_handleDomainIdentified_Success": {
			handleStatus:     types.StatusIdentified,
			updateOrchStatus: types.StatusIdentified,
			domains: map[string]types.StatusType{
				"testDomain1": types.StatusIdentifying,
			},
			expectedStatus:       types.StatusRunning,
			expectedDomainStatus: types.StatusIdentified,
			testCode: func() {
				eventCallback.EXPECT().HandleDesiredStateFeedbackEvent("device", test.ActivityID, "", types.StatusIdentified, "", gomock.Any())
				eventCallback.EXPECT().HandleDesiredStateFeedbackEvent("device", test.ActivityID, "", types.StatusRunning, "", gomock.Any())
				mockUpdateManager.EXPECT().Name().Return("testDomain1")
				mockUpdateManager.EXPECT().Command(context.Background(), test.ActivityID, generateCommand(types.CommandDownload))
			},
		},
		"test_handleDomainIdentified_domainUpdateStatus_StatusIdentificationFailed": {
			handleStatus:     types.StatusIdentified,
			updateOrchStatus: types.StatusIdentified,
			domains: map[string]types.StatusType{
				"testDomain1": types.StatusIdentifying,
				"testDomain2": types.StatusIdentificationFailed,
			},
			expectedStatus:       types.StatusIdentificationFailed,
			expectedIdentErrMsg:  "[testDomain1]: ",
			expectedDomainStatus: types.StatusIdentified,
			testCode:             func() {},
		},
		"test_handleDomainIdentified_domainUpdateStatus_StatusIdentifiying": {
			handleStatus:     types.StatusIdentified,
			updateOrchStatus: types.StatusIdentified,
			domains: map[string]types.StatusType{
				"testDomain1": types.StatusIdentifying,
				"testDomain2": types.StatusIdentifying,
			},
			expectedStatus:       types.StatusIdentified,
			expectedDomainStatus: types.StatusIdentified,
			expectedIdentErrMsg:  "",
			testCode:             func() {},
		},
		"test_handleDomainIdentified_checkIdentificationStatus": {
			handleStatus:     types.StatusIdentified,
			updateOrchStatus: types.StatusIdentified,
			domains: map[string]types.StatusType{
				"testDomain1": types.StatusIdentified,
			},
			expectedStatus:       types.StatusIdentified,
			expectedIdentErrMsg:  "",
			expectedDomainStatus: types.StatusIdentified,
			testCode:             func() {},
		},
		"test_handleDomainIdentificationFailed_Success": {
			handleStatus:     types.StatusIdentificationFailed,
			updateOrchStatus: types.StatusIdentificationFailed,
			domains: map[string]types.StatusType{
				"testDomain1": types.StatusIdentifying,
			},
			expectedStatus:       types.StatusIdentificationFailed,
			expectedDomainStatus: types.StatusIdentificationFailed,
			expectedIdentErrMsg:  "[testDomain1]: testMsg",
			testCode:             func() {},
		},
		"test_handleDomainIdentificationFailed_domainUpdateStatus_StatusIdentifying": {
			handleStatus:     types.StatusIdentificationFailed,
			updateOrchStatus: types.StatusIdentificationFailed,
			domains: map[string]types.StatusType{
				"testDomain1": types.StatusIdentifying,
				"testDomain2": types.StatusIdentifying,
			},
			expectedStatus:       types.StatusIdentificationFailed,
			expectedDomainStatus: types.StatusIdentificationFailed,
			testCode:             func() {},
		},
		"test_handleDomainIdentificationFailed_checkIdentificationStatus": {
			handleStatus:     types.StatusIdentificationFailed,
			updateOrchStatus: types.StatusIdentificationFailed,
			domains: map[string]types.StatusType{
				"testDomain1": types.StatusIdentificationFailed,
			},
			expectedStatus:       types.StatusIdentificationFailed,
			expectedDomainStatus: types.StatusIdentificationFailed,
			testCode:             func() {},
		},
		"test_handleDomainIdentificationFailed_checkIdentificationStatus_status_not_from_supported": {
			handleStatus:     types.StatusIdentificationFailed,
			updateOrchStatus: types.BaselineStatusActivationSuccess,
			domains: map[string]types.StatusType{
				"testDomain1": types.StatusIdentificationFailed,
			},
			expectedStatus:       types.BaselineStatusActivationSuccess,
			expectedDomainStatus: types.StatusIdentificationFailed,
			testCode:             func() {},
		},
		"test_handleDomainCompletedEvent_Success": {
			handleStatus:     types.StatusCompleted,
			updateOrchStatus: types.StatusRunning,
			domains: map[string]types.StatusType{
				"testDomain1": types.StatusRunning,
			},
			expectedStatus:       types.StatusRunning,
			expectedDomainStatus: types.StatusCompleted,
			testCode: func() {
				mockUpdateManager.EXPECT().Name().Return("testDomain1")
				mockUpdateManager.EXPECT().Command(context.Background(), test.ActivityID, generateCommand(types.CommandCleanup))
			},
		},
		"test_handleDomainCompletedEvent_orchestration_status_not_running": {
			handleStatus:     types.StatusCompleted,
			updateOrchStatus: types.StatusIncomplete,
			domains: map[string]types.StatusType{
				"testDomain1": types.StatusRunning,
			},
			expectedStatus:       types.StatusIncomplete,
			expectedDomainStatus: types.StatusRunning,
			testCode:             func() {},
		},
		"test_handleDomainCompletedEvent_domain_status_not_from_supported": {
			handleStatus:     types.StatusCompleted,
			updateOrchStatus: types.StatusRunning,
			domains: map[string]types.StatusType{
				"testDomain1": types.StatusCompleted,
			},
			expectedStatus:       types.StatusRunning,
			expectedDomainStatus: types.StatusCompleted,
			testCode:             func() {},
		},
		"test_handleDomainIncomplete_Success": {
			handleStatus:     types.StatusIncomplete,
			updateOrchStatus: types.StatusRunning,
			domains: map[string]types.StatusType{
				"testDomain1": types.StatusRunning,
			},
			expectedStatus:        types.StatusRunning,
			expectedDelayedStatus: types.StatusIncomplete,
			expectedDomainStatus:  types.StatusIncomplete,
			testCode: func() {
				mockUpdateManager.EXPECT().Name().Return("testDomain1")
				mockUpdateManager.EXPECT().Command(context.Background(), test.ActivityID, generateCommand(types.CommandCleanup))
			},
		},
		"test_handleDomainIncomplete_orchestration_status_not_running": {
			handleStatus:     types.StatusIncomplete,
			updateOrchStatus: types.StatusIncomplete,
			domains: map[string]types.StatusType{
				"testDomain1": types.StatusRunning,
			},
			expectedStatus:       types.StatusIncomplete,
			expectedDomainStatus: types.StatusRunning,
			testCode:             func() {},
		},
		"test_handleDomainIncomplete_domain_status_not_from_supported": {
			handleStatus:     types.StatusIncomplete,
			updateOrchStatus: types.StatusRunning,
			domains: map[string]types.StatusType{
				"testDomain1": types.StatusCompleted,
			},
			expectedStatus:       types.StatusRunning,
			expectedDomainStatus: types.StatusCompleted,
			testCode:             func() {},
		},
		"test_handleDomainDownloading_Success": {
			handleStatus:     types.BaselineStatusDownloading,
			updateOrchStatus: types.StatusRunning,
			domains: map[string]types.StatusType{
				"testDomain1": types.StatusIdentified,
			},
			expectedStatus:       types.StatusRunning,
			expectedDomainStatus: types.StatusIdentified,
			testCode: func() {
				eventCallback.EXPECT().HandleDesiredStateFeedbackEvent("device", test.ActivityID, "", types.StatusRunning, "", gomock.Any())
			},
		},
		"test_handleDomainDownloading_orchestration_status_not_running": {
			handleStatus:     types.BaselineStatusDownloading,
			updateOrchStatus: types.StatusIncomplete,
			domains: map[string]types.StatusType{
				"testDomain1": types.StatusRunning,
			},
			expectedStatus:       types.StatusIncomplete,
			expectedDomainStatus: types.StatusRunning,
			testCode:             func() {},
		},
		"test_handleDomainDownloading_domain_status_not_from_supported": {
			handleStatus:     types.BaselineStatusDownloading,
			updateOrchStatus: types.StatusRunning,
			domains: map[string]types.StatusType{
				"testDomain1": types.StatusCompleted,
			},
			expectedStatus:       types.StatusRunning,
			expectedDomainStatus: types.StatusCompleted,
			testCode:             func() {},
		},
		"test_handleDomainDownloadSuccess_Success": {
			handleStatus:     types.BaselineStatusDownloadSuccess,
			updateOrchStatus: types.StatusRunning,
			domains: map[string]types.StatusType{
				"testDomain1": types.StatusIdentified,
			},
			expectedStatus:       types.StatusRunning,
			expectedDomainStatus: types.BaselineStatusDownloadSuccess,
			testCode: func() {
				mockUpdateManager.EXPECT().Name().Return("testDomain1")
				mockUpdateManager.EXPECT().Command(context.Background(), test.ActivityID, generateCommand(types.CommandUpdate))
				eventCallback.EXPECT().HandleDesiredStateFeedbackEvent("device", test.ActivityID, "", types.StatusRunning, "", gomock.Any())
			},
		},
		"test_handleDomainDownloadSuccess_orchestration_status_not_running": {
			handleStatus:     types.BaselineStatusDownloadSuccess,
			updateOrchStatus: types.StatusIncomplete,
			domains: map[string]types.StatusType{
				"testDomain1": types.StatusRunning,
			},
			expectedStatus:       types.StatusIncomplete,
			expectedDomainStatus: types.StatusRunning,
			testCode:             func() {},
		},
		"test_handleDomainDownloadSuccess_domain_status_not_from_supported": {
			handleStatus:     types.BaselineStatusDownloadSuccess,
			updateOrchStatus: types.StatusRunning,
			domains: map[string]types.StatusType{
				"testDomain1": types.StatusCompleted,
			},
			expectedStatus:       types.StatusRunning,
			expectedDomainStatus: types.StatusCompleted,
			testCode:             func() {},
		},
		"test_handleDomainDownloadFailure_Success": {
			handleStatus:     types.BaselineStatusDownloadFailure,
			updateOrchStatus: types.StatusRunning,
			domains: map[string]types.StatusType{
				"testDomain1": types.StatusIdentified,
			},
			expectedStatus:        types.StatusRunning,
			expectedDomainStatus:  types.BaselineStatusDownloadFailure,
			expectedDelayedStatus: types.StatusIncomplete,
			testCode: func() {
				mockUpdateManager.EXPECT().Name().Return("testDomain1")
				mockUpdateManager.EXPECT().Command(context.Background(), test.ActivityID, generateCommand(types.CommandCleanup))
			},
		},
		"test_handleDomainDownloadFailure_orchestration_status_not_running": {
			handleStatus:     types.BaselineStatusDownloadFailure,
			updateOrchStatus: types.StatusIncomplete,
			domains: map[string]types.StatusType{
				"testDomain1": types.StatusRunning,
			},
			expectedStatus:       types.StatusIncomplete,
			expectedDomainStatus: types.StatusRunning,
			testCode:             func() {},
		},
		"test_handleDomainDownloadFailure_domain_status_not_from_supported": {
			handleStatus:     types.BaselineStatusDownloadFailure,
			updateOrchStatus: types.StatusRunning,
			domains: map[string]types.StatusType{
				"testDomain1": types.StatusCompleted,
			},
			expectedStatus:       types.StatusRunning,
			expectedDomainStatus: types.StatusCompleted,
			testCode:             func() {},
		},
		"test_handleDomainUpdating_Success": {
			handleStatus:     types.BaselineStatusUpdating,
			updateOrchStatus: types.StatusRunning,
			domains: map[string]types.StatusType{
				"testDomain1": types.BaselineStatusDownloadSuccess,
			},
			expectedStatus:       types.StatusRunning,
			expectedDomainStatus: types.BaselineStatusDownloadSuccess,
			testCode: func() {
				eventCallback.EXPECT().HandleDesiredStateFeedbackEvent("device", test.ActivityID, "", types.StatusRunning, "", gomock.Any())
			},
		},
		"test_handleDomainUpdating_orchestration_status_not_running": {
			handleStatus:     types.BaselineStatusUpdating,
			updateOrchStatus: types.StatusIncomplete,
			domains: map[string]types.StatusType{
				"testDomain1": types.StatusRunning,
			},
			expectedStatus:       types.StatusIncomplete,
			expectedDomainStatus: types.StatusRunning,
			testCode:             func() {},
		},
		"test_handleDomainUpdating_domain_status_not_from_supported": {
			handleStatus:     types.BaselineStatusUpdating,
			updateOrchStatus: types.StatusRunning,
			domains: map[string]types.StatusType{
				"testDomain1": types.StatusCompleted,
			},
			expectedStatus:       types.StatusRunning,
			expectedDomainStatus: types.StatusCompleted,
			testCode:             func() {},
		},
		"test_handleDomainUpdateSuccess_Success": {
			handleStatus:     types.BaselineStatusUpdateSuccess,
			updateOrchStatus: types.StatusRunning,
			domains: map[string]types.StatusType{
				"testDomain1": types.BaselineStatusDownloadSuccess,
			},
			expectedStatus:       types.StatusRunning,
			expectedDomainStatus: types.BaselineStatusUpdateSuccess,
			testCode: func() {
				mockUpdateManager.EXPECT().Name().Return("testDomain1")
				mockUpdateManager.EXPECT().Command(context.Background(), test.ActivityID, generateCommand(types.CommandActivate))
				eventCallback.EXPECT().HandleDesiredStateFeedbackEvent("device", test.ActivityID, "", types.StatusRunning, "", gomock.Any())
			},
		},
		"test_handleDomainUpdateSuccess_orchestration_status_not_running": {
			handleStatus:     types.BaselineStatusUpdateSuccess,
			updateOrchStatus: types.StatusIncomplete,
			domains: map[string]types.StatusType{
				"testDomain1": types.StatusRunning,
			},
			expectedStatus:       types.StatusIncomplete,
			expectedDomainStatus: types.StatusRunning,
			testCode:             func() {},
		},
		"test_handleDomainUpdateSuccess_domain_status_not_from_supported": {
			handleStatus:     types.BaselineStatusUpdateSuccess,
			updateOrchStatus: types.StatusRunning,
			domains: map[string]types.StatusType{
				"testDomain1": types.StatusCompleted,
			},
			expectedStatus:       types.StatusRunning,
			expectedDomainStatus: types.StatusCompleted,
			testCode:             func() {},
		},
		"test_handleDomainUpdateFailure_Success": {
			handleStatus:     types.BaselineStatusUpdateFailure,
			updateOrchStatus: types.StatusRunning,
			domains: map[string]types.StatusType{
				"testDomain1": types.BaselineStatusDownloadSuccess,
			},
			expectedStatus:        types.StatusRunning,
			expectedDomainStatus:  types.BaselineStatusUpdateFailure,
			expectedDelayedStatus: types.StatusIncomplete,
			testCode: func() {
				mockUpdateManager.EXPECT().Name().Return("testDomain1")
				mockUpdateManager.EXPECT().Command(context.Background(), test.ActivityID, generateCommand(types.CommandCleanup))
			},
		},
		"test_handleDomainUpdateFailure_orchestration_status_not_running": {
			handleStatus:     types.BaselineStatusUpdateFailure,
			updateOrchStatus: types.StatusIncomplete,
			domains: map[string]types.StatusType{
				"testDomain1": types.StatusRunning,
			},
			expectedStatus:       types.StatusIncomplete,
			expectedDomainStatus: types.StatusRunning,
			testCode:             func() {},
		},
		"test_handleDomainUpdateFailure_domain_status_not_from_supported": {
			handleStatus:     types.BaselineStatusUpdateFailure,
			updateOrchStatus: types.StatusRunning,
			domains: map[string]types.StatusType{
				"testDomain1": types.StatusCompleted,
			},
			expectedStatus:       types.StatusRunning,
			expectedDomainStatus: types.StatusCompleted,
			testCode:             func() {},
		},
		"test_handleDomainActivating_Success": {
			handleStatus:     types.BaselineStatusActivating,
			updateOrchStatus: types.StatusRunning,
			domains: map[string]types.StatusType{
				"testDomain1": types.BaselineStatusUpdateSuccess,
			},
			expectedStatus:       types.StatusRunning,
			expectedDomainStatus: types.BaselineStatusUpdateSuccess,
			testCode: func() {
				eventCallback.EXPECT().HandleDesiredStateFeedbackEvent("device", test.ActivityID, "", types.StatusRunning, "", gomock.Any())
			},
		},
		"test_handleDomainActivating_orchestration_status_not_running": {
			handleStatus:     types.BaselineStatusActivating,
			updateOrchStatus: types.StatusIncomplete,
			domains: map[string]types.StatusType{
				"testDomain1": types.StatusRunning,
			},
			expectedStatus:       types.StatusIncomplete,
			expectedDomainStatus: types.StatusRunning,
			testCode:             func() {},
		},
		"test_handleDomainActivating_domain_status_not_from_supported": {
			handleStatus:     types.BaselineStatusActivating,
			updateOrchStatus: types.StatusRunning,
			domains: map[string]types.StatusType{
				"testDomain1": types.StatusCompleted,
			},
			expectedStatus:       types.StatusRunning,
			expectedDomainStatus: types.StatusCompleted,
			testCode:             func() {},
		},
		"test_handleDomainActivationSuccess_Success": {
			handleStatus:     types.BaselineStatusActivationSuccess,
			updateOrchStatus: types.StatusRunning,
			domains: map[string]types.StatusType{
				"testDomain1": types.BaselineStatusUpdateSuccess,
			},
			expectedStatus:       types.StatusRunning,
			expectedDomainStatus: types.BaselineStatusActivationSuccess,
			testCode: func() {
				mockUpdateManager.EXPECT().Name().Return("testDomain1")
				mockUpdateManager.EXPECT().Command(context.Background(), test.ActivityID, generateCommand(types.CommandCleanup))
				eventCallback.EXPECT().HandleDesiredStateFeedbackEvent("device", test.ActivityID, "", types.StatusRunning, "", gomock.Any())
			},
		},
		"test_handleDomainActivationSuccess_orchestration_status_not_running": {
			handleStatus:     types.BaselineStatusActivationSuccess,
			updateOrchStatus: types.StatusIncomplete,
			domains: map[string]types.StatusType{
				"testDomain1": types.StatusRunning,
			},
			expectedStatus:       types.StatusIncomplete,
			expectedDomainStatus: types.StatusRunning,
			testCode:             func() {},
		},
		"test_handleDomainActivationSuccess_domain_status_not_from_supported": {
			handleStatus:     types.BaselineStatusActivationSuccess,
			updateOrchStatus: types.StatusRunning,
			domains: map[string]types.StatusType{
				"testDomain1": types.StatusCompleted,
			},
			expectedStatus:       types.StatusRunning,
			expectedDomainStatus: types.StatusCompleted,
			testCode:             func() {},
		},
		"test_handleDomainActivationFailure_Success": {
			handleStatus:     types.BaselineStatusActivationFailure,
			updateOrchStatus: types.StatusRunning,
			domains: map[string]types.StatusType{
				"testDomain1": types.BaselineStatusUpdateSuccess,
			},
			expectedStatus:        types.StatusRunning,
			expectedDomainStatus:  types.BaselineStatusActivationFailure,
			expectedDelayedStatus: types.StatusIncomplete,
			testCode: func() {
				mockUpdateManager.EXPECT().Name().Return("testDomain1")
				mockUpdateManager.EXPECT().Command(context.Background(), test.ActivityID, generateCommand(types.CommandCleanup))
			},
		},
		"test_handleDomainActivationFailure_orchestration_status_not_running": {
			handleStatus:     types.BaselineStatusActivationFailure,
			updateOrchStatus: types.StatusIncomplete,
			domains: map[string]types.StatusType{
				"testDomain1": types.StatusRunning,
			},
			expectedStatus:       types.StatusIncomplete,
			expectedDomainStatus: types.StatusRunning,
			testCode:             func() {},
		},
		"test_handleDomainActivationFailure_domain_status_not_from_supported": {
			handleStatus:     types.BaselineStatusActivationFailure,
			updateOrchStatus: types.StatusRunning,
			domains: map[string]types.StatusType{
				"testDomain1": types.StatusCompleted,
			},
			expectedStatus:       types.StatusRunning,
			expectedDomainStatus: types.StatusCompleted,
			testCode:             func() {},
		},
		"test_handleDomainCleanup_Success": {
			handleStatus:     types.BaselineStatusCleanup,
			updateOrchStatus: types.StatusRunning,
			domains: map[string]types.StatusType{
				"testDomain1": types.BaselineStatusActivationSuccess,
			},
			expectedStatus:       types.StatusRunning,
			expectedDomainStatus: types.BaselineStatusActivationSuccess,
			testCode: func() {
				eventCallback.EXPECT().HandleDesiredStateFeedbackEvent("device", test.ActivityID, "", types.StatusRunning, "", gomock.Any())
			},
		},
		"test_handleDomainCleanup_orchestration_status_not_running": {
			handleStatus:     types.BaselineStatusCleanup,
			updateOrchStatus: types.StatusIncomplete,
			domains: map[string]types.StatusType{
				"testDomain1": types.StatusRunning,
			},
			expectedStatus:       types.StatusIncomplete,
			expectedDomainStatus: types.StatusRunning,
			testCode:             func() {},
		},
		"test_handleDomainCleanup_domain_status_not_from_supported": {
			handleStatus:     types.BaselineStatusCleanup,
			updateOrchStatus: types.StatusRunning,
			domains: map[string]types.StatusType{
				"testDomain1": types.StatusCompleted,
			},
			expectedStatus:       types.StatusRunning,
			expectedDomainStatus: types.StatusCompleted,
			testCode:             func() {},
		},
		"test_handleDomainCleanupSuccess_Success": {
			handleStatus:     types.BaselineStatusCleanupSuccess,
			updateOrchStatus: types.StatusRunning,
			domains: map[string]types.StatusType{
				"testDomain1": types.BaselineStatusActivationSuccess,
			},
			expectedStatus:       types.StatusCompleted,
			expectedDomainStatus: types.BaselineStatusCleanupSuccess,
			testCode:             func() {},
		},
		"test_handleDomainCleanupSuccess_orchestration_status_not_running": {
			handleStatus:     types.BaselineStatusCleanupSuccess,
			updateOrchStatus: types.StatusIncomplete,
			domains: map[string]types.StatusType{
				"testDomain1": types.StatusRunning,
			},
			expectedStatus:       types.StatusIncomplete,
			expectedDomainStatus: types.StatusRunning,
			testCode:             func() {},
		},
		"test_handleDomainCleanupSuccess_domain_status_not_from_supported": {
			handleStatus:     types.BaselineStatusCleanupSuccess,
			updateOrchStatus: types.StatusRunning,
			domains: map[string]types.StatusType{
				"testDomain1": types.StatusRunning,
			},
			expectedStatus:       types.StatusRunning,
			expectedDomainStatus: types.StatusRunning,
			testCode:             func() {},
		},
		"test_handleDomainCleanupSuccess_handleDomainCleanupSuccess_domain_status_not_from_supported": {
			handleStatus:     types.BaselineStatusCleanupSuccess,
			updateOrchStatus: types.StatusRunning,
			domains: map[string]types.StatusType{
				"testDomain1": types.BaselineStatusActivationSuccess,
				"testDomain2": types.StatusRunning,
			},
			expectedStatus:       types.StatusRunning,
			expectedDomainStatus: types.BaselineStatusCleanupSuccess,
			testCode:             func() {},
		},
		"test_handleDomainCleanupFailure_Success": {
			handleStatus:     types.BaselineStatusCleanupFailure,
			updateOrchStatus: types.StatusRunning,
			domains: map[string]types.StatusType{
				"testDomain1": types.BaselineStatusActivationSuccess,
			},
			expectedStatus:       types.StatusCompleted,
			expectedDomainStatus: types.BaselineStatusCleanupFailure,
			testCode:             func() {},
		},
		"test_handleDomainCleanupFailure_orchestration_status_not_running": {
			handleStatus:     types.BaselineStatusCleanupFailure,
			updateOrchStatus: types.StatusIncomplete,
			domains: map[string]types.StatusType{
				"testDomain1": types.StatusRunning,
			},
			expectedStatus:       types.StatusIncomplete,
			expectedDomainStatus: types.StatusRunning,
			testCode:             func() {},
		},
		"test_handleDomainCleanupFailure_domain_status_not_from_supported": {
			handleStatus:     types.BaselineStatusCleanupFailure,
			updateOrchStatus: types.StatusRunning,
			domains: map[string]types.StatusType{
				"testDomain1": types.StatusSuperseded,
			},
			expectedStatus:       types.StatusRunning,
			expectedDomainStatus: types.StatusSuperseded,
			testCode:             func() {},
		},
		"test_status_not_from_supported": {
			handleStatus:     "NOT_SUPPORTED_STATUS",
			updateOrchStatus: types.StatusRunning,
			domains: map[string]types.StatusType{
				"testDomain1": types.StatusSuperseded,
			},
			expectedStatus:       types.StatusRunning,
			expectedDomainStatus: types.StatusSuperseded,
			testCode:             func() {},
		},
	}
	for testName, testCase := range testCases {
		t.Run(testName, func(t *testing.T) {
			t.Log("TestName: ", testName)
			testUpdOrch := &updateOrchestrator{
				operation: &updateOperation{
					activityID:           test.ActivityID,
					domains:              testCase.domains,
					desiredStateCallback: eventCallback,
					actions:              testOperationActions,
					status:               testCase.updateOrchStatus,
					identDone:            make(chan bool, 1),
					identErrChan:         make(chan bool, 1),
					errChan:              make(chan bool, 1),
					done:                 make(chan bool, 1),
					statesPerDomain: map[api.UpdateManager]*types.DesiredState{
						mockUpdateManager: {},
					},
				},
				cfg: test.CreateTestConfig(false, false),
			}

			testCase.testCode()

			testUpdOrch.HandleDesiredStateFeedbackEvent("testDomain1", test.ActivityID, "", testCase.handleStatus, "testMsg", testUpdateActions)

			assert.Equal(t, testCase.expectedDomainStatus, testUpdOrch.operation.domains["testDomain1"])
			assert.Equal(t, testCase.expectedStatus, testUpdOrch.operation.status)
			assert.Equal(t, testCase.expectedDelayedStatus, testUpdOrch.operation.delayedStatus)
			assert.Equal(t, testCase.expectedIdentErrMsg, testUpdOrch.operation.identErrMsg)
		})
	}
}

func TestValidateActivity(t *testing.T) {
	t.Run("test_HandleDesiredStateFeedbackEvent_validateActivity_updateOrchestrator_operation_is_nil", func(t *testing.T) {
		testUpdOrch := &updateOrchestrator{}
		testUpdOrch.HandleDesiredStateFeedbackEvent("testDomain", test.ActivityID, "", types.BaselineStatusCleanupFailure, "testMsg", testUpdateActions)
	})
	t.Run("test_validateActivity_activityId_missmatch", func(t *testing.T) {
		testUpdOrch := &updateOrchestrator{
			operation: &updateOperation{
				activityID: test.ActivityID,
			},
		}
		assert.False(t, testUpdOrch.validateActivity("testDomain", "difActivityId"))
	})
	t.Run("test_validateActivity_domains_missmatch", func(t *testing.T) {
		testUpdOrch := &updateOrchestrator{
			operation: &updateOperation{
				activityID: test.ActivityID,
				domains: map[string]types.StatusType{
					"testDomain": types.StatusCompleted,
				},
			},
		}
		assert.False(t, testUpdOrch.validateActivity("difTestDomain", test.ActivityID))
	})
}

func TestDomainUpdateCompleted(t *testing.T) {
	t.Run("test_CfgNoRebootRequired_status_complete", func(t *testing.T) {
		orchestrator := generateUpdOrch(false, make(map[string]map[string]*types.Action), types.StatusCompleted, nil)

		orchestrator.domainUpdateCompleted()

		assert.Equal(t, 0, len(orchestrator.operation.errChan))
		assert.Equal(t, 1, len(orchestrator.operation.done))
		assert.Equal(t, 1, len(orchestrator.operation.identDone))

		assert.True(t, <-orchestrator.operation.done)
		assert.True(t, <-orchestrator.operation.identDone)

		assert.Equal(t, types.StatusCompleted, orchestrator.operation.status)
		assert.Equal(t, false, orchestrator.operation.rebootRequired)
	})
	t.Run("test_actions_moreThanZero_and_CfgRebootRequired", func(t *testing.T) {
		orchestrator := generateUpdOrch(true, map[string]map[string]*types.Action{
			"testDomain1": {
				"testAction": {},
			},
		}, types.StatusCompleted, nil)

		orchestrator.domainUpdateCompleted()

		assert.Equal(t, 0, len(orchestrator.operation.errChan))
		assert.Equal(t, 1, len(orchestrator.operation.done))
		assert.Equal(t, 1, len(orchestrator.operation.identDone))

		assert.True(t, <-orchestrator.operation.done)
		assert.True(t, <-orchestrator.operation.identDone)

		assert.Equal(t, types.StatusCompleted, orchestrator.operation.status)
		assert.Equal(t, true, orchestrator.operation.rebootRequired)
	})
	t.Run("test_delayedStatusStatusIncomplete", func(t *testing.T) {
		orchestrator := generateUpdOrch(false, map[string]map[string]*types.Action{
			"testDomain1": {
				"testAction": {},
			},
		}, types.StatusIncomplete, nil)

		orchestrator.domainUpdateCompleted()

		assert.Equal(t, 1, len(orchestrator.operation.errChan))
		assert.Equal(t, 0, len(orchestrator.operation.done))
		assert.Equal(t, 0, len(orchestrator.operation.identDone))

		assert.True(t, <-orchestrator.operation.errChan)

		assert.Equal(t, types.StatusIncomplete, orchestrator.operation.status)
		assert.Equal(t, false, orchestrator.operation.rebootRequired)
		assert.Equal(t, "the update process is incompleted", orchestrator.operation.errMsg)
	})
	t.Run("test_domains_moreThanZero_all_supported_and_CfgRebootRequired", func(t *testing.T) {
		orchestrator := generateUpdOrch(true, map[string]map[string]*types.Action{
			"testDomain1": {
				"testAction": {},
			},
		}, types.StatusCompleted, map[string]types.StatusType{
			"d1": types.BaselineStatusCleanupSuccess,
			"d2": types.BaselineStatusCleanupFailure,
		})

		orchestrator.domainUpdateCompleted()

		assert.Equal(t, 0, len(orchestrator.operation.errChan))
		assert.Equal(t, 1, len(orchestrator.operation.done))
		assert.Equal(t, 1, len(orchestrator.operation.identDone))

		assert.True(t, <-orchestrator.operation.done)
		assert.True(t, <-orchestrator.operation.identDone)

		assert.Equal(t, types.StatusCompleted, orchestrator.operation.status)
		assert.Equal(t, true, orchestrator.operation.rebootRequired)
	})
	t.Run("test_domains_moreThanZero_one_not_supported", func(t *testing.T) {
		orchestrator := generateUpdOrch(true, map[string]map[string]*types.Action{
			"testDomain1": {
				"testAction": {},
			},
		}, types.StatusCompleted, map[string]types.StatusType{
			"d1": types.BaselineStatusCleanupSuccess,
			"d2": types.BaselineStatusCleanup,
		})

		orchestrator.domainUpdateCompleted()

		assert.Equal(t, 0, len(orchestrator.operation.errChan))
		assert.Equal(t, 0, len(orchestrator.operation.done))
		assert.Equal(t, 0, len(orchestrator.operation.identDone))

		assert.Equal(t, types.StatusIdentifying, orchestrator.operation.status)
		assert.Equal(t, false, orchestrator.operation.rebootRequired)
	})
}

func generateUpdOrch(cfgRebootReqired bool, actions map[string]map[string]*types.Action, delayedStatus types.StatusType, domains map[string]types.StatusType) *updateOrchestrator {
	return &updateOrchestrator{
		operation: &updateOperation{
			actions:        actions,
			delayedStatus:  delayedStatus,
			status:         types.StatusIdentifying,
			errMsg:         "testErrMsg",
			errChan:        make(chan bool, 1),
			identDone:      make(chan bool, 1),
			done:           make(chan bool, 1),
			rebootRequired: false,
			domains:        domains,
		},
		cfg: test.CreateTestConfig(cfgRebootReqired, true),
	}
}

func generateCommand(cmdType types.CommandType) *types.DesiredStateCommand {
	return &types.DesiredStateCommand{
		Command: cmdType,
	}
}
