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

//go:build windows
// +build windows

package orchestration

import (
	"os/exec"
	"time"

	"github.com/eclipse-kanto/update-manager/logger"
)

// Reboot makes the host device to reboot.
// Implementation for Windows just executes the shutdown command using the os/exec package.
func (rebootManager *rebootManager) Reboot(timeout time.Duration) error {
	logger.Debug("the system is about to reboot after successful update operation in '%s'", timeout)
	<-time.After(timeout)

	return exec.Command("cmd", "/C", "shutdown", "/s").Run()
}
