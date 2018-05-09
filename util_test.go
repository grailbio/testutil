// Copyright 2017 GRAIL, Inc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package testutil_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/grailbio/testutil"
	"github.com/stretchr/testify/require"
)

func setEnvWithReset(t *testing.T, key string, value string) func() {
	origValue, hasValue := os.LookupEnv(key)
	err := os.Setenv(key, value)
	require.NoError(t, err)
	return func() {
		if hasValue {
			os.Setenv(key, origValue)
		} else {
			os.Unsetenv(key)
		}
	}
}

func clearEnvWithReset(t *testing.T, key string) func() {
	origValue, hasValue := os.LookupEnv(key)
	err := os.Unsetenv(key)
	require.NoError(t, err)
	return func() {
		if hasValue {
			os.Setenv(key, origValue)
		} else {
		}
	}
}

func TestGetFilePathInGrailEnv(t *testing.T) {
	defer clearEnvWithReset(t, "TEST_SRCDIR")()
	defer clearEnvWithReset(t, "TEST_WORKSPACE")()

	grailPath := "/grail"
	defer setEnvWithReset(t, "GRAIL", grailPath)()
	require.Equal(t, filepath.Join(grailPath, "foo/bar/baz"), testutil.GetFilePath("foo/bar/baz"))
}

func TestGetFilePathInBazelEnv(t *testing.T) {
	defer clearEnvWithReset(t, "GRAIL")()

	bazelSrc := "/bazel_src"
	defer setEnvWithReset(t, "TEST_SRCDIR", bazelSrc)()

	bazelSpace := "/bazel_space"
	defer setEnvWithReset(t, "TEST_WORKSPACE", bazelSpace)()

	require.Equal(t, filepath.Join(bazelSrc, "grail/foo/bar/baz"), testutil.GetFilePath("@grail//foo/bar/baz"))
	require.Equal(t, filepath.Join(bazelSrc, "bar/foo/bar/baz"), testutil.GetFilePath("@bar//foo/bar/baz"))
}
