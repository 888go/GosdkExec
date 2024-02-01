// Copyright 2021 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build windows

package cmd类_test

import (
	"e.coding.net/gogit/go/gosdk/core/os_exec_cn"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"testing"
)

func init() {
	registerHelperCommand("pipehandle", cmdPipeHandle)
}

func cmdPipeHandle(args ...string) {
	handle, _ := strconv.ParseUint(args[0], 16, 64)
	pipe := os.NewFile(uintptr(handle), "")
	_, err := fmt.Fprint(pipe, args[1])
	if err != nil {
		fmt.Fprintf(os.Stderr, "writing to pipe failed: %v\n", err)
		os.Exit(1)
	}
	pipe.Close()
}

func TestPipePassing(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Error(err)
	}
	const marker = "arrakis, dune, desert planet"
	childProc := helperCommand(t, "pipehandle", strconv.FormatUint(uint64(w.Fd()), 16), marker)
	childProc.Cmd父类.SysProcAttr = &syscall.SysProcAttr{AdditionalInheritedHandles: []syscall.Handle{syscall.Handle(w.Fd())}}
	err = childProc.I运行_异步()
	if err != nil {
		t.Error(err)
	}
	w.Close()
	response, err := io.ReadAll(r)
	if err != nil {
		t.Error(err)
	}
	r.Close()
	if string(response) != marker {
		t.Errorf("got %q; want %q", string(response), marker)
	}
	err = childProc.I等待运行完成()
	if err != nil {
		t.Error(err)
	}
}

func TestNoInheritHandles(t *testing.T) {
	cmd := cmd类.I设置命令("cmd", "/c exit 88")
	cmd.Cmd父类.SysProcAttr = &syscall.SysProcAttr{NoInheritHandles: true}
	err := cmd.I运行()
	exitError, ok := err.(*exec.ExitError)
	if !ok {
		t.Fatalf("got error %v; want ExitError", err)
	}
	if exitError.ExitCode() != 88 {
		t.Fatalf("got exit code %d; want 88", exitError.ExitCode())
	}
}

// 启动子进程，而不使用以父进程的SYSTEMROOT副本开始的用户代码。
// (See issue 25210.)
func TestChildCriticalEnv(t *testing.T) {
	cmd := helperCommand(t, "echoenv", "SYSTEMROOT")

	// Explicitly remove SYSTEMROOT from the command's environment.
	var env []string
	for _, kv := range cmd.I取环境变量数组() {
		k, _, ok := strings.Cut(kv, "=")
		if !ok || !strings.EqualFold(k, "SYSTEMROOT") {
			env = append(env, kv)
		}
	}
	cmd.Cmd父类.Env = env

	out, err := cmd.I运行_带组合返回值()
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(out)) == "" {
		t.Error("no SYSTEMROOT found")
	}
}
