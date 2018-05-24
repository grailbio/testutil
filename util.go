// Copyright 2017 GRAIL, Inc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package testutil

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"runtime/debug"
	"strings"

	"v.io/x/lib/gosh"
)

// MockTB is a mock implementation of gosh.TB. FailNow and Fatalf will
// set Failed to true. Logf and Fatalf write their log message to Result.
// MockTB is intended for building negatived tests.
type MockTB struct {
	Failed bool
	Result string
}

// FailNow implements TB.
func (m *MockTB) FailNow() {
	m.Failed = true
}

// Logf implements TB.
func (m *MockTB) Logf(format string, args ...interface{}) {
	m.Result = fmt.Sprintf(format, args...)
}

// Fatalf implements TB.
func (m *MockTB) Fatalf(format string, args ...interface{}) {
	m.Failed = true
	m.Result = fmt.Sprintf(format, args...)
}

// Caller returns a string of the form <file>:<line> for the caller
// at the specified depth.
func Caller(depth int) string {
	_, file, line, _ := runtime.Caller(depth + 1)
	return fmt.Sprintf("%s:%d", filepath.Base(file), line)
}

// ExpectedError tests for an expected error and associated error message.
func ExpectedError(t interface {
	Errorf(string, ...interface{})
}, depth int, err error, msg string) {
	_, file, line, _ := runtime.Caller(depth + 1)
	if err == nil || !strings.Contains(err.Error(), msg) {
		t.Errorf("%v:%v: got %v, want an error containing %v", filepath.Base(file), line, err, msg)
	}
}

// NoCleanupOnError avoids calling the supplied cleanup function when
// a test has failed or paniced. The Log function is called with args
// when the test has failed and is typically used to log the location
// of the state that would have been removed by the cleanup function.
// Common usage would be:
//
// tempdir, cleanup := testutil.TempDir(t, "", "scandb-state-")
// defer testutil.NoCleanupOnError(t, cleanup, "tempdir:", tempdir)
func NoCleanupOnError(t interface {
	Fail()
	Log(...interface{})
	Failed() bool
}, cleanup func(), args ...interface{}) {
	if t.Failed() {
		if len(args) > 0 {
			t.Log(args...)
		}
		return
	}
	if recover() != nil {
		debug.PrintStack()
		if len(args) > 0 {
			t.Log(args...)
		}
		t.Fail()
		return
	}
	cleanup()
}

// FailOnError will call Fatal if the supplied error parameter is non-nil,
// with the location of its caller at the specified depth and the non-nil error.
func FailOnError(depth int, t interface {
	Fatal(...interface{})
}, err error) {
	if err != nil {
		t.Fatal(Caller(depth), err)
	}
}

// GetFilePath detects if we're running under "bazel test". If so, it builds
// a path to the test data file based on Bazel environment variables.
// Otherwise, it tries to build a path relative to $GRAIL.
// If that fails, it returns the input path unchanged.
//
// relativePath will need to be prefixed with a Bazel workspace designation if
// the paths go across workspaces. Only certain workspaces are recognized when
// running tests for the Go tool.  Add more workspaces as necessary in the map
// below.
func GetFilePath(relativePath string) string {
	grailPath, hasGrailPath := os.LookupEnv("GRAIL")
	bazelPath, hasBazelPath := os.LookupEnv("TEST_SRCDIR")
	workspace, hasWorkspace := os.LookupEnv("TEST_WORKSPACE")

	isGrail := hasGrailPath
	isBazel := hasBazelPath && hasWorkspace

	workspaceFromPath := regexp.MustCompile("@([^/]*)//(.*)")
	matches := workspaceFromPath.FindStringSubmatch(relativePath)
	if len(matches) > 0 {
		workspace = matches[1]
		relativePath = matches[2]
	}

	switch {
	case isBazel:
		return filepath.Join(bazelPath, workspace, relativePath)
	case isGrail:
		// Provide a mapping from bazel workspaces to a $GRAIL-relative path.
		// TODO(treaster): Figure out how to do this mapping dynamically.
		knownWorkspaces := map[string]string{
			"":      "",
			"grail": "",
		}
		expandedPath, hasWorkspace := knownWorkspaces[workspace]
		if !hasWorkspace {
			panic(fmt.Sprintf("Unrecognized workspace %q", workspace))
		}
		return filepath.Join(grailPath, expandedPath, relativePath)
	}

	panic("Unexpected test environment. Should be running with either $GRAIL or in a bazel build space.")
	return ""
}

// GetTmpDir will retrieve/generate a test-specific directory appropriate
// for writing scratch data. When running under Bazel, Bazel should clean
// up the directory. However, when running under vanilla Go tooling, it will
// not be cleaned up. Thus, it's probably best for a test to clean up
// any test directories itself.
func GetTmpDir() string {
	bazelPath, hasBazelPath := os.LookupEnv("TEST_TMPDIR")
	if hasBazelPath {
		return bazelPath
	}

	tmpPath, err := ioutil.TempDir("/tmp", "go_test_")
	if err != nil {
		panic(err.Error)
	}
	return tmpPath
}

// WriteTmp writes the supplied contents to a temporary file and returns the
// name of that file.
func WriteTmp(t interface {
	Fatalf(string, ...interface{})
}, depth int, contents string) string {
	f, err := ioutil.TempFile("", "WriteTmp-")
	if err != nil {
		t.Fatalf("%v: %v", Caller(depth+1), err)
	}
	defer f.Close()
	if _, err := f.Write([]byte(fmt.Sprintf("%s", contents))); err != nil {
		t.Fatalf("%v: %v", Caller(depth+1), err)
	}
	return f.Name()
}

// CompareFile compares the supplied contents against the contents of the
// specified file and if they differ calls t.Errorf and displays a diff -u of
// them. The file is assumed to be in ./testadata. If specified the strip
// function can be used to cleanup the contents to be compared to remove
// things such as dates or other spurious information that's not relevant
// to the comparison.
func CompareFile(t interface {
	Fatalf(string, ...interface{})
	Logf(string, ...interface{})
	Errorf(string, ...interface{})
	FailNow()
}, depth int, contents string, golden string, strip func(string) string) {
	fn := filepath.Join("testdata", golden)
	data, err := ioutil.ReadFile(fn)
	if err != nil {
		t.Fatalf("%v: %v", Caller(depth+1), err)
	}
	got, want := contents, string(data)
	if strip != nil {
		got, want = strip(got), strip(want)
	}
	if got != want {
		gf := WriteTmp(t, depth+1, got)
		defer os.Remove(gf)
		cmd := exec.Command("diff", "-u", gf, fn)
		diff, _ := cmd.CombinedOutput()
		t.Logf("%v: got %v", Caller(depth+1), got)
		t.Logf("%v: diff %v %v", Caller(depth+1), gf, fn)
		t.Errorf("%v: %v: got != want: diff %v", Caller(depth+1), golden, string(diff))
	}
}

// CompareFiles compares 2 files in the same manner as CompareFile.
func CompareFiles(t interface {
	Fatalf(string, ...interface{})
	Logf(string, ...interface{})
	Errorf(string, ...interface{})
	FailNow()
}, depth int, a, golden string, strip func(string) string) {
	ac, err := ioutil.ReadFile(a)
	if err != nil {
		t.Fatalf("%v: %v", Caller(depth+1), err)
	}
	CompareFile(t, depth+1, string(ac), golden, strip)
}

// IsBazel checks if the current process is started by "bazel test".
func IsBazel() bool {
	return os.Getenv("TEST_TMPDIR") != "" && os.Getenv("RUNFILES_DIR") != ""
}

// GoExecutable returns the Go executable for "path", or builds the executable
// and returns its path. The latter happens when the caller is not running under
// Bazel. "path" must start with "//go/src/grail.com/".  For example,
// "//go/src/grail.com/cmd/bio-metrics/bio-metrics".
func GoExecutable(t interface {
	Fatalf(string, ...interface{})
},
	sh *gosh.Shell,
	path string) string {
	re := regexp.MustCompile("^//go/src/(.*/([^/]+))/([^/]+)$")
	match := re.FindStringSubmatch(path)
	if match == nil || match[2] != match[3] {
		t.Fatalf("%v: target must be of format \"//go/src/path/target/target\"",
			path)
	}
	if IsBazel() {
		expandedPath := GetFilePath(path)
		if _, err := os.Stat(expandedPath); err == nil {
			return expandedPath
		}
		pattern := GetFilePath(fmt.Sprintf("//go/src/%s/*/%s", match[1], match[2]))
		paths, err := filepath.Glob(pattern)
		if err != nil {
			t.Fatalf("glob %v: %v", pattern, err)
		}
		if len(paths) != 1 {
			t.Fatalf("Pattern %s must match exactly one executable, but found %v", pattern, paths)
		}
		return paths[0]
	}
	tempdir := sh.MakeTempDir()
	return gosh.BuildGoPkg(sh, tempdir, "grail.com/"+match[1])
}
