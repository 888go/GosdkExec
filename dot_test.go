// Copyright 2022 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cmd类_test

import (
	. "e.coding.net/gogit/go/gosdk/core/os_exec_cn"
	"e.coding.net/gogit/go/gosdk/internal/testenv"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

var pathVar = func() string {
	if runtime.GOOS == "plan9" {
		return "path"
	}
	return "PATH"
}()

func TestLookPath(t *testing.T) {
	testenv.MustHaveExec(t)

	tmpDir := filepath.Join(t.TempDir(), "testdir")
	if err := os.Mkdir(tmpDir, 0777); err != nil {
		t.Fatal(err)
	}

	executable := "execabs-test"
	if runtime.GOOS == "windows" {
		executable += ".exe"
	}
	if err := os.WriteFile(filepath.Join(tmpDir, executable), []byte{1, 2, 3}, 0777); err != nil {
		t.Fatal(err)
	}
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := os.Chdir(cwd); err != nil {
			panic(err)
		}
	}()
	if err = os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PWD", tmpDir)
	t.Logf(". is %#q", tmpDir)

	origPath := os.Getenv(pathVar)

	//在PATH中添加“.”，以便exec.LookPath在所有系统的当前目录中查找。
	// 并尝试用“../testdir”来欺骗它。
	for _, errdot := range []string{"1", "0"} {
		t.Run("GODEBUG=execerrdot="+errdot, func(t *testing.T) {
			t.Setenv("GODEBUG", "execerrdot="+errdot)
			for _, dir := range []string{".", "../testdir"} {
				t.Run(pathVar+"="+dir, func(t *testing.T) {
					t.Setenv(pathVar, dir+string(filepath.ListSeparator)+origPath)
					good := dir + "/execabs-test"
					if found, err := I查找路径(good); err != nil || !strings.HasPrefix(found, good) {
						t.Fatalf(`I查找路径(%#q) = %#q, %v, want "%s...", nil`, good, found, err, good)
					}
					if runtime.GOOS == "windows" {
						good = dir + `\execabs-test`
						if found, err := I查找路径(good); err != nil || !strings.HasPrefix(found, good) {
							t.Fatalf(`I查找路径(%#q) = %#q, %v, want "%s...", nil`, good, found, err, good)
						}
					}

					_, err := I查找路径("execabs-test")
					if errdot == "1" {
						if err == nil {
							t.Fatalf("I查找路径 didn't fail when finding a non-relative path")
						} else if !errors.Is(err, exec.ErrDot) {
							测试 := errors.Is(err, exec.ErrDot)
							fmt.Println(测试)
							t.Fatalf("I查找路径 returned unexpected error: want Is ErrDot, got %q", err)
						}
					} else {
						if err != nil {
							t.Fatalf("I查找路径 failed unexpectedly: %v", err)
						}
					}

					cmd := I设置命令("execabs-test")
					if errdot == "1" {
						if cmd.Cmd父类.Err == nil {
							t.Fatalf("I设置命令 didn't fail when finding a non-relative path")
						} else if !errors.Is(cmd.Cmd父类.Err, exec.ErrDot) {
							t.Fatalf("I设置命令 returned unexpected error: want Is ErrDot, got %q", cmd.Cmd父类.Err)
						}
						cmd.Cmd父类.Err = nil
					} else {
						if cmd.Cmd父类.Err != nil {
							t.Fatalf("I设置命令 failed unexpectedly: %v", err)
						}
					}

					// 清除cmd.Err应该可以继续执行，并且应该失败，因为它不是有效的二进制文件。
					if err := cmd.I运行(); err == nil {
						t.Fatalf("I运行 did not fail: expected exec error")
					} else if errors.Is(err, exec.ErrDot) {
						t.Fatalf("I运行 returned unexpected error ErrDot: want error like ENOEXEC: %q", err)
					}
				})
			}
		})
	}

	// 测试PATH中的第一个条目是当前目录的绝对名称时的行为。
	//
	// 在Windows上，根据进程环境，“.”可以隐式包含在显式%PATH%之前，也可以不隐式包含；
	// see https://go.dev/issue/4394.
	//
	// 如果“.”中的相对项解析为与从%PATH%中的绝对项解析的可执行文件相同的可执行程序，LookPath应返回路径的绝对版本，而不是ErrDot。
	// (See https://go.dev/issue/53536.)
	//
	// 如果PATH不隐式包含“.”（例如在Unix平台上，或在配置了NoDefaultCurrentDirectoryInExePath的Windows上）,
	//那么无论“.”的行为如何，该查找都应该成功，因此即使在这些平台上，作为控制用例运行也可能很有用。
	t.Run(pathVar+"=$PWD", func(t *testing.T) {
		t.Setenv(pathVar, tmpDir+string(filepath.ListSeparator)+origPath)
		good := filepath.Join(tmpDir, "execabs-test")
		if found, err := I查找路径(good); err != nil || !strings.HasPrefix(found, good) {
			t.Fatalf(`I查找路径(%#q) = %#q, %v, want \"%s...\", nil`, good, found, err, good)
		}

		if found, err := I查找路径("execabs-test"); err != nil || !strings.HasPrefix(found, good) {
			t.Fatalf(`I查找路径(%#q) = %#q, %v, want \"%s...\", nil`, "execabs-test", found, err, good)
		}

		cmd := I设置命令("execabs-test")
		if cmd.Cmd父类.Err != nil {
			t.Fatalf("I设置命令(%#q).Err = %v; want nil", "execabs-test", cmd.Cmd父类.Err)
		}
	})

	t.Run(pathVar+"=$OTHER", func(t *testing.T) {
		// 控制情况：如果PATH为空时查找返回ErrDot，那么我们知道PATH隐式包含“.”。
		//如果没有，那么我们不希望在这个测试中看到ErrDot（因为路径将是绝对的）。
		wantErrDot := false
		t.Setenv(pathVar, "")
		if found, err := I查找路径("execabs-test"); errors.Is(err, exec.ErrDot) {
			wantErrDot = true
		} else if err == nil {
			t.Fatalf(`with PATH='', I查找路径(%#q) = %#q; want non-nil error`, "execabs-test", found)
		}

		// 将PATH设置为包含一个显式目录，该目录包含一个完全独立的可执行文件，该可执行文件恰好与“.”中的可执行程序同名。
		//如果隐式包含“.”，则查找（非限定）可执行文件名称将返回ErrDot;
		//否则，“.”中的可执行文件应无效，查找应明确解析为PATH中的目录。

		dir := t.TempDir()
		executable := "execabs-test"
		if runtime.GOOS == "windows" {
			executable += ".exe"
		}
		if err := os.WriteFile(filepath.Join(dir, executable), []byte{1, 2, 3}, 0777); err != nil {
			t.Fatal(err)
		}
		t.Setenv(pathVar, dir+string(filepath.ListSeparator)+origPath)

		found, err := I查找路径("execabs-test")
		if wantErrDot {
			wantFound := filepath.Join(".", executable)
			if found != wantFound || !errors.Is(err, exec.ErrDot) {
				t.Fatalf(`I查找路径(%#q) = %#q, %v, want %#q, Is ErrDot`, "execabs-test", found, err, wantFound)
			}
		} else {
			wantFound := filepath.Join(dir, executable)
			if found != wantFound || err != nil {
				t.Fatalf(`I查找路径(%#q) = %#q, %v, want %#q, nil`, "execabs-test", found, err, wantFound)
			}
		}
	})
}
