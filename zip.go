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

package archives

import (
	stdzip "archive/zip"
	"bytes"
	"fmt"
	"io"
	"sync"
)

// _ ensures that tar implements the [Archiver] interface.
var _ Archiver = (&zip{})

// zip implements the [Archiver] interface for zip archives.
type zip struct{}

// Extensions returns the supported extensions for the zip extractor.
func (z *zip) Extensions() []string {
	return []string{"zip"}
}

// Open creates a new [Archive] from the provided reader using the zip
// format. Due to the nature of zip archives, the entire archive is read
// into memory.
func (z *zip) Open(r io.Reader, _ string) (Archive, error) {
	b, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("failed to read archive: %w", err)
	}

	zr, err := stdzip.NewReader(bytes.NewReader(b), int64(len(b)))
	if err != nil {
		return nil, fmt.Errorf("failed to create zip reader: %w", err)
	}
	return &zipArchive{zr: zr}, nil
}

// zipArchive is an implementation of the Archive interface for zip
// archives. It is safe for concurrent use.
type zipArchive struct {
	io.ReadCloser

	mu  sync.Mutex
	pos int
	zr  *stdzip.Reader
}

// Close closes the zipArchive, rendering it unusable for I/O.
func (z *zipArchive) Close() error {
	if z.ReadCloser != nil {
		return z.ReadCloser.Close()
	}

	return nil
}

// Next returns the next file in the archive and updates the
// zipArchive's ReadCloser to point to the file's contents.
func (z *zipArchive) Next() (*Header, error) {
	z.mu.Lock()
	defer z.mu.Unlock()

	if z.pos >= len(z.zr.File) {
		return nil, io.EOF
	}

	f := z.zr.File[z.pos]
	z.pos++

	var err error
	z.ReadCloser, err = f.Open()
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}

	fType := HeaderFile
	if f.FileInfo().IsDir() {
		fType = HeaderDir
	}

	return &Header{
		Name:    f.Name,
		Type:    fType,
		Size:    int64(f.UncompressedSize64), // #nosec // Why: Not an overflow.
		Mode:    f.Mode(),
		ModTime: f.Modified,
	}, nil
}
