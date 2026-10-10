// Copyright (C) 2026 archives contributors
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.
//
// SPDX-License-Identifier: MPL-2.0

package archives

import (
	stdzip "archive/zip"
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"sync"
)

// maxZipSymlinkTarget is the maximum length of a symlink target read
// from a zip archive. This bounds memory use on malformed archives.
const maxZipSymlinkTarget = 4096

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

	mode := f.Mode()
	h := &Header{
		Name:    f.Name,
		Size:    int64(f.UncompressedSize64), // #nosec // Why: Not an overflow.
		Mode:    mode,
		ModTime: f.Modified,
	}

	//nolint:exhaustive // Why: All other types are unsupported.
	switch mode.Type() {
	case 0:
		h.Type = HeaderFile
	case fs.ModeDir:
		h.Type = HeaderDir
	case fs.ModeSymlink:
		// Zip stores the symlink target as the entry's contents.
		h.Type = HeaderSymlink
		target, err := io.ReadAll(io.LimitReader(z.ReadCloser, maxZipSymlinkTarget+1))
		if err != nil {
			return nil, fmt.Errorf("failed to read symlink target for %s: %w", f.Name, err)
		}
		if len(target) > maxZipSymlinkTarget {
			return nil, fmt.Errorf("symlink target for %s exceeds %d bytes", f.Name, maxZipSymlinkTarget)
		}
		h.Linkname = string(target)
	default:
		h.Type = HeaderUnsupported
	}

	return h, nil
}
