// Copyright 2011 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cmd类

import (
	"os/exec"
	"testing"
)

var nonExistentPaths = []string{
	"some-non-existent-path",
	"non-existent-path/slashed",
}

func TestLookPathNotFound(t *testing.T) {
	for _, name := range nonExistentPaths {
		path, err := I查找路径(name)
		if err == nil {
			t.Fatalf("I查找路径 found %q in $PATH", name)
		}
		if path != "" {
			t.Fatalf("I查找路径 path == %q when err != nil", path)
		}
		perr, ok := err.(*exec.Error)
		if !ok {
			t.Fatal("I查找路径 error is not an exec.Error")
		}
		if perr.Name != name {
			t.Fatalf("want Error name %q, got %q", name, perr.Name)
		}
	}
}
