// Copyright (C) 2026 archives contributors
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.
//
// SPDX-License-Identifier: MPL-2.0

//go:build !cgo

package archives

import (
	"bufio"
	"io"

	"github.com/ulikunitz/xz"
)

// newXZReader creates a new xz reader from the provided reader.
func newXZReader(r io.Reader) (io.ReadCloser, error) {
	wr, err := xz.NewReader(bufio.NewReader(r))
	if err != nil {
		return nil, err
	}

	return io.NopCloser(wr), nil
}
