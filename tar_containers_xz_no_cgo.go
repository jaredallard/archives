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

// newXZWriter creates a new xz writer that writes to w. The pure-Go
// encoder has no compression levels, so level only controls the
// dictionary size, matching that of the equivalent liblzma preset.
func newXZWriter(w io.Writer, level CompressionLevel) (io.WriteCloser, error) {
	dictCap := 8 << 20
	switch level { //nolint:exhaustive // Why: Default is the initial value.
	case CompressionFastest:
		dictCap = 256 << 10
	case CompressionBetter:
		dictCap = 16 << 20
	case CompressionBest:
		dictCap = 64 << 20
	}
	return xz.WriterConfig{DictCap: dictCap}.NewWriter(w)
}
