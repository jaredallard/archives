// Copyright (C) 2024 archives contributors
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public
// License along with this program. If not, see
// <https://www.gnu.org/licenses/>.
//
// SPDX-License-Identifier: LGPL-3.0

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
