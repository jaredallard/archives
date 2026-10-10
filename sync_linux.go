// Copyright (C) 2026 archives contributors
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.
//
// SPDX-License-Identifier: MPL-2.0

package archives

import (
	"os"

	"golang.org/x/sys/unix"
)

// On Linux, a single syncfs at the end writes back every dirty inode
// and directory entry on the filesystem, so entries aren't synced
// individually.
const (
	syncEachFile = false
	syncEachDir  = false
)

// syncEntry is unused on Linux, see [syncFinal].
func syncEntry(*os.File) error {
	return nil
}

// syncFinal syncs the filesystem containing root. Writeback errors are
// only reported on Linux 5.8+.
func syncFinal(root *os.Root) error {
	return withRootFd(root, unix.Syncfs)
}
