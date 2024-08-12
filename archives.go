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

// Package archives provides functions for working with compressed
// archives.
package archives

import (
	"errors"
	"fmt"
	"io"
	"strings"
)

// Configures extractors supported by this package and values
// initialized by the init function.
var (
	extractors = []Archiver{&tar{}}
	extensions = map[string]Archiver{}
)

// init initializes calls all extractors to register their supported
// extensions.
func init() {
	for i := range extractors {
		for _, ext := range extractors[i].Extensions() {
			extensions[ext] = extractors[i]
		}
	}
}

// OpenOptions contains the options for opening an archive.
type OpenOptions struct {
	// Extension is the extension of the archive to extract. This is
	// required.
	//
	// Extension should be complete, including the leading period. For
	// example:
	//		 .tar
	// 		 .tar.gz
	Extension string
}

// ExtractOptions contains the options for extracting an archive.
type ExtractOptions struct {
	// Extension is the extension of the archive to extract. This is
	// required.
	//
	// Extension should be complete, including the leading period. For
	// example:
	//		 .tar
	// 		 .tar.gz
	Extension string

	// PreservePermissions, if set, will preserve the permissions of the
	// files in the archive.
	//
	// Defaults to true.
	PreservePermissions *bool

	// PreserveOwnership, if set, will preserve the ownership of the files
	// in the archive. If false, the files will be owned by the user
	// running the program.
	//
	// Defaults to false.
	PreserveOwnership bool
}

// ptr returns a pointer to the provided value.
func ptr[T comparable](v T) *T {
	return &v
}

// applyDefaults applies the default values to the provided options.
func applyDefaults(opts *ExtractOptions) {
	if opts.PreservePermissions == nil {
		opts.PreservePermissions = ptr(true)
	}
}

// Open opens an archive from the provided reader. The underlying
// [Archiver] is determined by the extension of the archive.
func Open(r io.Reader, opts OpenOptions) (Archive, error) {
	if r == nil {
		return nil, fmt.Errorf("reader must not be nil")
	} else if opts.Extension == "" {
		return nil, fmt.Errorf("extension must be provided (set opts.Extension)")
	}

	ext := strings.TrimPrefix(opts.Extension, ".")

	archiver, ok := extensions[ext]
	if !ok || archiver == nil {
		return nil, fmt.Errorf("unsupported archive extension: %s", ext)
	}
	return archiver.Open(r, ext)
}

// Extract extracts an archive to the provided destination. The
// underlying [Archiver] is determined by the extension of the archive.
func Extract(r io.Reader, dest string, opts ExtractOptions) error {
	applyDefaults(&opts)

	a, err := Open(r, OpenOptions{
		Extension: opts.Extension,
	})
	if err != nil {
		return fmt.Errorf("failed to open archive: %w", err)
	}

	return extract(a, dest, &opts)
}

// PickFilterFn is a function that filters files in an archive.
type PickFilterFn func(*Header) bool

// Pick returns an [io.Reader] that returns a specific file from the
// provided [Archive]. The file is determined by the provided filter
// function.
//
// If the caller intends to pick one file from an archive, they should
// also make sure to close the archive after they are done with the
// returned [io.Reader] to prevent resource leaks.
func Pick(a Archive, filter PickFilterFn) (io.Reader, error) {
	for {
		h, err := a.Next()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil, fmt.Errorf("file not found in archive")
			}

			return nil, fmt.Errorf("failed to read archive header: %w", err)
		}

		// Only consider files.
		if h.Type != HeaderFile {
			continue
		}

		if filter(h) {
			// Return the same archive since [archive.Next] progressed the
			// reader to the file. This is a convenience to the caller.
			return a, nil
		}
	}
}

// PickFilterByName returns a [PickFilterFn] that filters files by name.
func PickFilterByName(name string) PickFilterFn {
	return func(h *Header) bool {
		return h.Name == name
	}
}
