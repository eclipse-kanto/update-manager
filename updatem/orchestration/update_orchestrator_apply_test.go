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
				identDone:            make(chan bool, 1),
				done:                 make(chan bool, 1),
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

		orchestrator.operation.identDone <- true
		orchestrator.operation.done <- true

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
				identDone:            make(chan bool, 1),
				identErrChan:         make(chan bool, 1),
				errChan:              make(chan bool, 1),
				done:                 make(chan bool, 1),
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
		orchestrator.operation.done <- true
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

func TestWaitIdentification(t *testing.T) {
	const (
		identDone    = "1"
		errChan      = "2"
		identErrChan = "3"
		none         = "4"
	)
	testCases := map[string]struct {
		ctx              context.Context
		testChan         string
		terminateContext bool
		expectedErr      error
		expectedStatus   types.StatusType
	}{
		"test_case_identErrChan": {
			ctx:            context.Background(),
			testChan:       identErrChan,
			expectedErr:    fmt.Errorf("testIdentErrMsg"),
			expectedStatus: types.StatusIdentifying,
		},
		"test_case_errChan": {
			ctx:            context.Background(),
			testChan:       errChan,
			expectedErr:    fmt.Errorf("testErrMsg"),
			expectedStatus: types.StatusIdentifying,
		},
		"test_case_identDone": {
			ctx:            context.Background(),
			testChan:       identDone,
			expectedStatus: types.StatusIdentifying,
		},
		"test_case_terminateContext": {
			ctx:              context.Background(),
			testChan:         none,
			expectedErr:      fmt.Errorf("the update manager instance is terminated"),
			expectedStatus:   types.StatusIncomplete,
			terminateContext: true,
		},
	}

	for testName, testCase := range testCases {
		t.Run(testName, func(t *testing.T) {
			orchestrator := &updateOrchestrator{
				operation: &updateOperation{
					identErrChan: make(chan bool, 1),
					errChan:      make(chan bool, 1),
					identDone:    make(chan bool, 1),
					identErrMsg:  "testIdentErrMsg",
					errMsg:       "testErrMsg",
					status:       types.StatusIdentifying,
				},
			}

			if testCase.testChan == identDone {
				orchestrator.operation.identDone <- true
			} else if testCase.testChan == errChan {
				orchestrator.operation.errChan <- true
			} else if testCase.testChan == identErrChan {
				orchestrator.operation.identErrChan <- true
			}

			var actualErr error
			if testCase.terminateContext {
				newContext, cancel := context.WithTimeout(testCase.ctx, time.Second)
				cancel()
				actualErr = orchestrator.waitIdentification(newContext)
			} else {
				actualErr = orchestrator.waitIdentification(testCase.ctx)
			}

			assert.Equal(t, testCase.expectedErr, actualErr)
			assert.Equal(t, testCase.expectedStatus, orchestrator.operation.status)
		})
	}
}

func TestWaitCompletion(t *testing.T) {
	const (
		errChan = "1"
		done    = "2"
		none    = "4"
	)
	testCases := map[string]struct {
		ctx              context.Context
		testChan         string
		terminateContext bool
		expectedErr      error
		expectedStatus   types.StatusType
	}{
		"test_case_errChan": {
			ctx:            context.Background(),
			testChan:       errChan,
			expectedErr:    fmt.Errorf("testErrMsg"),
			expectedStatus: types.StatusIdentifying,
		},
		"test_case_done": {
			ctx:            context.Background(),
			testChan:       done,
			expectedStatus: types.StatusIdentifying,
		},
		"test_case_terminateContext": {
			ctx:              context.Background(),
			testChan:         none,
			expectedErr:      fmt.Errorf("the update manager instance is terminated"),
			expectedStatus:   types.StatusIncomplete,
			terminateContext: true,
		},
	}

	for testName, testCase := range testCases {
		t.Run(testName, func(t *testing.T) {
			orchestrator := &updateOrchestrator{
				operation: &updateOperation{
					errChan: make(chan bool, 1),
					done:    make(chan bool, 1),
					errMsg:  "testErrMsg",
					status:  types.StatusIdentifying,
				},
			}

			if testCase.testChan == done {
				orchestrator.operation.done <- true
			} else if testCase.testChan == errChan {
				orchestrator.operation.errChan <- true
			}

			var actualErr error
			if testCase.terminateContext {
				newContext, cancel := context.WithTimeout(testCase.ctx, time.Second)
				cancel()
				actualErr = orchestrator.waitCompletion(newContext)
			} else {
				actualErr = orchestrator.waitCompletion(testCase.ctx)
			}

			assert.Equal(t, testCase.expectedErr, actualErr)
			assert.Equal(t, testCase.expectedStatus, orchestrator.operation.status)
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

		assert.NotNil(t, orchestrator.operation.done)
		assert.NotNil(t, orchestrator.operation.errChan)
		assert.NotNil(t, orchestrator.operation.identDone)
		assert.NotNil(t, orchestrator.operation.identErrChan)

		orchestrator.operation.done = nil
		orchestrator.operation.errChan = nil
		orchestrator.operation.identDone = nil
		orchestrator.operation.identErrChan = nil

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
