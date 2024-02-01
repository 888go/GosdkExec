// Copyright 2020 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build ignore

// 这是一个测试程序，它验证它可以从描述符3中读取，并且没有其他描述符打开。
// 这不是通过TestHelperProcess和GO_EXEC_TEST_，
// 因为C库可以在背后打开文件描述符，从而混淆测试。见第25628期。
package main

import (
	"e.coding.net/gogit/go/gosdk/core/os_exec_cn"
	"e.coding.net/gogit/go/gosdk/internal/poll"
	"fmt"
	"io"
	"os"
	"os/exec/internal/fdtest"
	"runtime"
	"strings"
)

func main() {
	fd3 := os.NewFile(3, "fd3")
	defer fd3.Close()

	bs, err := io.ReadAll(fd3)
	if err != nil {
		fmt.Printf("ReadAll from fd 3: %v\n", err)
		os.Exit(1)
	}

	// 现在确认没有其他打开的fds。
	// stdin == 0
	// stdout == 1
	// stderr == 2
	// descriptor from parent == 3
	// 除网络轮询器使用的任何描述符外，所有描述符4及以上都应可用。
	for fd := uintptr(4); fd <= 100; fd++ {
		if poll.IsPollDescriptor(fd) {
			continue
		}

		if !fdtest.Exists(fd) {
			continue
		}

		fmt.Printf("泄漏的父文件。fdtest.Exists（%d）为true，但为false \n", fd)

		fdfile := fmt.Sprintf("/proc/self/fd/%d", fd)
		link, err := os.Readlink(fdfile)
		fmt.Printf("readlink(%q) = %q, %v\n", fdfile, link, err)

		var args []string
		switch runtime.GOOS {
		case "plan9":
			args = []string{fmt.Sprintf("/proc/%d/fd", os.Getpid())}
		case "aix", "solaris", "illumos":
			args = []string{fmt.Sprint(os.Getpid())}
		default:
			args = []string{"-p", fmt.Sprint(os.Getpid())}
		}

		// Determine which command to use to display open files.
		ofcmd := "lsof"
		switch runtime.GOOS {
		case "dragonfly", "freebsd", "netbsd", "openbsd":
			ofcmd = "fstat"
		case "plan9":
			ofcmd = "/bin/cat"
		case "aix":
			ofcmd = "procfiles"
		case "solaris", "illumos":
			ofcmd = "pfiles"
		}

		cmd := cmd类.I设置命令(ofcmd, args...)
		out, err := cmd.I运行_带组合返回值()
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s failed: %v\n", strings.Join(cmd.Args, " "), err)
		}
		fmt.Printf("%s", out)
		os.Exit(1)
	}

	os.Stdout.Write(bs)
}
