// Copyright (C) 2026 archives contributors
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.
//
// SPDX-License-Identifier: MPL-2.0

package archives

import (
	"compress/bzip2"
	"compress/gzip"
	"io"

	"github.com/klauspost/compress/zstd"
)

// newGzipReader creates a new gzip reader from the provided reader.
func newGzipReader(r io.Reader) (io.ReadCloser, error) {
	return gzip.NewReader(r)
}

// newBzip2Reader creates a new bzip2 reader from the provided reader.
func newBzip2Reader(r io.Reader) io.ReadCloser {
	return io.NopCloser(bzip2.NewReader(r))
}

// newZstdReader creates a new zstd reader from the provided reader.
func newZstdReader(r io.Reader) (io.ReadCloser, error) {
	r, err := zstd.NewReader(r)
	if err != nil {
		return nil, err
	}
	return io.NopCloser(r), nil
}
