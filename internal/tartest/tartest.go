// Copyright (C) 2026 archives contributors
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.
//
// SPDX-License-Identifier: MPL-2.0

// Package tartest contains helpers for creating tar archives for usage
// in tests in the archives package.
package tartest

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"

	"github.com/klauspost/compress/zstd"
	xznocgo "github.com/ulikunitz/xz"
)

// Container represents the container of a tar file.
type Container int

const (
	// ContainerNone denotes no container. This is the default.
	ContainerNone Container = iota
	// ContainerGz is a tar archive compressed with gzip.
	ContainerGz
	// ContainerXz is a tar archive compressed with xz.
	ContainerXz
	// ContainerBz2 is a tar archive compressed with bzip2.
	ContainerBz2
	// ContainerZstd is a tar archive compressed with zstd.
	ContainerZstd
)

// Options is a struct for interacting with containers.
type Options struct {
	Container Container
}

// OptionFn modifies a [Options] struct.
type OptionFn func(*Options)

// WithContainer denotes that a specific container should be used when
// creating the tar archive.
func WithContainer(c Container) OptionFn {
	return func(o *Options) {
		o.Container = c
	}
}

// Create creates a new tar archive with a single file, file.txt,
// containing the contents "hello world".
func Create(options ...OptionFn) (io.Reader, error) {
	opts := &Options{Container: ContainerNone}
	for _, o := range options {
		o(opts)
	}

	buf := new(bytes.Buffer)

	var container io.WriteCloser
	switch opts.Container { //nolint:exhaustive // Why: None is zero value.
	case ContainerGz:
		container = gzip.NewWriter(buf)
	case ContainerXz:
		var err error
		container, err = xznocgo.NewWriter(buf)
		if err != nil {
			return nil, fmt.Errorf("failed to create xz writer: %w", err)
		}
	case ContainerBz2:
		var err error
		container, err = newBzip2Writer(buf)
		if err != nil {
			return nil, fmt.Errorf("failed to create bzip2 writer: %w", err)
		}
	case ContainerZstd:
		var err error
		container, err = zstd.NewWriter(buf)
		if err != nil {
			return nil, fmt.Errorf("failed to create zstd writer: %w", err)
		}
	}

	var tw *tar.Writer
	if container == nil {
		tw = tar.NewWriter(buf)
	} else {
		tw = tar.NewWriter(container)
		defer container.Close() //nolint:errcheck // Why: Best effort
	}

	contents := []byte("hello world")
	if err := tw.WriteHeader(&tar.Header{
		Name: "file.txt",
		Size: int64(len(contents)),
		Mode: 0o644,
	}); err != nil {
		return nil, fmt.Errorf("failed to write header: %w", err)
	}

	_, err := tw.Write(contents)
	if err != nil {
		return nil, fmt.Errorf("failed to write contents: %w", err)
	}

	if err := tw.Close(); err != nil {
		return nil, fmt.Errorf("failed to close tar writer: %w", err)
	}

	return buf, nil
}
