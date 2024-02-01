// Copyright 2017 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build unix

package cmd类_test

import (
	"e.coding.net/gogit/go/gosdk/internal/testenv"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

func init() {
	registerHelperCommand("pwd", cmdPwd)
	registerHelperCommand("sleep", cmdSleep)
}

func cmdPwd(...string) {
	pwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println(pwd)
}

func cmdSleep(args ...string) {
	n, err := strconv.Atoi(args[0])
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	time.Sleep(time.Duration(n) * time.Second)
}

func TestCredentialNoSetGroups(t *testing.T) {
	if runtime.GOOS == "android" {
		maySkipHelperCommand("echo")
		t.Skip("unsupported on Android")
	}

	u, err := user.Current()
	if err != nil {
		t.Fatalf("error getting current user: %v", err)
	}

	uid, err := strconv.Atoi(u.Uid)
	if err != nil {
		t.Fatalf("error converting Uid=%s to integer: %v", u.Uid, err)
	}

	gid, err := strconv.Atoi(u.Gid)
	if err != nil {
		t.Fatalf("error converting Gid=%s to integer: %v", u.Gid, err)
	}

	//如果NoSetGroups为true，则不会调用setgroups，cmd.Run应成功
	cmd := helperCommand(t, "echo", "foo")
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Credential: &syscall.Credential{
			Uid:         uint32(uid),
			Gid:         uint32(gid),
			NoSetGroups: true,
		},
	}

	if err = cmd.I运行(); err != nil {
		t.Errorf("Failed to run command: %v", err)
	}
}

// For issue #19314: 确保SIGSTOP不会导致进程看起来已完成。
func TestWaitid(t *testing.T) {
	t.Parallel()

	cmd := helperCommand(t, "sleep", "3")
	if err := cmd.I运行_异步(); err != nil {
		t.Fatal(err)
	}

	// 这里的睡眠是不必要的，因为测试仍然应该通过，但它们有助于让我们更有可能测试孩子的预期状态。
	time.Sleep(100 * time.Millisecond)

	if err := cmd.Process.Signal(syscall.SIGSTOP); err != nil {
		cmd.Process.Kill()
		t.Fatal(err)
	}

	ch := make(chan error)
	go func() {
		ch <- cmd.I等待运行完成()
	}()

	time.Sleep(100 * time.Millisecond)

	if err := cmd.Process.Signal(syscall.SIGCONT); err != nil {
		t.Error(err)
		syscall.Kill(cmd.Process.Pid, syscall.SIGCONT)
	}

	cmd.Process.Kill()

	<-ch
}

// https://go.dev/issue/50599: 如果未显式设置Env，则设置Dir应隐式将PWD更新为正确的路径，并且Environ应列出更新的值。
func TestImplicitPWD(t *testing.T) {
	t.Parallel()

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		name string
		dir  string
		want string
	}{
		{"empty", "", cwd},
		{"dot", ".", cwd},
		{"dotdot", "..", filepath.Dir(cwd)},
		{"PWD", cwd, cwd},
		{"PWDdotdot", cwd + string(filepath.Separator) + "..", filepath.Dir(cwd)},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			cmd := helperCommand(t, "pwd")
			if cmd.Env != nil {
				t.Fatalf("test requires helperCommand not to set Env field")
			}
			cmd.Dir = tc.dir

			var pwds []string
			for _, kv := range cmd.I取环境变量数组() {
				if strings.HasPrefix(kv, "PWD=") {
					pwds = append(pwds, strings.TrimPrefix(kv, "PWD="))
				}
			}

			wantPWDs := []string{tc.want}
			if tc.dir == "" {
				if _, ok := os.LookupEnv("PWD"); !ok {
					wantPWDs = nil
				}
			}
			if !reflect.DeepEqual(pwds, wantPWDs) {
				t.Errorf("PWD entries in cmd.I取环境变量数组():\n\t%s\nwant:\n\t%s", strings.Join(pwds, "\n\t"), strings.Join(wantPWDs, "\n\t"))
			}

			cmd.Stderr = new(strings.Builder)
			out, err := cmd.I运行_带返回值()
			if err != nil {
				t.Fatalf("%v:\n%s", err, cmd.Stderr)
			}
			got := strings.Trim(string(out), "\r\n")
			t.Logf("in\n\t%s\n`pwd` reported\n\t%s", tc.dir, got)
			if got != tc.want {
				t.Errorf("want\n\t%s", tc.want)
			}
		})
	}
}

// 但是，如果显式设置了cmd.Env，则设置Dir不应覆盖它。
// （这将检查https://go.dev/issue/50599 的实现不会中断可能已显式匹配PWD变量的现有用户。）
func TestExplicitPWD(t *testing.T) {
	maySkipHelperCommand("pwd")
	testenv.MustHaveSymlink(t)

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	link := filepath.Join(t.TempDir(), "link")
	if err := os.Symlink(cwd, link); err != nil {
		t.Fatal(err)
	}

	// 现在link是cwd的另一个同样有效的名称。如果我们将Dir设置为一个，将PWD设置为另一个，则子流程应报告PWD版本。
	cases := []struct {
		name string
		dir  string
		pwd  string
	}{
		{name: "original PWD", pwd: cwd},
		{name: "link PWD", pwd: link},
		{name: "in link with original PWD", dir: link, pwd: cwd},
		{name: "in dir with link PWD", dir: cwd, pwd: link},
		// 理想情况下，我们也希望测试如果我们将PWD设置为完全虚假的值（或空字符串）会发生什么情况，
		//但这样我们就不知道子流程应该实际生成什么输出：cwd本身可能包含从测试环境中的PWD值保存的符号链接。
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			cmd := helperCommand(t, "pwd")
			// 这故意与设置cmd.Dir然后调用cmd.Environ的通常顺序相反。
			//这里，我们希望PWD不匹配cmd.Dir，因此我们不关心cmd.Dir是否反映在cmd.Envron中。
			cmd.Env = append(cmd.I取环境变量数组(), "PWD="+tc.pwd)
			cmd.Dir = tc.dir

			var pwds []string
			for _, kv := range cmd.I取环境变量数组() {
				if strings.HasPrefix(kv, "PWD=") {
					pwds = append(pwds, strings.TrimPrefix(kv, "PWD="))
				}
			}

			wantPWDs := []string{tc.pwd}
			if !reflect.DeepEqual(pwds, wantPWDs) {
				t.Errorf("PWD entries in cmd.I取环境变量数组():\n\t%s\nwant:\n\t%s", strings.Join(pwds, "\n\t"), strings.Join(wantPWDs, "\n\t"))
			}

			cmd.Stderr = new(strings.Builder)
			out, err := cmd.I运行_带返回值()
			if err != nil {
				t.Fatalf("%v:\n%s", err, cmd.Stderr)
			}
			got := strings.Trim(string(out), "\r\n")
			t.Logf("in\n\t%s\nwith PWD=%s\nsubprocess os.Getwd() reported\n\t%s", tc.dir, tc.pwd, got)
			if got != tc.pwd {
				t.Errorf("want\n\t%s", tc.pwd)
			}
		})
	}
}
