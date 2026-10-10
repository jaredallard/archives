// Copyright (C) 2026 archives contributors
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.
//
// SPDX-License-Identifier: MPL-2.0

// Package archives provides functions for working with compressed
// archives.
package archives

import (
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"
)

// Configures extractors supported by this package and values
// initialized by the init function.
var (
	extractors = []Archiver{&tar{}, &zip{}}
	extensions = map[string]Archiver{}
)

// init initializes calls all extractors to register their supported
// extensions.
//
//nolint:gochecknoinits // Why: This is acceptable for this package.
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
	// Extension should be complete, including the leading period. This
	// should be the output of [Ext] to ensure that the extension contains
	// all formats (e.g., .tar.gz and .zip). If you opt to use
	// [filepath.Ext] it will not include the second extension.
	Extension string
}

// WriterOptions contains the options for writing an archive.
type WriterOptions struct {
	// Extension is the extension of the archive to write. This is
	// required.
	//
	// Extension should be complete, including the leading period. For
	// example:
	//		 .tar
	// 		 .tar.gz
	Extension string
}

// CreateOptions contains the options for creating an archive from a
// directory.
type CreateOptions struct {
	// Extension is the extension of the archive to create. This is
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

	// Sync, if set, ensures that the extracted contents are durable on
	// stable storage before returning (e.g., when restoring a disk).
	//
	// On Linux, this is a single syncfs(2) of the filesystem containing
	// the destination once extraction completes. This also writes back
	// unrelated dirty data on that filesystem, does not cover other
	// filesystems mounted inside the destination, and only reports
	// write errors on Linux 5.8+. On macOS, every extracted file and
	// directory is fsynced, followed by a single flush of the storage
	// device's cache. On other platforms, every extracted file and
	// directory (except on Windows) is fsynced.
	//
	// Defaults to false.
	Sync bool
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

// Ext returns the extension of a file name based on the supported
// extensions in this package. If the extension is not supported, the
// output of [filepath.Ext] will be returned instead.
//
// Examples:
//
//	archives.Ext("file.tar.gz")   // ".tar.gz"
//	archives.Ext("file.zip")      // ".zip"
//	archives.Ext("file.unknown")  // ".unknown"
func Ext(name string) string {
	// try supported extensions first
	for ext := range extensions {
		if strings.HasSuffix(name, ext) {
			// Return leading period to match [filepath.Ext].
			return "." + ext
		}
	}

	// fallback to filepath.Ext
	return strings.ToLower(filepath.Ext(name))
}

// Open opens an archive from the provided reader. The underlying
// [Archiver] is determined by the extension of the archive.
func Open(r io.Reader, opts OpenOptions) (Archive, error) {
	if r == nil {
		return nil, fmt.Errorf("reader must not be nil")
	}

	archiver, ext, err := archiverFor(opts.Extension)
	if err != nil {
		return nil, err
	}
	return archiver.Open(r, ext)
}

// archiverFor returns the [Archiver] for the provided extension, along
// with the extension without its leading period.
func archiverFor(extension string) (Archiver, string, error) {
	if extension == "" {
		return nil, "", fmt.Errorf("extension must be provided (set opts.Extension)")
	}

	ext := strings.TrimPrefix(extension, ".")

	archiver, ok := extensions[ext]
	if !ok || archiver == nil {
		return nil, "", fmt.Errorf("unsupported archive extension: %s", ext)
	}
	return archiver, ext, nil
}

// NewWriter returns an [ArchiveWriter] that writes an archive to w. The
// underlying [Archiver] is determined by the extension of the archive.
//
// The caller must call [ArchiveWriter.Close] to finish the archive.
// Creating bzip2 compressed archives is not supported.
func NewWriter(w io.Writer, opts WriterOptions) (ArchiveWriter, error) {
	if w == nil {
		return nil, fmt.Errorf("writer must not be nil")
	}

	archiver, ext, err := archiverFor(opts.Extension)
	if err != nil {
		return nil, err
	}
	return archiver.NewWriter(w, ext)
}

// Extract extracts an archive to the provided destination. The
// underlying [Archiver] is determined by the extension of the archive.
//
// Symlinks and hard links are recreated. Symlink targets are restored
// as recorded in the archive, but no entry is ever written outside of
// dest, including through symlinks. Existing files at an entry's path
// are replaced. Directory metadata (permissions, ownership and times)
// is applied after all entries have been extracted. Device nodes and
// FIFOs are not supported and cause an error.
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

// Create creates an archive containing the contents of the directory
// src and writes it to w. The underlying [Archiver] is determined by
// the extension of the archive. Entry names are relative to src.
//
// Symlinks are never followed. They are stored as symlinks with their
// target exactly as it is on disk, even if it points outside of src or
// does not exist. Ownership is not stored and hard links are stored as
// regular files. Device nodes, FIFOs and sockets are not supported and
// cause an error. w must not write into src.
func Create(w io.Writer, src string, opts CreateOptions) error {
	//nolint:staticcheck // Why: CreateOptions will gain fields WriterOptions doesn't have.
	aw, err := NewWriter(w, WriterOptions{
		Extension: opts.Extension,
	})
	if err != nil {
		return fmt.Errorf("failed to create archive: %w", err)
	}

	if err := create(aw, src); err != nil {
		aw.Close() //nolint:errcheck // Why: Best effort, already failed.
		return err
	}

	if err := aw.Close(); err != nil {
		return fmt.Errorf("failed to finish archive: %w", err)
	}
	return nil
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
