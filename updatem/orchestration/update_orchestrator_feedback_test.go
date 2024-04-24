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
	"fmt"
	"testing"
	"time"

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

type testWaitChannel func(map[types.CommandType]chan bool) error

func TestFeedbackHandleDesiredStateFeedbackEvent(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	eventCallback := mocks.NewMockUpdateManagerCallback(mockCtrl)
	mockUpdateManager := mocks.NewMockUpdateManager(mockCtrl)

	waitCommandChannel := func(command types.CommandType) testWaitChannel {
		return func(commands map[types.CommandType]chan bool) error {
			timeout := time.Second
			select {
			case <-commands[command]:
				return nil
			case <-time.After(timeout):
				return fmt.Errorf("%s command signal not received in %v", command, timeout)
			}
		}
	}

	testCases := map[string]struct {
		emptyIdentActions     bool
		handleStatus          types.StatusType
		updateOrchStatus      types.StatusType
		domains               map[string]types.StatusType
		expectedStatus        types.StatusType
		expectedDelayedStatus types.StatusType
		expectedDomainStatus  types.StatusType
		expectedErrMsg        string
		testCode              func()
		waitChannel           testWaitChannel
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
			},
			waitChannel: waitCommandChannel(types.CommandDownload),
		},

		"test_handleDomainIdentified_domainUpdateStatus_StatusIdentificationFailed": {
			handleStatus:     types.StatusIdentified,
			updateOrchStatus: types.StatusIdentified,
			domains: map[string]types.StatusType{
				"testDomain1": types.StatusIdentifying,
				"testDomain2": types.StatusIdentificationFailed,
			},
			expectedStatus:       types.StatusIdentificationFailed,
			expectedErrMsg:       "[testDomain1]: ",
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
			testCode:             func() {},
		},

		"test_handleDomainIdentified_checkIdentificationStatus": {
			handleStatus:     types.StatusIdentified,
			updateOrchStatus: types.StatusIdentified,
			domains: map[string]types.StatusType{
				"testDomain1": types.StatusIdentified,
			},
			expectedStatus:       types.StatusIdentified,
			expectedDomainStatus: types.StatusIdentified,
			testCode:             func() {},
		},

		"test_handleDomainIdentified_empty_actions": {
			emptyIdentActions: true,
			handleStatus:      types.StatusIdentified,
			updateOrchStatus:  types.StatusIdentifying,
			domains: map[string]types.StatusType{
				"testDomain1": types.StatusIdentifying,
			},
			expectedStatus:       types.StatusCompleted,
			expectedDomainStatus: types.BaselineStatusCleanupSuccess,
			testCode: func() {
				mockUpdateManager.EXPECT().Name().Return("testDomain1").AnyTimes()
				eventCallback.EXPECT().HandleDesiredStateFeedbackEvent("device", test.ActivityID, "", types.StatusIdentified, "", gomock.Any())
			},
		},

		"test_handleDomainIdentificationFailed_Success": {
			handleStatus:     types.StatusIdentificationFailed,
			updateOrchStatus: types.StatusIdentificationFailed,
			domains: map[string]types.StatusType{
				"testDomain1": types.StatusIdentifying,
			},
			expectedStatus:       types.StatusIdentificationFailed,
			expectedDomainStatus: types.StatusIdentificationFailed,
			expectedErrMsg:       "[testDomain1]: testMsg",
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
			expectedStatus:       types.StatusCompleted,
			expectedDomainStatus: types.BaselineStatusCleanupSuccess,
			testCode:             func() {},
		},
		"test_handleDomainCompletedEvent_orchestration_status_identified": {
			handleStatus:     types.StatusCompleted,
			updateOrchStatus: types.StatusIdentified,
			domains: map[string]types.StatusType{
				"testDomain1": types.StatusIdentified,
			},
			expectedStatus:       types.StatusCompleted,
			expectedDomainStatus: types.BaselineStatusCleanupSuccess,
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
			expectedStatus:        types.StatusIncomplete,
			expectedDelayedStatus: types.StatusIncomplete,
			expectedDomainStatus:  types.BaselineStatusCleanupFailure,
			expectedErrMsg:        "the update process is incompleted",
			testCode:              func() {},
		},
		"test_handleDomainIncomplete_orchestration_status_identified": {
			handleStatus:     types.StatusIncomplete,
			updateOrchStatus: types.StatusIdentified,
			domains: map[string]types.StatusType{
				"testDomain1": types.StatusIdentified,
			},
			expectedStatus:        types.StatusIncomplete,
			expectedDomainStatus:  types.BaselineStatusCleanupFailure,
			expectedDelayedStatus: types.StatusIncomplete,
			expectedErrMsg:        "the update process is incompleted",
			testCode:              func() {},
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
				eventCallback.EXPECT().HandleDesiredStateFeedbackEvent("device", test.ActivityID, "", types.StatusRunning, "", gomock.Any())
			},
			waitChannel: waitCommandChannel(types.CommandUpdate),
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
				mockUpdateManager.EXPECT().Name().Return("testDomain1").AnyTimes()
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
				eventCallback.EXPECT().HandleDesiredStateFeedbackEvent("device", test.ActivityID, "", types.StatusRunning, "", gomock.Any())
			},
			waitChannel: waitCommandChannel(types.CommandActivate),
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
				mockUpdateManager.EXPECT().Name().Return("testDomain1").AnyTimes()
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
				eventCallback.EXPECT().HandleDesiredStateFeedbackEvent("device", test.ActivityID, "", types.StatusRunning, "", gomock.Any())
			},
			waitChannel: waitCommandChannel(types.CommandCleanup),
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
				mockUpdateManager.EXPECT().Name().Return("testDomain1").AnyTimes()
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
			updateActions := []*types.Action{}
			if !testCase.emptyIdentActions {
				updateActions = testUpdateActions
			}
			testUpdOrch := &updateOrchestrator{
				operation: &updateOperation{
					activityID:           test.ActivityID,
					domains:              testCase.domains,
					desiredStateCallback: eventCallback,
					actions:              testOperationActions,
					status:               testCase.updateOrchStatus,
					commandChannels:      generateCommandChannels(),
					done:                 make(chan bool, 1),
					errChan:              make(chan bool, 1),
					statesPerDomain: map[api.UpdateManager]*types.DesiredState{
						mockUpdateManager: {},
					},
				},
				cfg: createTestConfig(false, false),
			}

			testCase.testCode()
			errorChan := make(chan error, 1)
			go func() {
				if testCase.waitChannel != nil {
					errorChan <- testCase.waitChannel(testUpdOrch.operation.commandChannels)
				}
				errorChan <- nil
			}()

			testUpdOrch.HandleDesiredStateFeedbackEvent("testDomain1", test.ActivityID, "", testCase.handleStatus, "testMsg", updateActions)

			assert.Equal(t, testCase.expectedDomainStatus, testUpdOrch.operation.domains["testDomain1"])
			assert.Equal(t, testCase.expectedStatus, testUpdOrch.operation.status)
			assert.Equal(t, testCase.expectedDelayedStatus, testUpdOrch.operation.delayedStatus)
			assert.Equal(t, testCase.expectedErrMsg, testUpdOrch.operation.errMsg)

			assert.Nil(t, <-errorChan)
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
	testActions := map[string]map[string]*types.Action{
		"testDomain1": {
			"testAction": {},
		},
	}
	testCases := map[string]struct {
		cfgRebootReqired       bool
		actions                map[string]map[string]*types.Action
		delayedStatus          types.StatusType
		domain                 map[string]types.StatusType
		errChanLen             bool
		doneChanLen            bool
		expectedRebootRequired bool
		expectedStatus         types.StatusType
		expectedErrMsg         string
	}{
		"test_CfgNoRebootRequired_status_complete": {
			actions:        make(map[string]map[string]*types.Action),
			delayedStatus:  types.StatusCompleted,
			doneChanLen:    true,
			expectedStatus: types.StatusCompleted,
		},
		"test_actions_moreThanZero_and_CfgRebootRequired": {
			cfgRebootReqired:       true,
			actions:                testActions,
			delayedStatus:          types.StatusCompleted,
			doneChanLen:            true,
			expectedRebootRequired: true,
			expectedStatus:         types.StatusCompleted,
		},
		"test_delayedStatusStatusIncomplete": {
			actions:        testActions,
			delayedStatus:  types.StatusIncomplete,
			errChanLen:     true,
			expectedStatus: types.StatusIncomplete,
			expectedErrMsg: "the update process is incompleted",
		},
		"test_domains_moreThanZero_all_supported_and_CfgRebootRequired": {
			cfgRebootReqired: true,
			actions:          testActions,
			delayedStatus:    types.StatusCompleted,
			domain: map[string]types.StatusType{
				"d1": types.BaselineStatusCleanupSuccess,
				"d2": types.BaselineStatusCleanupFailure,
			},
			doneChanLen:            true,
			expectedStatus:         types.StatusCompleted,
			expectedRebootRequired: true,
		},
		"test_domains_moreThanZero_one_not_supported": {
			cfgRebootReqired: true,
			actions:          testActions,
			delayedStatus:    types.StatusCompleted,
			domain: map[string]types.StatusType{
				"d1": types.BaselineStatusCleanupSuccess,
				"d2": types.BaselineStatusCleanup,
			},
			expectedStatus: types.StatusIdentifying,
		},
	}

	for testName, testCase := range testCases {
		t.Run(testName, func(t *testing.T) {
			orchestrator := generateUpdOrch(testCase.cfgRebootReqired, testCase.actions, testCase.delayedStatus, testCase.domain)

			orchestrator.domainUpdateCompleted()

			if testCase.errChanLen {
				assert.Equal(t, 1, len(orchestrator.operation.errChan))
				assert.True(t, <-orchestrator.operation.errChan)
			} else {
				assert.Equal(t, 0, len(orchestrator.operation.errChan))
			}
			if testCase.doneChanLen {
				assert.Equal(t, 1, len(orchestrator.operation.done))
				assert.True(t, <-orchestrator.operation.done)
			} else {
				assert.Equal(t, 0, len(orchestrator.operation.done))
			}

			assert.Equal(t, testCase.expectedStatus, orchestrator.operation.status)
			assert.Equal(t, testCase.expectedRebootRequired, orchestrator.operation.rebootRequired)
			assert.Equal(t, testCase.expectedErrMsg, orchestrator.operation.errMsg)
		})
	}
}

func generateUpdOrch(cfgRebootReqired bool, actions map[string]map[string]*types.Action, delayedStatus types.StatusType, domains map[string]types.StatusType) *updateOrchestrator {
	return &updateOrchestrator{
		operation: &updateOperation{
			actions:         actions,
			delayedStatus:   delayedStatus,
			status:          types.StatusIdentifying,
			errMsg:          "",
			errChan:         make(chan bool, 1),
			done:            make(chan bool, 1),
			commandChannels: generateCommandChannels(),
			domains:         domains,
		},
		cfg:          createTestConfig(cfgRebootReqired, true),
		phaseTimeout: 10 * time.Minute,
	}
}

func generateCommand(cmdType types.CommandType) *types.DesiredStateCommand {
	return &types.DesiredStateCommand{
		Command: cmdType,
	}
}
