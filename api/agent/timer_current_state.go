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
	"sync"
	"time"
)

type currentStateNotifier struct {
	lock          sync.Mutex
	internalTimer *time.Timer
	interval      time.Duration

	agent *updateAgent

	currentStateBytes []byte
}

func newCurrentStateNotifier(interval time.Duration, agent *updateAgent) *currentStateNotifier {
	return &currentStateNotifier{
		interval: interval,
		agent:    agent,
	}
}

func (t *currentStateNotifier) set(currentStateBytes []byte) {
	t.lock.Lock()
	defer t.lock.Unlock()

	t.currentStateBytes = currentStateBytes
	if t.internalTimer == nil {
		t.internalTimer = time.AfterFunc(t.interval, t.notifyEvent)
	}
}

func (t *currentStateNotifier) stop() {
	t.lock.Lock()
	defer t.lock.Unlock()

	t.currentStateBytes = nil
	if t.internalTimer != nil {
		t.internalTimer.Stop()
		t.internalTimer = nil
	}
}

func (t *currentStateNotifier) notifyEvent() {
	t.lock.Lock()
	defer t.lock.Unlock()

	if t.internalTimer == nil {
		return
	}
	t.internalTimer = nil
	t.agent.publishCurrentState(t.currentStateBytes)
}
