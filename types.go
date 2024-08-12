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

// Archiver is an interface for interacting with creating [Archive]s
// from [io.Reader]s.
type Archiver interface {
	// Open opens the provided reader and returns an archive. Depending on
	// the implementation, this may read the entire archive into memory
	// (e.g., zip).
	Open(r io.Reader, ext string) (Archive, error)

	// Extensions should return a list of supported extensions for this
	// extractor.
	Extensions() []string
}
