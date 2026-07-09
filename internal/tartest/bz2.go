// Copyright (C) 2026 archives contributors
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.
//
// SPDX-License-Identifier: MPL-2.0

package tartest

import (
	"io"
	"os/exec"
)

// newBzip2Writer creates a new bzip2 writer that writes to the provided
// writer using the bzip2 command.
func newBzip2Writer(dest io.Writer) (io.WriteCloser, error) {
	cmd := exec.Command("bzip2")
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	cmd.Stdout = dest
	return &cmdCloser{cmd, stdin}, cmd.Start()
}

// cmdCloser is a helper type that wraps an exec.Cmd and an
// [io.WriteCloser], closing the WriteCloser when Close is called.
type cmdCloser struct {
	cmd *exec.Cmd
	io.WriteCloser
}

func (c *cmdCloser) Close() error {
	// close the stdin pipe to signal the command to finish
	if err := c.WriteCloser.Close(); err != nil {
		return err
	}

	return c.cmd.Wait()
}
