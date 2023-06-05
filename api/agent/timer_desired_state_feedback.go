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
	"fmt"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/eclipse-kanto/update-manager/api/types"
	"github.com/eclipse-kanto/update-manager/logger"
)

type desiredStateFeedbackNotifier struct {
	lock          sync.Mutex
	internalTimer *time.Timer
	interval      time.Duration

	agent *updateAgent

	activityID      string
	actions         []*types.Action
	reportedActions []*types.Action
}

func newDesiredStateFeedbackNotifier(interval time.Duration, agent *updateAgent) *desiredStateFeedbackNotifier {
	return &desiredStateFeedbackNotifier{
		interval:        interval,
		agent:           agent,
		actions:         []*types.Action{},
		reportedActions: []*types.Action{},
	}
}

func (t *desiredStateFeedbackNotifier) set(activityID string, actions []*types.Action) {
	t.lock.Lock()
	defer t.lock.Unlock()

	t.updateActions(actions)
	if t.internalTimer == nil {
		t.activityID = activityID
		t.updateReportedActions()
		t.agent.publishDesiredStateFeedback(activityID, "", types.StatusRunning, "", actions)
		t.internalTimer = time.AfterFunc(t.interval, t.notifyEvent)
	}
}

func (t *desiredStateFeedbackNotifier) stop() {
	t.lock.Lock()
	defer t.lock.Unlock()

	t.activityID = ""
	t.actions = []*types.Action{}
	t.reportedActions = []*types.Action{}
	if t.internalTimer != nil {
		t.internalTimer.Stop()
		t.internalTimer = nil
	}
}

func (t *desiredStateFeedbackNotifier) notifyEvent() {
	t.lock.Lock()
	defer t.lock.Unlock()

	if t.internalTimer == nil {
		return
	}
	t.internalTimer = nil
	if reflect.DeepEqual(t.actions, t.reportedActions) {
		return
	}
	t.updateReportedActions()
	t.agent.publishDesiredStateFeedback(t.activityID, "", types.StatusRunning, "", t.actions)
}

func (t *desiredStateFeedbackNotifier) updateActions(actions []*types.Action) {
	if logger.IsTraceEnabled() {
		traceMsg := fmt.Sprintf("the current actions are: %v", toActionsString(t.actions))
		logger.Trace(traceMsg)
	}
	t.actions = actions
	if logger.IsTraceEnabled() {
		traceMsg := fmt.Sprintf("the updated actions are: %s", toActionsString(t.actions))
		logger.Trace(traceMsg)
	}
}

func (t *desiredStateFeedbackNotifier) updateReportedActions() {
	t.reportedActions = t.actions
	if logger.IsTraceEnabled() {
		traceMsg := fmt.Sprintf("the reported actions for status '%s' are: %s", types.StatusRunning, toActionsString(t.reportedActions))
		logger.Trace(traceMsg)
	}
}

func toActionsString(actions []*types.Action) string {
	var sb strings.Builder
	sb.WriteString("[ ")
	for _, action := range actions {
		actionString := fmt.Sprintf("{Component:{ID:%s Version:%s} Status:%s Progress:%v Message:%s} ", action.Component.ID, action.Component.Version, action.Status, action.Progress, action.Message)
		sb.WriteString(actionString)
	}
	sb.WriteString("]")
	return sb.String()
}
