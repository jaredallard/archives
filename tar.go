// Copyright (C) 2026 archives contributors
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.
//
// SPDX-License-Identifier: MPL-2.0

package archives

import (
	stdtar "archive/tar"
	"fmt"
	"io"
	"io/fs"
	"strings"
)

// _ ensures that tar implements the [Archiver] interface.
var _ Archiver = (&tar{})

// tar implements the [Archiver] interface for tar archives and their
// compressed variants.
type tar struct{}

// Extensions returns the supported extensions for the tar extractor.
func (t *tar) Extensions() []string {
	return []string{"tar", "tgz", "tar.gz", "txz", "tar.xz", "tbz2", "tar.bz2", "tar.zst"}
}

func (t *tar) Open(r io.Reader, ext string) (Archive, error) {
	// Determine if we're dealing with a compressed tar archive and if so,
	// create the appropriate reader.
	var container io.ReadCloser
	switch ext {
	case "tar":
		container = io.NopCloser(r)
	case "tgz", "tar.gz":
		var err error
		container, err = newGzipReader(r)
		if err != nil {
			return nil, fmt.Errorf("failed to create gzip reader: %w", err)
		}
	case "tbz2", "tar.bz2":
		container = newBzip2Reader(r)
	case "txz", "tar.xz":
		var err error
		container, err = newXZReader(r)
		if err != nil {
			return nil, fmt.Errorf("failed to create xz reader: %w", err)
		}
	case "tar.zst":
		var err error
		container, err = newZstdReader(r)
		if err != nil {
			return nil, fmt.Errorf("failed to create zstd reader: %w", err)
		}
	default:
		// This only happens if we're missing a case in the switch statement.
		return nil, fmt.Errorf("unsupported tar extension: %s", ext)
	}

	tr := stdtar.NewReader(container)
	return &tarArchive{tr, container}, nil
}

// NewWriter returns an [ArchiveWriter] that writes a tar archive,
// compressed according to ext, to w.
func (t *tar) NewWriter(w io.Writer, ext string) (ArchiveWriter, error) {
	var container io.WriteCloser
	switch ext {
	case "tar":
		container = nopWriteCloser{w}
	case "tgz", "tar.gz":
		container = newGzipWriter(w)
	case "tbz2", "tar.bz2":
		return nil, fmt.Errorf("creating bzip2 archives is not supported")
	case "txz", "tar.xz":
		var err error
		container, err = newXZWriter(w)
		if err != nil {
			return nil, fmt.Errorf("failed to create xz writer: %w", err)
		}
	case "tar.zst":
		var err error
		container, err = newZstdWriter(w)
		if err != nil {
			return nil, fmt.Errorf("failed to create zstd writer: %w", err)
		}
	default:
		// This only happens if we're missing a case in the switch statement.
		return nil, fmt.Errorf("unsupported tar extension: %s", ext)
	}

	return &tarWriter{stdtar.NewWriter(container), container}, nil
}

// nopWriteCloser wraps an [io.Writer] with a no-op Close method.
type nopWriteCloser struct {
	io.Writer
}

func (nopWriteCloser) Close() error { return nil }

// tarWriter implements [ArchiveWriter] for tar archives.
type tarWriter struct {
	*stdtar.Writer
	container io.WriteCloser
}

// WriteHeader writes the tar header for h.
func (t *tarWriter) WriteHeader(h *Header) error {
	th := &stdtar.Header{
		Name:     h.Name,
		Linkname: h.Linkname,
		Mode:     tarMode(h.Mode),
		Uid:      h.UID,
		Gid:      h.GID,
		ModTime:  h.ModTime,
		Format:   stdtar.FormatPAX,
	}

	switch h.Type {
	case HeaderFile:
		th.Typeflag = stdtar.TypeReg
		th.Size = h.Size
	case HeaderDir:
		th.Typeflag = stdtar.TypeDir
		if !strings.HasSuffix(th.Name, "/") {
			th.Name += "/"
		}
	case HeaderSymlink:
		th.Typeflag = stdtar.TypeSymlink
	case HeaderHardlink:
		th.Typeflag = stdtar.TypeLink
	case HeaderUnsupported:
		return fmt.Errorf("unsupported header type (%s: %v)", h.Name, h.Type)
	default:
		return fmt.Errorf("unknown header type (%s: %v)", h.Name, h.Type)
	}

	if err := t.Writer.WriteHeader(th); err != nil {
		return fmt.Errorf("failed to write header for %s: %w", h.Name, err)
	}
	return nil
}

// Close finishes the tar archive and flushes any compression.
func (t *tarWriter) Close() error {
	if err := t.Writer.Close(); err != nil {
		t.container.Close() //nolint:errcheck // Why: Best effort, already failed.
		return err
	}
	return t.container.Close()
}

// tarMode converts the permission and special bits of mode into a tar
// header mode.
func tarMode(mode fs.FileMode) int64 {
	m := int64(mode.Perm())
	if mode&fs.ModeSetuid != 0 {
		m |= 0o4000
	}
	if mode&fs.ModeSetgid != 0 {
		m |= 0o2000
	}
	if mode&fs.ModeSticky != 0 {
		m |= 0o1000
	}
	return m
}

type tarArchive struct {
	*stdtar.Reader
	closer io.Closer
}

func (t *tarArchive) Close() error {
	return t.closer.Close()
}

func (t *tarArchive) Next() (*Header, error) {
	h, err := t.Reader.Next()
	// PAX global headers carry metadata for the archive, not an entry.
	for err == nil && h.Typeflag == stdtar.TypeXGlobalHeader {
		h, err = t.Reader.Next()
	}
	if err != nil {
		return nil, err
	}

	var hType HeaderType
	//nolint:staticcheck // Why: TypeRegA is deprecated but still found in old archives.
	switch h.Typeflag {
	case stdtar.TypeDir:
		hType = HeaderDir
	case stdtar.TypeSymlink:
		hType = HeaderSymlink
	case stdtar.TypeLink:
		hType = HeaderHardlink
	case stdtar.TypeReg, stdtar.TypeRegA, stdtar.TypeCont, stdtar.TypeGNUSparse:
		hType = HeaderFile
	default:
		hType = HeaderUnsupported
	}

	return &Header{
		Name:       h.Name,
		Type:       hType,
		Mode:       h.FileInfo().Mode(),
		Linkname:   h.Linkname,
		Size:       h.Size,
		AccessTime: h.AccessTime,
		ModTime:    h.ModTime,
		UID:        h.Uid,
		GID:        h.Gid,
	}, nil
}
