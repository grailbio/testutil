package h

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/davecgh/go-spew/spew"
)

func diffStrings(got, want interface{}) string {
	toString := func(v interface{}) string {
		if s, ok := v.(string); ok {
			return s
		}
		return spew.Sdump(v)
	}
	tempDir, err := ioutil.TempDir("", "")
	if err != nil {
		panic(fmt.Sprintf("tempdir: %v", err))
	}
	defer os.RemoveAll(tempDir) // nolint: errcheck

	gotPath := filepath.Join(tempDir, "got")
	wantPath := filepath.Join(tempDir, "want")
	if err := ioutil.WriteFile(gotPath, []byte(toString(got)), 0600); err != nil {
		panic(err)
	}
	if err := ioutil.WriteFile(wantPath, []byte(toString(want)), 0600); err != nil {
		panic(err)
	}
	cmd := exec.Command("diff", "-C", "3", gotPath, wantPath)
	cmd.Stderr = os.Stderr
	output, _ := cmd.Output()
	return string(output)
}
