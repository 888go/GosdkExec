// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Use an external test to avoid os/exec -> net/http -> crypto/x509 -> os/exec
// circular dependency on non-cgo darwin.

package cmd类_test

import (
	"bufio"
	"bytes"
	"context"
	"e.coding.net/gogit/go/gosdk/core/os_exec_cn"
	"e.coding.net/gogit/go/gosdk/core/os_exec_cn/internal/fdtest"
	"e.coding.net/gogit/go/gosdk/internal/poll"
	"e.coding.net/gogit/go/gosdk/internal/testenv"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

// haveUnexpectedFDs 在初始化时设置，以报告在程序启动时是否打开了任何文件描述符。
var haveUnexpectedFDs bool

func init() {
	if os.Getenv("GO_EXEC_TEST_PID") != "" {
		return
	}
	if runtime.GOOS == "windows" {
		return
	}
	for fd := uintptr(3); fd <= 100; fd++ {
		if poll.IsPollDescriptor(fd) {
			continue
		}

		if fdtest.Exists(fd) {
			haveUnexpectedFDs = true
			return
		}
	}
}

// TestMain允许测试二进制文件模拟许多其他二进制文件，其中一些可能会操作os.Stdin、os.Stdout,
// 和或os.Stderr（因此不能作为普通测试函数运行，因为测试包monkey在运行测试之前会修补这些变量）。
func TestMain(m *testing.M) {
	flag.Parse()

	pid := os.Getpid()
	if os.Getenv("GO_EXEC_TEST_PID") == "" {
		os.Setenv("GO_EXEC_TEST_PID", strconv.Itoa(pid))

		code := m.Run()
		if code == 0 && flag.Lookup("test.run").Value.String() == "" && flag.Lookup("test.list").Value.String() == "" {
			for cmd := range helperCommands {
				if _, ok := helperCommandUsed.Load(cmd); !ok {
					fmt.Fprintf(os.Stderr, "helper command unused: %q\n", cmd)
					code = 1
				}
			}
		}
		os.Exit(code)
	}

	args := flag.Args()
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "No command\n")
		os.Exit(2)
	}

	cmd, args := args[0], args[1:]
	f, ok := helperCommands[cmd]
	if !ok {
		fmt.Fprintf(os.Stderr, "Unknown command %q\n", cmd)
		os.Exit(2)
	}
	f(args...)
	os.Exit(0)
}

// registerHelperCommand 注册测试进程可以模拟的命令。命令应在使用它的同一源文件中注册。
// 如果所有测试都运行并通过，则必须使用所有注册的命令。
// （如果随着时间的推移删除或重构测试，这可以防止过时的命令堆积。）
func registerHelperCommand(name string, f func(...string)) {
	if helperCommands[name] != nil {
		panic("已注册重复命令: " + name)
	}
	helperCommands[name] = f
}

// maySkipHelperCommand 记录调用了使用命名助手命令的测试，
// 但在实际调用helperCommand之前可能会在测试上调用Skip。
func maySkipHelperCommand(name string) {
	helperCommandUsed.Store(name, true)
}

// helperCommand 返回将运行命名助手命令的exec.Cmd。
func helperCommand(t *testing.T, name string, args ...string) *cmd类.Cmd {
	t.Helper()
	return helperCommandContext(t, nil, name, args...)
}

// helperCommandContext 类似于helperCommand，但也接受运行命令的上下文。
func helperCommandContext(t *testing.T, ctx context.Context, name string, args ...string) (cmd *cmd类.Cmd) {
	helperCommandUsed.LoadOrStore(name, true)

	t.Helper()
	testenv.MustHaveExec(t)

	cs := append([]string{name}, args...)
	if ctx != nil {
		cmd = cmd类.I设置命令_上下文(ctx, exePath(t), cs...)
	} else {
		cmd = cmd类.I设置命令(exePath(t), cs...)
	}
	return cmd
}

// exePath 返回正在运行的可执行文件的路径。
func exePath(t testing.TB) string {
	exeOnce.Do(func() {
		// 如果调用方修改cmd.Dir，请使用os.Executable而不是os.Args[0]：
		//如果测试二进制文件像"./exec.test"一样被调用，那么它不应该错误地失败。
		exeOnce.path, exeOnce.err = os.Executable()
	})

	if exeOnce.err != nil {
		if t == nil {
			panic(exeOnce.err)
		}
		t.Fatal(exeOnce.err)
	}

	return exeOnce.path
}

var exeOnce struct {
	path string
	err  error
	sync.Once
}

var helperCommandUsed sync.Map

var helperCommands = map[string]func(...string){
	"echo":               cmdEcho,
	"echoenv":            cmdEchoEnv,
	"cat":                cmdCat,
	"pipetest":           cmdPipeTest,
	"stdinClose":         cmdStdinClose,
	"exit":               cmdExit,
	"describefiles":      cmdDescribeFiles,
	"extraFilesAndPipes": cmdExtraFilesAndPipes,
	"stderrfail":         cmdStderrFail,
	"yes":                cmdYes,
}

func cmdEcho(args ...string) {
	iargs := []any{}
	for _, s := range args {
		iargs = append(iargs, s)
	}
	fmt.Println(iargs...)
}

func cmdEchoEnv(args ...string) {
	for _, s := range args {
		fmt.Println(os.Getenv(s))
	}
}

func cmdCat(args ...string) {
	if len(args) == 0 {
		io.Copy(os.Stdout, os.Stdin)
		return
	}
	exit := 0
	for _, fn := range args {
		f, err := os.Open(fn)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			exit = 2
		} else {
			defer f.Close()
			io.Copy(os.Stdout, f)
		}
	}
	os.Exit(exit)
}

func cmdPipeTest(...string) {
	bufr := bufio.NewReader(os.Stdin)
	for {
		line, _, err := bufr.ReadLine()
		if err == io.EOF {
			break
		} else if err != nil {
			os.Exit(1)
		}
		if bytes.HasPrefix(line, []byte("O:")) {
			os.Stdout.Write(line)
			os.Stdout.Write([]byte{'\n'})
		} else if bytes.HasPrefix(line, []byte("E:")) {
			os.Stderr.Write(line)
			os.Stderr.Write([]byte{'\n'})
		} else {
			os.Exit(1)
		}
	}
}

func cmdStdinClose(...string) {
	b, err := io.ReadAll(os.Stdin)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	if s := string(b); s != stdinCloseTestString {
		fmt.Fprintf(os.Stderr, "Error: Read %q, want %q", s, stdinCloseTestString)
		os.Exit(1)
	}
}

func cmdExit(args ...string) {
	n, _ := strconv.Atoi(args[0])
	os.Exit(n)
}

func cmdDescribeFiles(args ...string) {
	f := os.NewFile(3, fmt.Sprintf("fd3"))
	ln, err := net.FileListener(f)
	if err == nil {
		fmt.Printf("fd3: listener %s\n", ln.Addr())
		ln.Close()
	}
}

func cmdExtraFilesAndPipes(args ...string) {
	n, _ := strconv.Atoi(args[0])
	pipes := make([]*os.File, n)
	for i := 0; i < n; i++ {
		pipes[i] = os.NewFile(uintptr(3+i), strconv.Itoa(i))
	}
	response := ""
	for i, r := range pipes {
		buf := make([]byte, 10)
		n, err := r.Read(buf)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Child: 读取错误：管道%d上的%v\n", err, i)
			os.Exit(1)
		}
		response = response + string(buf[:n])
	}
	fmt.Fprintf(os.Stderr, "child: %s", response)
}

func cmdStderrFail(...string) {
	fmt.Fprintf(os.Stderr, "some stderr text\n")
	os.Exit(1)
}

func cmdYes(args ...string) {
	if len(args) == 0 {
		args = []string{"y"}
	}
	s := strings.Join(args, " ") + "\n"
	for {
		_, err := os.Stdout.WriteString(s)
		if err != nil {
			os.Exit(1)
		}
	}
}

func TestEcho(t *testing.T) {
	bs, err := helperCommand(t, "echo", "foo bar", "baz").I运行_带返回值()
	if err != nil {
		t.Errorf("echo: %v", err)
	}
	if g, e := string(bs), "foo bar baz\n"; g != e {
		t.Errorf("echo: want %q, got %q", e, g)
	}
}

func TestCommandRelativeName(t *testing.T) {
	cmd := helperCommand(t, "echo", "foo")

	// 将我们自己的二进制文件作为父目录的相对路径 (e.g. "_test/exec.test") 运行。
	base := filepath.Base(os.Args[0]) // "exec.test"
	dir := filepath.Dir(os.Args[0])   // "/tmp/go-buildNNNN/os/exec/_test"
	if dir == "." {
		t.Skip("skipping; running test at root somehow")
	}
	parentDir := filepath.Dir(dir) // "/tmp/go-buildNNNN/os/exec"
	dirBase := filepath.Base(dir)  // "_test"
	if dirBase == "." {
		t.Skipf("skipping; unexpected shallow dir of %q", dir)
	}

	cmd.Cmd父类.Path = filepath.Join(dirBase, base)
	cmd.Cmd父类.Dir = parentDir

	out, err := cmd.I运行_带返回值()
	if err != nil {
		t.Errorf("echo: %v", err)
	}
	if g, e := string(out), "foo\n"; g != e {
		t.Errorf("echo: want %q, got %q", e, g)
	}
}

func TestCatStdin(t *testing.T) {
	// Cat，测试标准输入和标准输出。
	input := "Input string\nLine 2"
	p := helperCommand(t, "cat")
	p.Cmd父类.Stdin = strings.NewReader(input)
	bs, err := p.I运行_带返回值()
	if err != nil {
		t.Errorf("cat: %v", err)
	}
	s := string(bs)
	if s != input {
		t.Errorf("cat: want %q, got %q", input, s)
	}
}

func TestEchoFileRace(t *testing.T) {
	cmd := helperCommand(t, "echo")
	stdin, err := cmd.I取Stdin管道()
	if err != nil {
		t.Fatalf("I取Stdin管道: %v", err)
	}
	if err := cmd.I运行_异步(); err != nil {
		t.Fatalf("I运行_异步: %v", err)
	}
	wrote := make(chan bool)
	go func() {
		defer close(wrote)
		fmt.Fprint(stdin, "echo\n")
	}()
	if err := cmd.I等待运行完成(); err != nil {
		t.Fatalf("I等待运行完成: %v", err)
	}
	<-wrote
}

func TestCatGoodAndBadFile(t *testing.T) {
	// 测试组合输出和误差值。
	bs, err := helperCommand(t, "cat", "/bogus/file.foo", "exec_test.go").I运行_带组合返回值()
	if _, ok := err.(*exec.ExitError); !ok {
		t.Errorf("expected *exec.ExitError from cat combined; got %T: %v", err, err)
	}
	errLine, body, ok := strings.Cut(string(bs), "\n")
	if !ok {
		t.Fatalf("expected two lines from cat; got %q", bs)
	}
	if !strings.HasPrefix(errLine, "Error: open /bogus/file.foo") {
		t.Errorf("expected stderr to complain about file; got %q", errLine)
	}
	if !strings.Contains(body, "func TestCatGoodAndBadFile(t *testing.T)") {
		t.Errorf("expected test code; got %q (len %d)", body, len(body))
	}
}

func TestNoExistExecutable(t *testing.T) {
	//无法运行不存在的可执行文件
	err := cmd类.I设置命令("/no-exist-executable").I运行()
	if err == nil {
		t.Error("expected error from /no-exist-executable")
	}
}

func TestExitStatus(t *testing.T) {
	// 测试退出值是否正确返回
	cmd := helperCommand(t, "exit", "42")
	err := cmd.I运行()
	want := "exit status 42"
	switch runtime.GOOS {
	case "plan9":
		want = fmt.Sprintf("exit status: '%s %d: 42'", filepath.Base(cmd.Cmd父类.Path), cmd.Cmd父类.ProcessState.Pid())
	}
	if werr, ok := err.(*exec.ExitError); ok {
		if s := werr.Error(); s != want {
			t.Errorf("from exit 42 got exit %q, want %q", s, want)
		}
	} else {
		t.Fatalf("expected *exec.ExitError from exit 42; got %T: %v", err, err)
	}
}

func TestExitCode(t *testing.T) {
	// 测试退出代码是否正确返回
	cmd := helperCommand(t, "exit", "42")
	cmd.I运行()
	want := 42
	if runtime.GOOS == "plan9" {
		want = 1
	}
	got := cmd.Cmd父类.ProcessState.ExitCode()
	if want != got {
		t.Errorf("ExitCode got %d, want %d", got, want)
	}

	cmd = helperCommand(t, "/no-exist-executable")
	cmd.I运行()
	want = 2
	if runtime.GOOS == "plan9" {
		want = 1
	}
	got = cmd.Cmd父类.ProcessState.ExitCode()
	if want != got {
		t.Errorf("ExitCode got %d, want %d", got, want)
	}

	cmd = helperCommand(t, "exit", "255")
	cmd.I运行()
	want = 255
	if runtime.GOOS == "plan9" {
		want = 1
	}
	got = cmd.Cmd父类.ProcessState.ExitCode()
	if want != got {
		t.Errorf("ExitCode got %d, want %d", got, want)
	}

	cmd = helperCommand(t, "cat")
	cmd.I运行()
	want = 0
	got = cmd.Cmd父类.ProcessState.ExitCode()
	if want != got {
		t.Errorf("ExitCode got %d, want %d", got, want)
	}

	// Test when command does not call I运行().
	cmd = helperCommand(t, "cat")
	want = -1
	got = cmd.Cmd父类.ProcessState.ExitCode()
	if want != got {
		t.Errorf("ExitCode got %d, want %d", got, want)
	}
}

func TestPipes(t *testing.T) {
	check := func(what string, err error) {
		if err != nil {
			t.Fatalf("%s: %v", what, err)
		}
	}
	// Cat，测试标准输入和标准输出。
	c := helperCommand(t, "pipetest")
	stdin, err := c.I取Stdin管道()
	check("I取Stdin管道", err)
	stdout, err := c.I取标准管道()
	check("I取标准管道", err)
	stderr, err := c.I取Stderr管道()
	check("I取Stderr管道", err)

	outbr := bufio.NewReader(stdout)
	errbr := bufio.NewReader(stderr)
	line := func(what string, br *bufio.Reader) string {
		line, _, err := br.ReadLine()
		if err != nil {
			t.Fatalf("%s: %v", what, err)
		}
		return string(line)
	}

	err = c.I运行_异步()
	check("I运行_异步", err)

	_, err = stdin.Write([]byte("O:I am output\n"))
	check("first stdin Write", err)
	if g, e := line("first output line", outbr), "O:I am output"; g != e {
		t.Errorf("got %q, want %q", g, e)
	}

	_, err = stdin.Write([]byte("E:I am error\n"))
	check("second stdin Write", err)
	if g, e := line("first error line", errbr), "E:I am error"; g != e {
		t.Errorf("got %q, want %q", g, e)
	}

	_, err = stdin.Write([]byte("O:I am output2\n"))
	check("third stdin Write 3", err)
	if g, e := line("second output line", outbr), "O:I am output2"; g != e {
		t.Errorf("got %q, want %q", g, e)
	}

	stdin.Close()
	err = c.I等待运行完成()
	check("I等待运行完成", err)
}

const stdinCloseTestString = "Some test string."

// Issue 6270.
func TestStdinClose(t *testing.T) {
	check := func(what string, err error) {
		if err != nil {
			t.Fatalf("%s: %v", what, err)
		}
	}
	cmd := helperCommand(t, "stdinClose")
	stdin, err := cmd.I取Stdin管道()
	check("I取Stdin管道", err)
	// Check that we can access methods of the underlying os.File.`
	if _, ok := stdin.(interface {
		Fd() uintptr
	}); !ok {
		t.Error("can't access methods of underlying *os.File")
	}
	check("I运行_异步", cmd.I运行_异步())
	go func() {
		_, err := io.Copy(stdin, strings.NewReader(stdinCloseTestString))
		check("Copy", err)
		// Before the fix, this next line would race with cmd.I等待运行完成.
		check("Close", stdin.Close())
	}()
	check("I等待运行完成", cmd.I等待运行完成())
}

// 问题 17647.
// 过去，当在竞赛检测器下运行时，上面的TestStdinClose会失败。
// 该测试是TestStdinClose的变体，在比赛检测器下运行时也会失败。
// 此测试由cmd/dist在竞赛检测器下运行，以验证竞赛检测器不再报告任何问题。
func TestStdinCloseRace(t *testing.T) {
	cmd := helperCommand(t, "stdinClose")
	stdin, err := cmd.I取Stdin管道()
	if err != nil {
		t.Fatalf("I取Stdin管道: %v", err)
	}
	if err := cmd.I运行_异步(); err != nil {
		t.Fatalf("I运行_异步: %v", err)
	}
	go func() {
		// 我们不检查Kill的错误返回。进程可能已经退出，在这种情况下Kill将返回错误“进程已经完成”。
		//此测试的目的是查看竞赛检测器是否报告错误；这次杀戮成功与否并不重要。
		cmd.Cmd父类.Process.Kill()
	}()
	go func() {
		// Send the wrong string, so that the child fails even
		// if the other goroutine doesn't manage to kill it first.
		// This test is to check that the race detector does not
		// falsely report an error, so it doesn't matter how the
		// child process fails.
		io.Copy(stdin, strings.NewReader("unexpected string"))
		if err := stdin.Close(); err != nil {
			t.Errorf("stdin.Close: %v", err)
		}
	}()
	if err := cmd.I等待运行完成(); err == nil {
		t.Fatalf("I等待运行完成: succeeded unexpectedly")
	}
}

// Issue 5071
func TestPipeLookPathLeak(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("we don't currently suppore counting open handles on windows")
	}

	openFDs := func() []uintptr {
		var fds []uintptr
		for i := uintptr(0); i < 100; i++ {
			if fdtest.Exists(i) {
				fds = append(fds, i)
			}
		}
		return fds
	}

	want := openFDs()
	for i := 0; i < 6; i++ {
		cmd := cmd类.I设置命令("something-that-does-not-exist-executable")
		cmd.I取标准管道()
		cmd.I取Stderr管道()
		cmd.I取Stdin管道()
		if err := cmd.I运行(); err == nil {
			t.Fatal("unexpected success")
		}
	}
	got := openFDs()
	if !reflect.DeepEqual(got, want) {
		t.Errorf("set of open file descriptors changed: got %v, want %v", got, want)
	}
}

func TestExtraFilesFDShuffle(t *testing.T) {
	maySkipHelperCommand("extraFilesAndPipes")
	testenv.SkipFlaky(t, 5780)
	switch runtime.GOOS {
	case "windows":
		t.Skip("no operating system support; skipping")
	}

	// syscall.StartProcess maps all the FDs passed to it in
	// ProcAttr.Files (the concatenation of stdin,stdout,stderr and
	// ExtraFiles) into consecutive FDs in the child, that is:
	// Files{11, 12, 6, 7, 9, 3} should result in the file
	// represented by FD 11 in the parent being made available as 0
	// in the child, 12 as 1, etc.
	//
	// We want to test that FDs in the child do not get overwritten
	// by one another as this shuffle occurs. The original implementation
	// was buggy in that in some data dependent cases it would overwrite
	// stderr in the child with one of the ExtraFile members.
	// Testing for this case is difficult because it relies on using
	// the same FD values as that case. In particular, an FD of 3
	// must be at an index of 4 or higher in ProcAttr.Files and
	// the FD of the write end of the Stderr pipe (as obtained by
	// I取Stderr管道()) must be the same as the size of ProcAttr.Files;
	// therefore we test that the read end of this pipe (which is what
	// is returned to the parent by I取Stderr管道() being one less than
	// the size of ProcAttr.Files, i.e. 3+len(cmd.ExtraFiles).
	//
	// Moving this test case around within the overall tests may
	// affect the FDs obtained and hence the checks to catch these cases.
	npipes := 2
	c := helperCommand(t, "extraFilesAndPipes", strconv.Itoa(npipes+1))
	rd, wr, _ := os.Pipe()
	defer rd.Close()
	if rd.Fd() != 3 {
		t.Errorf("bad test value for test pipe: fd %d", rd.Fd())
	}
	stderr, _ := c.I取Stderr管道()
	wr.WriteString("_LAST")
	wr.Close()

	pipes := make([]struct {
		r, w *os.File
	}, npipes)
	data := []string{"a", "b"}

	for i := 0; i < npipes; i++ {
		r, w, err := os.Pipe()
		if err != nil {
			t.Fatalf("unexpected error creating pipe: %s", err)
		}
		pipes[i].r = r
		pipes[i].w = w
		w.WriteString(data[i])
		c.Cmd父类.ExtraFiles = append(c.Cmd父类.ExtraFiles, pipes[i].r)
		defer func() {
			r.Close()
			w.Close()
		}()
	}
	// 把fd 3放在末尾。
	c.Cmd父类.ExtraFiles = append(c.Cmd父类.ExtraFiles, rd)

	stderrFd := int(stderr.(*os.File).Fd())
	if stderrFd != ((len(c.Cmd父类.ExtraFiles) + 3) - 1) {
		t.Errorf("bad test value for stderr pipe")
	}

	expected := "child: " + strings.Join(data, "") + "_LAST"

	err := c.I运行_异步()
	if err != nil {
		t.Fatalf("I运行: %v", err)
	}

	buf := make([]byte, 512)
	n, err := stderr.Read(buf)
	if err != nil {
		t.Errorf("Read: %s", err)
	} else {
		if m := string(buf[:n]); m != expected {
			t.Errorf("Read: '%s' not '%s'", m, expected)
		}
	}
	c.I等待运行完成()
}

func TestExtraFiles(t *testing.T) {
	if haveUnexpectedFDs {
		// 这个测试的目的是确保我们打开的所有描述符都标记为close-on-exec。
		//如果haveUnexpectedFDs为true，那么在我们开始测试时还有其他描述符打开，因此这些描述符在exec上显然不接近，
		//它们会混淆测试。我们可以修改测试以期望这些描述符保持开放，但由于我们不知道它们来自何处或它们在做什么，这似乎很脆弱。
		//例如，也许出于某种原因，它们来自这个系统上的启动代码。此外，该测试不是系统特定的；
		//只要大多数系统不跳过测试，我们仍将测试我们关心的内容。
		t.Skip("skipping test because test was run with FDs open")
	}

	testenv.MustHaveExec(t)
	testenv.MustHaveGoBuild(t)

	//此测试在禁用cgo的情况下运行。外部链接需要cgo，所以如果需要外部链接，它就不起作用。
	testenv.MustInternalLink(t)

	if runtime.GOOS == "windows" {
		t.Skipf("skipping test on %q", runtime.GOOS)
	}

	// 强制网络使用，以验证epoll（或其他）fd不会泄漏给孩子，
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	// 确保复制的fds不会泄露给孩子。
	f, err := ln.(*net.TCPListener).File()
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	ln2, err := net.FileListener(f)
	if err != nil {
		t.Fatal(err)
	}
	defer ln2.Close()

	// 强制加载TLS根证书（这可能涉及cgo），以确保没有任何潜在的C代码泄漏fds。
	ts := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	// quiet expected TLS handshake error "remote error: bad certificate"
	ts.Config.ErrorLog = log.New(io.Discard, "", 0)
	ts.StartTLS()
	defer ts.Close()
	_, err = http.Get(ts.URL)
	if err == nil {
		t.Errorf("success trying to fetch %s; want an error", ts.URL)
	}

	tf, err := os.CreateTemp("", "")
	if err != nil {
		t.Fatalf("TempFile: %v", err)
	}
	defer os.Remove(tf.Name())
	defer tf.Close()

	const text = "Hello, fd 3!"
	_, err = tf.Write([]byte(text))
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
	_, err = tf.Seek(0, io.SeekStart)
	if err != nil {
		t.Fatalf("Seek: %v", err)
	}

	tempdir := t.TempDir()
	exe := filepath.Join(tempdir, "read3.exe")

	c := cmd类.I设置命令(testenv.GoToolPath(t), "build", "-o", exe, "read3.go")
	// 在没有cgo的情况下构建测试，这样C库函数就不会意外地打开描述符。见第25628期。
	c.Cmd父类.Env = append(os.Environ(), "CGO_ENABLED=0")
	if output, err := c.I运行_带组合返回值(); err != nil {
		t.Logf("go build -o %s read3.go\n%s", exe, output)
		t.Fatalf("go build failed: %v", err)
	}

	// 即使程序挂起，也要使用截止日期来尝试获得一些输出。
	ctx := context.Background()
	if deadline, ok := t.Deadline(); ok {
		// 留出20%的宽限期来刷新输出，这在linux386构建器上可能很大，因为我们正在strace下运行子进程。
		deadline = deadline.Add(-time.Until(deadline) / 5)

		var cancel context.CancelFunc
		ctx, cancel = context.WithDeadline(ctx, deadline)
		defer cancel()
	}

	c = cmd类.I设置命令_上下文(ctx, exe)
	var stdout, stderr bytes.Buffer
	c.Cmd父类.Stdout = &stdout
	c.Cmd父类.Stderr = &stderr
	c.Cmd父类.ExtraFiles = []*os.File{tf}
	if runtime.GOOS == "illumos" {
		// illumos中的一些设施是通过libc访问proc实现的；这样的访问可以短暂地占用低编号fd。
		//如果这与检查泄漏描述符的测试同时发生，则检查可能会变得混乱，并报告一个虚假的泄漏描述符。（更多详细分析请参见第42431期。）
		//
		// 尝试限制在子进程中使用其他线程，以使此测试不那么脆弱：
		c.Cmd父类.Env = append(os.Environ(), "GOMAXPROCS=1")
	}
	err = c.I运行()
	if err != nil {
		t.Fatalf("I运行: %v\n--- stdout:\n%s--- stderr:\n%s", err, stdout.Bytes(), stderr.Bytes())
	}
	if stdout.String() != text {
		t.Errorf("got stdout %q, stderr %q; want %q on stdout", stdout.String(), stderr.String(), text)
	}
}

func TestExtraFilesRace(t *testing.T) {
	if runtime.GOOS == "windows" {
		maySkipHelperCommand("describefiles")
		t.Skip("no operating system support; skipping")
	}
	listen := func() net.Listener {
		ln, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatal(err)
		}
		return ln
	}
	listenerFile := func(ln net.Listener) *os.File {
		f, err := ln.(*net.TCPListener).File()
		if err != nil {
			t.Fatal(err)
		}
		return f
	}
	runCommand := func(c *cmd类.Cmd, out chan<- string) {
		bout, err := c.I运行_带组合返回值()
		if err != nil {
			out <- "ERROR:" + err.Error()
		} else {
			out <- string(bout)
		}
	}

	for i := 0; i < 10; i++ {
		if testing.Short() && i >= 3 {
			break
		}
		la := listen()
		ca := helperCommand(t, "describefiles")
		ca.Cmd父类.ExtraFiles = []*os.File{listenerFile(la)}
		lb := listen()
		cb := helperCommand(t, "describefiles")
		cb.Cmd父类.ExtraFiles = []*os.File{listenerFile(lb)}
		ares := make(chan string)
		bres := make(chan string)
		go runCommand(ca, ares)
		go runCommand(cb, bres)
		if got, want := <-ares, fmt.Sprintf("fd3: listener %s\n", la.Addr()); got != want {
			t.Errorf("iteration %d, process A got:\n%s\nwant:\n%s\n", i, got, want)
		}
		if got, want := <-bres, fmt.Sprintf("fd3: listener %s\n", lb.Addr()); got != want {
			t.Errorf("iteration %d, process B got:\n%s\nwant:\n%s\n", i, got, want)
		}
		la.Close()
		lb.Close()
		for _, f := range ca.Cmd父类.ExtraFiles {
			f.Close()
		}
		for _, f := range cb.Cmd父类.ExtraFiles {
			f.Close()
		}

	}
}

type delayedInfiniteReader struct{}

func (delayedInfiniteReader) Read(b []byte) (int, error) {
	time.Sleep(100 * time.Millisecond)
	for i := range b {
		b[i] = 'x'
	}
	return len(b), nil
}

// Issue 9173: ignore stdin pipe writes if the program completes successfully.
func TestIgnorePipeErrorOnSuccess(t *testing.T) {
	testWith := func(r io.Reader) func(*testing.T) {
		return func(t *testing.T) {
			cmd := helperCommand(t, "echo", "foo")
			var out bytes.Buffer
			cmd.Cmd父类.Stdin = r
			cmd.Cmd父类.Stdout = &out
			if err := cmd.I运行(); err != nil {
				t.Fatal(err)
			}
			if got, want := out.String(), "foo\n"; got != want {
				t.Errorf("output = %q; want %q", got, want)
			}
		}
	}
	t.Run("10MB", testWith(strings.NewReader(strings.Repeat("x", 10<<20))))
	t.Run("Infinite", testWith(delayedInfiniteReader{}))
}

type badWriter struct{}

func (w *badWriter) Write(data []byte) (int, error) {
	return 0, io.ErrUnexpectedEOF
}

func TestClosePipeOnCopyError(t *testing.T) {
	cmd := helperCommand(t, "yes")
	cmd.Cmd父类.Stdout = new(badWriter)
	err := cmd.I运行()
	if err == nil {
		t.Errorf("yes unexpectedly completed successfully")
	}
}

func TestOutputStderrCapture(t *testing.T) {
	cmd := helperCommand(t, "stderrfail")
	_, err := cmd.I运行_带返回值()
	ee, ok := err.(*exec.ExitError)
	if !ok {
		t.Fatalf("I运行_带返回值 error type = %T; want ExitError", err)
	}
	got := string(ee.Stderr)
	want := "some stderr text\n"
	if got != want {
		t.Errorf("ExitError.Stderr = %q; want %q", got, want)
	}
}

func TestContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	c := helperCommandContext(t, ctx, "pipetest")
	stdin, err := c.I取Stdin管道()
	if err != nil {
		t.Fatal(err)
	}
	stdout, err := c.I取标准管道()
	if err != nil {
		t.Fatal(err)
	}
	if err := c.I运行_异步(); err != nil {
		t.Fatal(err)
	}

	if _, err := stdin.Write([]byte("O:hi\n")); err != nil {
		t.Fatal(err)
	}
	buf := make([]byte, 5)
	n, err := io.ReadFull(stdout, buf)
	if n != len(buf) || err != nil || string(buf) != "O:hi\n" {
		t.Fatalf("ReadFull = %d, %v, %q", n, err, buf[:n])
	}
	go cancel()

	if err := c.I等待运行完成(); err == nil {
		t.Fatal("expected I等待运行完成 failure")
	}
}

func TestContextCancel(t *testing.T) {
	if runtime.GOOS == "netbsd" && runtime.GOARCH == "arm64" {
		maySkipHelperCommand("cat")
		testenv.SkipFlaky(t, 42061)
	}

	// To reduce noise in the final goroutine dump,
	// let other parallel tests complete if possible.
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	c := helperCommandContext(t, ctx, "cat")

	stdin, err := c.I取Stdin管道()
	if err != nil {
		t.Fatal(err)
	}
	defer stdin.Close()

	if err := c.I运行_异步(); err != nil {
		t.Fatal(err)
	}

	// At this point the process is alive. Ensure it by sending data to stdin.
	if _, err := io.WriteString(stdin, "echo"); err != nil {
		t.Fatal(err)
	}

	cancel()

	// Calling cancel should have killed the process, so writes
	// should now fail.  Give the process a little while to die.
	start := time.Now()
	delay := 1 * time.Millisecond
	for {
		if _, err := io.WriteString(stdin, "echo"); err != nil {
			break
		}

		if time.Since(start) > time.Minute {
			// Panic instead of calling t.Fatal so that we get a goroutine dump.
			// We want to know exactly what the os/exec goroutines got stuck on.
			panic("canceling context did not stop program")
		}

		// Back off exponentially (up to 1-second sleeps) to give the OS time to
		// terminate the process.
		delay *= 2
		if delay > 1*time.Second {
			delay = 1 * time.Second
		}
		time.Sleep(delay)
	}

	if err := c.I等待运行完成(); err == nil {
		t.Error("program unexpectedly exited successfully")
	} else {
		t.Logf("exit status: %v", err)
	}
}

// test that environment variables are de-duped.
func TestDedupEnvEcho(t *testing.T) {
	cmd := helperCommand(t, "echoenv", "FOO")
	cmd.Cmd父类.Env = append(cmd.I取环境变量数组(), "FOO=bad", "FOO=good")
	out, err := cmd.I运行_带组合返回值()
	if err != nil {
		t.Fatal(err)
	}
	if got, want := strings.TrimSpace(string(out)), "good"; got != want {
		t.Errorf("output = %q; want %q", got, want)
	}
}

func TestEnvNULCharacter(t *testing.T) {
	if runtime.GOOS == "plan9" {
		t.Skip("plan9 explicitly allows NUL in the enviroment")
	}
	cmd := helperCommand(t, "echoenv", "FOO", "BAR")
	cmd.Cmd父类.Env = append(cmd.I取环境变量数组(), "FOO=foo\x00BAR=bar")
	out, err := cmd.I运行_带组合返回值()
	if err == nil {
		t.Errorf("output = %q; want error", string(out))
	}
}

func TestString(t *testing.T) {
	echoPath, err := cmd类.I查找路径("echo")
	if err != nil {
		t.Skip(err)
	}
	tests := [...]struct {
		path string
		args []string
		want string
	}{
		{"echo", nil, echoPath},
		{"echo", []string{"a"}, echoPath + " a"},
		{"echo", []string{"a", "b"}, echoPath + " a b"},
	}
	for _, test := range tests {
		cmd := cmd类.I设置命令(test.path, test.args...)
		if got := cmd.I取命令(); got != test.want {
			t.Errorf("String(%q, %q) = %q, want %q", test.path, test.args, got, test.want)
		}
	}
}

func TestStringPathNotResolved(t *testing.T) {
	_, err := cmd类.I查找路径("makemeasandwich")
	if err == nil {
		t.Skip("wow, thanks")
	}
	cmd := cmd类.I设置命令("makemeasandwich", "-lettuce")
	want := "makemeasandwich -lettuce"
	if got := cmd.I取命令(); got != want {
		t.Errorf("String(%q, %q) = %q, want %q", "makemeasandwich", "-lettuce", got, want)
	}
}

func TestNoPath(t *testing.T) {
	err := new(exec.Cmd).Start()
	want := "exec: no command"
	if err == nil || err.Error() != want {
		t.Errorf("new(Cmd).I运行_异步() = %v, want %q", err, want)
	}
}
