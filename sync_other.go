// Copyright (C) 2026 archives contributors
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.
//
// SPDX-License-Identifier: MPL-2.0

//go:build !linux && !darwin

package archives

import (
	"os"
	"runtime"
)

// Entries are synced individually. Windows does not support syncing
// directories.
const (
	syncEachFile = true
	syncEachDir  = runtime.GOOS != "windows"
)

// syncEntry syncs f to stable storage.
func syncEntry(f *os.File) error {
	return f.Sync()
}

// syncFinal is a no-op, as every entry has already been synced.
func syncFinal(*os.Root) error {
	return nil
}
