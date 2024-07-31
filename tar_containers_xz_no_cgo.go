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
