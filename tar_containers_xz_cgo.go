// Copyright (C) 2026 archives contributors
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.
//
// SPDX-License-Identifier: MPL-2.0

//go:build cgo

package archives

import (
	"io"

	"github.com/jamespfennell/xz"
)

// newXZReader creates a new xz reader from the provided reader.
func newXZReader(r io.Reader) (io.ReadCloser, error) {
	return xz.NewReader(r), nil
}

// newXZWriter creates a new xz writer that writes to w. level is
// mapped to a liblzma preset.
func newXZWriter(w io.Writer, level CompressionLevel) (io.WriteCloser, error) {
	preset := xz.DefaultCompression
	switch level { //nolint:exhaustive // Why: Default is the initial value.
	case CompressionFastest:
		preset = xz.BestSpeed
	case CompressionBetter:
		preset = 7
	case CompressionBest:
		preset = xz.BestCompression
	}
	return xz.NewWriterLevel(w, preset), nil
}
