// Copyright (C) 2026 archives contributors
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.
//
// SPDX-License-Identifier: MPL-2.0

package archives

import (
	"io"
	"os"
	"time"
)

// HeaderType denotes a type of header. Not all extractors may support
// all header types.
type HeaderType int

// Contains the supported header types.
const (
	HeaderFile HeaderType = iota
	HeaderDir

	// HeaderSymlink is a symbolic link. The target is stored in
	// [Header.Linkname].
	HeaderSymlink

	// HeaderHardlink is a hard link to another entry in the archive. The
	// archive path of the target is stored in [Header.Linkname].
	HeaderHardlink

	// HeaderUnsupported is an entry type that cannot be extracted by
	// this package (e.g., a device node or FIFO).
	HeaderUnsupported
)

// Header represents metadata about a file in an archive.
type Header struct {
	// Name is the name of the file or directory.
	Name string

	// Type is the type of header.
	Type HeaderType

	// Size is the size of the file. If the header is a directory, this
	// will be 0.
	Size int64

	// Mode is the file mode.
	Mode os.FileMode

	// Linkname is the target of a link. For [HeaderSymlink] this is the
	// symlink target as recorded in the archive. For [HeaderHardlink]
	// this is the archive path of the linked entry.
	Linkname string

	// AccessTime is the time the file was last accessed.
	AccessTime time.Time

	// ModTime is the time the file was last modified.
	ModTime time.Time

	// UID is the user ID of the file.
	UID int

	// GID is the group ID of the file.
	GID int
}

// Archive represents an archive containing folders and files.
type Archive interface {
	io.Reader

	// Close closes the archive. No other methods should be called after
	// this.
	Close() error

	// Next returns a header for the next file in the archive. If there
	// are no more files, it will return io.EOF. When called, the embedded
	// [io.Reader] will target the file in the returned [Header].
	Next() (*Header, error)
}

// ArchiveWriter writes entries to an archive. Each entry is started
// with [ArchiveWriter.WriteHeader], after which the contents of a
// [HeaderFile] entry are written using the embedded [io.Writer].
type ArchiveWriter interface {
	io.Writer

	// WriteHeader starts a new entry described by h. For [HeaderFile]
	// entries, h.Size must be set to the exact number of bytes that will
	// be written. The permission, setuid, setgid and sticky bits of
	// h.Mode are stored, the type is determined by h.Type. AccessTime is
	// not stored, and UID/GID are only stored by formats that support
	// them (e.g., tar).
	WriteHeader(h *Header) error

	// Close finishes writing the archive, flushing any compression. It
	// does not close the underlying [io.Writer].
	Close() error
}

// Archiver is an interface for reading [Archive]s from [io.Reader]s
// and writing them to [io.Writer]s with an [ArchiveWriter].
type Archiver interface {
	// Open opens the provided reader and returns an archive. Depending on
	// the implementation, this may read the entire archive into memory
	// (e.g., zip).
	Open(r io.Reader, ext string) (Archive, error)

	// NewWriter returns an [ArchiveWriter] that writes an archive of the
	// provided extension to w.
	NewWriter(w io.Writer, ext string) (ArchiveWriter, error)

	// Extensions should return a list of supported extensions for this
	// extractor.
	Extensions() []string
}
