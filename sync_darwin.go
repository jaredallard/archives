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

// On macOS, entries are synced individually with a plain fsync, which
// hands them to the storage device, followed by a single F_FULLFSYNC
// at the end to flush the device's cache.
const (
	syncEachFile = true
	syncEachDir  = true
)

// syncEntry performs a plain fsync of f. Unlike [os.File.Sync], this
// does not flush the storage device's cache, see [syncFinal].
func syncEntry(f *os.File) error {
	return withFd(f, unix.Fsync)
}

// syncFinal flushes the cache of the storage device containing root,
// making all previously synced entries durable.
func syncFinal(root *os.Root) error {
	return withRootFd(root, func(fd int) error {
		_, err := unix.FcntlInt(uintptr(fd), unix.F_FULLFSYNC, 0)
		return err
	})
}
