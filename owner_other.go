// Copyright (C) 2026 archives contributors
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.
//
// SPDX-License-Identifier: MPL-2.0

//go:build !unix

package archives

import "io/fs"

// fileOwner always reports that ownership is unavailable, as this
// platform does not expose user and group IDs.
func fileOwner(_ fs.FileInfo) (uid, gid int, ok bool) {
	return 0, 0, false
}
