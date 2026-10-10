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
	"compress/flate"
	"fmt"
	"io"
	"io/fs"
	"strings"
	"sync"
)

// maxZipSymlinkTarget is the maximum length of a symlink target read
// from a zip archive. This bounds memory use on malformed archives.
const maxZipSymlinkTarget = 4096

// _ ensures that zip implements the [Archiver] interface.
var _ Archiver = (&zip{})

// _ ensures that zip supports compression levels.
var _ levelArchiver = (&zip{})

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

// NewWriter returns an [ArchiveWriter] that writes a zip archive to w.
func (z *zip) NewWriter(w io.Writer, ext string) (ArchiveWriter, error) {
	return z.newWriter(w, ext, CompressionDefault)
}

// newWriter returns an [ArchiveWriter] that writes a zip archive,
// compressed with level, to w.
func (z *zip) newWriter(w io.Writer, _ string, level CompressionLevel) (ArchiveWriter, error) {
	zw := stdzip.NewWriter(w)
	if level != CompressionDefault {
		lvl := deflateLevel(level)
		zw.RegisterCompressor(stdzip.Deflate, func(w io.Writer) (io.WriteCloser, error) {
			return flate.NewWriter(w, lvl)
		})
	}
	return &zipWriter{zw: zw}, nil
}

// zipWriter implements [ArchiveWriter] for zip archives.
type zipWriter struct {
	zw *stdzip.Writer

	// cur is the writer for the contents of the current file entry, or
	// nil if the current entry has no contents.
	cur io.Writer
}

// WriteHeader starts a new zip entry for h. Hard links are not
// supported by the zip format.
func (z *zipWriter) WriteHeader(h *Header) error {
	z.cur = nil

	fh := &stdzip.FileHeader{
		Name:     h.Name,
		Modified: h.ModTime,
		Method:   stdzip.Store,
	}
	mode := h.Mode & (fs.ModePerm | fs.ModeSetuid | fs.ModeSetgid | fs.ModeSticky)

	switch h.Type {
	case HeaderFile:
		fh.Method = stdzip.Deflate
	case HeaderDir:
		mode |= fs.ModeDir
		if !strings.HasSuffix(fh.Name, "/") {
			fh.Name += "/"
		}
	case HeaderSymlink:
		mode |= fs.ModeSymlink
	case HeaderHardlink:
		return fmt.Errorf("zip archives do not support hard links: %s", h.Name)
	case HeaderUnsupported:
		return fmt.Errorf("unsupported header type (%s: %v)", h.Name, h.Type)
	default:
		return fmt.Errorf("unknown header type (%s: %v)", h.Name, h.Type)
	}
	fh.SetMode(mode)

	w, err := z.zw.CreateHeader(fh)
	if err != nil {
		return fmt.Errorf("failed to write header for %s: %w", h.Name, err)
	}

	switch h.Type { //nolint:exhaustive // Why: Other types have no contents.
	case HeaderFile:
		z.cur = w
	case HeaderSymlink:
		// Zip stores the symlink target as the entry's contents.
		if _, err := io.WriteString(w, h.Linkname); err != nil {
			return fmt.Errorf("failed to write symlink target for %s: %w", h.Name, err)
		}
	}

	return nil
}

// Write writes to the current file entry.
func (z *zipWriter) Write(p []byte) (int, error) {
	if z.cur == nil {
		return 0, fmt.Errorf("current zip entry is not a file")
	}
	return z.cur.Write(p)
}

// Close finishes the zip archive.
func (z *zipWriter) Close() error {
	return z.zw.Close()
}
