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
	"sync"
	"testing"
	"time"

	"github.com/eclipse-kanto/update-manager/api"
	"github.com/eclipse-kanto/update-manager/api/types"
	"github.com/eclipse-kanto/update-manager/test"
	"github.com/eclipse-kanto/update-manager/test/mocks"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestApply(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	doneChan := make(chan bool, 1)
	applyChan := make(chan bool, 1)
	successReturnChan := make(chan bool, 1)
	errReturnChan := make(chan error, 1)

	t.Run("test_successful_operation", func(t *testing.T) {
		ctx := context.Background()
		eventCallback := mocks.NewMockUpdateManagerCallback(mockCtrl)
		mockUpdateManager := mocks.NewMockUpdateManager(mockCtrl)

		orchestrator := &updateOrchestrator{
			cfg:          createTestConfig(false, false),
			phaseTimeout: test.Interval,
			operation: &updateOperation{
				desiredState:         &types.DesiredState{},
				desiredStateCallback: eventCallback,
				phaseChannels:        generatePhaseChannels(),
				activityID:           test.ActivityID,
				actions: map[string]map[string]*types.Action{
					"action1": {
						"action2": {
							Message: "testMsg",
						},
					},
				},
			},
		}

		statePerDomain := &types.DesiredState{}
		orchestrator.operation.statesPerDomain = map[api.UpdateManager]*types.DesiredState{
			mockUpdateManager: statePerDomain,
		}

		orchestrator.operation.phaseChannels[phaseIdentification] <- true
		orchestrator.operation.phaseChannels[phaseDownload] <- false
		expectedActions := []*types.Action{
			{
				Message: "testMsg",
			},
		}

		mockUpdateManager.EXPECT().Apply(ctx, test.ActivityID, statePerDomain).DoAndReturn(func(ctx context.Context, activityID string, state *types.DesiredState) {
			applyChan <- true
		})
		eventCallback.EXPECT().HandleDesiredStateFeedbackEvent("device", test.ActivityID, "", types.StatusIdentifying, "", expectedActions).Times(1)

		go applyCall(ctx, orchestrator, doneChan, successReturnChan, errReturnChan)

		<-applyChan
		assert.Nil(t, <-errReturnChan)
		assert.False(t, orchestrator.operation.rebootRequired)
		assert.False(t, <-successReturnChan)
		<-doneChan
	})

	t.Run("test_apply_error", func(t *testing.T) {
		ctx := context.Background()
		eventCallback := mocks.NewMockUpdateManagerCallback(mockCtrl)
		mockUpdateManager := mocks.NewMockUpdateManager(mockCtrl)

		orchestrator := &updateOrchestrator{
			cfg:          createTestConfig(false, false),
			phaseTimeout: test.Interval,
			operation: &updateOperation{
				desiredState:         &types.DesiredState{},
				desiredStateCallback: eventCallback,
				errChan:              make(chan bool, 1),
				errMsg:               "testErrMsg",
				activityID:           test.ActivityID,
				actions: map[string]map[string]*types.Action{
					"action1": {
						"action2": {
							Message: "testMsg",
						},
					},
				},
			},
		}

		statePerDomain := &types.DesiredState{}
		orchestrator.operation.statesPerDomain = map[api.UpdateManager]*types.DesiredState{
			mockUpdateManager: statePerDomain,
		}
		orchestrator.operation.errChan <- true
		expectedActions := []*types.Action{
			{
				Message: "testMsg",
			},
		}

		mockUpdateManager.EXPECT().Apply(ctx, test.ActivityID, statePerDomain).DoAndReturn(func(ctx context.Context, activityID string, state *types.DesiredState) {
			applyChan <- true
		})

		eventCallback.EXPECT().HandleDesiredStateFeedbackEvent("device", test.ActivityID, "", types.StatusIdentifying, "", expectedActions).Times(1)

		go applyCall(ctx, orchestrator, doneChan, successReturnChan, errReturnChan)

		<-applyChan
		assert.Equal(t, fmt.Errorf("testErrMsg"), <-errReturnChan)
		assert.False(t, orchestrator.operation.rebootRequired)
		assert.False(t, <-successReturnChan)
		<-doneChan
	})
}

func applyCall(ctx context.Context, orchestrator *updateOrchestrator, done chan bool, successCh chan bool, errCh chan error) {
	success, err := orchestrator.apply(ctx)
	successCh <- success
	errCh <- err
	done <- true
}

func TestWaitPhase(t *testing.T) {
	const (
		phaseDone = "1"
		errChan   = "2"
		none      = "4"
	)
	testCases := map[string]struct {
		ctx              context.Context
		testChan         string
		phase            phase
		phaseDone        bool
		terminateContext bool
		expectedWait     bool
		expectedErr      error
		expectedStatus   types.StatusType
	}{
		"test_case_errChan": {
			ctx:            context.Background(),
			testChan:       errChan,
			expectedErr:    fmt.Errorf("testErrMsg"),
			expectedStatus: types.StatusIdentifying,
		},
		"test_case_phaseDone_identification": {
			ctx:            context.Background(),
			testChan:       phaseDone,
			phase:          phaseIdentification,
			expectedWait:   true,
			expectedStatus: types.StatusIdentifying,
		},
		"test_case_phaseDone_cleanup": {
			ctx:            context.Background(),
			testChan:       phaseDone,
			phase:          phaseCleanup,
			expectedStatus: types.StatusIdentifying,
		},
		"test_case_terminateContext": {
			ctx:              context.Background(),
			testChan:         none,
			expectedErr:      fmt.Errorf("the update manager instance is terminated"),
			expectedStatus:   types.StatusIncomplete,
			terminateContext: true,
		},
		"test_case_timeout_identification": {
			ctx:            context.Background(),
			testChan:       none,
			phase:          phaseIdentification,
			expectedErr:    fmt.Errorf("identification phase not done in 1s"),
			expectedStatus: types.StatusIdentificationFailed,
		},
		"test_case_timeout_download": {
			ctx:            context.Background(),
			testChan:       none,
			phase:          phaseDownload,
			expectedErr:    fmt.Errorf("download phase not done in 1s"),
			expectedStatus: types.StatusIncomplete,
		},
	}

	for testName, testCase := range testCases {
		t.Run(testName, func(t *testing.T) {
			orchestrator := &updateOrchestrator{
				operation: &updateOperation{
					errChan:       make(chan bool, 1),
					phaseChannels: generatePhaseChannels(),
					errMsg:        "testErrMsg",
					status:        types.StatusIdentifying,
				},
				phaseTimeout: time.Second,
			}

			wg := sync.WaitGroup{}
			var phaseHandler phaseHandler

			if testCase.testChan == errChan {
				orchestrator.operation.errChan <- true
			} else if testCase.testChan == phaseDone {
				if testCase.phase == phaseIdentification {
					wg.Add(1)
					phaseHandler = func(ctx context.Context, phase phase, orchestrator *updateOrchestrator) {
						wg.Done()
					}
				}
				orchestrator.operation.phaseChannels[testCase.phase] <- testCase.expectedWait
			}

			var actualErr error
			var actualWait bool
			if testCase.terminateContext {
				newContext, cancel := context.WithTimeout(testCase.ctx, time.Second)
				cancel()
				actualWait, actualErr = orchestrator.waitPhase(newContext, testCase.phase, phaseHandler)
			} else {
				actualWait, actualErr = orchestrator.waitPhase(testCase.ctx, testCase.phase, phaseHandler)
			}

			assert.Equal(t, testCase.expectedErr, actualErr)
			assert.Equal(t, testCase.expectedWait, actualWait)
			assert.Equal(t, testCase.expectedStatus, orchestrator.operation.status)

			wg.Wait()
		})
	}
}

func TestHandlePhaseCompletion(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockUpdateManager := mocks.NewMockUpdateManager(mockCtrl)

	testDomain1 := "testName1"
	testDomain2 := "testName2"
	operation := &updateOperation{
		activityID: test.ActivityID,
		domains: map[string]types.StatusType{
			testDomain1: types.StatusIdentifying,
			testDomain2: types.StatusIdentifying,
		},
	}
	operation.statesPerDomain = map[api.UpdateManager]*types.DesiredState{
		mockUpdateManager: {},
	}
	orchestrator := &updateOrchestrator{}

	mockCommand := func(mockUpdateManager *mocks.MockUpdateManager, command types.CommandType, domains ...string) func() {
		return func() {
			for _, domain := range domains {
				mockUpdateManager.EXPECT().Name().Return(domain).Times(1)
				mockUpdateManager.EXPECT().Command(context.Background(), test.ActivityID, generateCommand(command))
			}
		}
	}

	testCases := map[string]struct {
		noOperation   bool
		domainStatus1 types.StatusType
		domainStatus2 types.StatusType
		phase         phase
		expectedCalls func()
	}{
		"test_handle_phase_completion_identify": {
			domainStatus1: types.StatusIdentified,
			domainStatus2: types.StatusIdentified,
			phase:         phaseIdentification,
			expectedCalls: mockCommand(mockUpdateManager, types.CommandDownload, testDomain1, testDomain2),
		},
		"test_handle_phase_completion_download": {
			domainStatus1: types.BaselineStatusDownloadSuccess,
			domainStatus2: types.BaselineStatusDownloadFailure,
			phase:         phaseDownload,
			expectedCalls: mockCommand(mockUpdateManager, types.CommandUpdate, testDomain1),
		},
		"test_handle_phase_completion_update": {
			domainStatus1: types.BaselineStatusUpdateSuccess,
			domainStatus2: types.BaselineStatusUpdateFailure,
			phase:         phaseUpdate,
			expectedCalls: mockCommand(mockUpdateManager, types.CommandActivate, testDomain1),
		},
		"test_handle_phase_completion_activate": {
			domainStatus1: types.BaselineStatusActivationSuccess,
			domainStatus2: types.BaselineStatusActivationFailure,
			phase:         phaseActivation,
			expectedCalls: mockCommand(mockUpdateManager, types.CommandCleanup, testDomain1),
		},
		"test_handle_phase_completion_cleanup": {
			domainStatus1: types.BaselineStatusCleanupSuccess,
			domainStatus2: types.BaselineStatusCleanupFailure,
			phase:         phaseCleanup,
			expectedCalls: func() {},
		},
		"test_handle_phase_completion_update_failure": {
			domainStatus1: types.BaselineStatusUpdateFailure,
			domainStatus2: types.BaselineStatusUpdateFailure,
			phase:         phaseUpdate,
			expectedCalls: mockCommand(mockUpdateManager, types.CommandActivate),
		},
		"test_handle_phase_completion_no_operation": {
			noOperation: true,
		},
		"test_handle_phase_completion_unknown_phase": {
			domainStatus1: types.BaselineStatusCleanupSuccess,
			domainStatus2: types.BaselineStatusCleanupFailure,
			phase:         phase("unknown"),
			expectedCalls: func() {},
		},
	}
	for testName, testCase := range testCases {
		t.Run(testName, func(t *testing.T) {
			if testCase.noOperation {
				orchestrator.operation = nil
			} else {
				orchestrator.operation = operation
				orchestrator.operation.domains[testDomain1] = testCase.domainStatus1
				orchestrator.operation.domains[testDomain2] = testCase.domainStatus2
				testCase.expectedCalls()
			}
			handlePhaseCompletion(context.Background(), testCase.phase, orchestrator)
		})
	}
}

func TestCommand(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockUpdateManager := mocks.NewMockUpdateManager(mockCtrl)
	orchestrator := &updateOrchestrator{
		operation: &updateOperation{statesPerDomain: map[api.UpdateManager]*types.DesiredState{
			mockUpdateManager: {},
		}},
	}
	t.Run("test_command_existing_domain", func(t *testing.T) {
		command := &types.DesiredStateCommand{
			Command: types.CommandActivate,
		}

		mockUpdateManager.EXPECT().Name().Return("testName").Times(1)
		mockUpdateManager.EXPECT().Command(context.Background(), test.ActivityID, command)

		orchestrator.command(context.Background(), test.ActivityID, "testName", types.CommandActivate)
	})
	t.Run("test_command_domain_not_exists", func(t *testing.T) {
		mockUpdateManager.EXPECT().Name().Return("testName").Times(1)

		orchestrator.command(context.Background(), test.ActivityID, "difTestName", types.CommandActivate)
	})
}

func TestSetupUpdateOperation(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	t.Run("test_setupupdateoperation_newUpdateOperation_err_nil", func(t *testing.T) {
		domainAgents := map[string]api.UpdateManager{
			"domain1": mocks.NewMockUpdateManager(mockCtrl),
		}
		handler := &TestDesiredStateFeedbackHandler{}

		expectedOp := &updateOperation{
			activityID: test.ActivityID,
			status:     types.StatusIdentifying,
			domains: map[string]types.StatusType{
				"domain1": types.StatusIdentifying,
			},
			desiredState: test.DesiredState,
			statesPerDomain: map[api.UpdateManager]*types.DesiredState{
				domainAgents["domain1"]: test.DesiredState,
			},
			actions:              map[string]map[string]*types.Action{},
			desiredStateCallback: handler,
		}

		orchestrator := &updateOrchestrator{
			operation: &updateOperation{},
		}

		err := orchestrator.setupUpdateOperation(domainAgents, test.ActivityID, test.DesiredState, handler)

		assert.NotNil(t, orchestrator.operation.phaseChannels)
		assert.NotNil(t, orchestrator.operation.errChan)

		orchestrator.operation.errChan = nil
		orchestrator.operation.phaseChannels = nil

		assert.Equal(t, expectedOp, orchestrator.operation)
		assert.Nil(t, err)
	})
	t.Run("test_setupupdateoperation_newUpdateOperation_err_not_nil", func(t *testing.T) {
		domainAgents := map[string]api.UpdateManager{
			"domain1": mocks.NewMockUpdateManager(mockCtrl),
		}

		handler := &TestDesiredStateFeedbackHandler{}

		orchestrator := &updateOrchestrator{
			operation: &updateOperation{},
		}
		expectedOp := &updateOperation{
			status:               types.StatusIncomplete,
			desiredStateCallback: handler,
		}

		err := orchestrator.setupUpdateOperation(domainAgents, test.ActivityID, &types.DesiredState{}, handler)

		assert.Equal(t, expectedOp, orchestrator.operation)
		assert.Equal(t, "the desired state manifest does not contain any supported domain", err.Error())
	})
}

func TestDisposeUpdateOperation(t *testing.T) {
	t.Run("test_opeartion_disposed", func(t *testing.T) {
		orchestrator := &updateOrchestrator{
			operationLock: sync.Mutex{},
			operation:     &updateOperation{},
		}

		orchestrator.disposeUpdateOperation()
		assert.Nil(t, orchestrator.operation)
	})
	t.Run("test_no_operation_to_dispose", func(t *testing.T) {
		orchestrator := &updateOrchestrator{
			operationLock: sync.Mutex{},
		}

		orchestrator.disposeUpdateOperation()
		assert.Nil(t, orchestrator.operation)
	})
}
