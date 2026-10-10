// Copyright (C) 2026 archives contributors
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.
//
// SPDX-License-Identifier: MPL-2.0

//go:build linux || darwin

package archives

import (
	"fmt"
	"os"
)

// withFd calls fn with the file descriptor of f.
func withFd(f *os.File, fn func(fd int) error) error {
	rc, err := f.SyscallConn()
	if err != nil {
		return err
	}

	var fnErr error
	if err := rc.Control(func(fd uintptr) {
		fnErr = fn(int(fd)) //nolint:gosec // Why: File descriptors fit in an int.
	}); err != nil {
		return err
	}

	return fnErr
}

// withRootFd calls fn with a file descriptor for the directory of root.
func withRootFd(root *os.Root, fn func(fd int) error) error {
	d, err := root.Open(".")
	if err != nil {
		return fmt.Errorf("failed to open destination: %w", err)
	}
	defer d.Close()

	return withFd(d, fn)
}
