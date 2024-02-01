// Copyright 2013 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Use an external test to avoid os/exec -> internal/testenv -> os/exec
// circular dependency.

package cmd类_test

import (
	"e.coding.net/gogit/go/gosdk/core/os_exec_cn"
	"e.coding.net/gogit/go/gosdk/internal/testenv"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func init() {
	registerHelperCommand("exec", cmdExec)
	registerHelperCommand("lookpath", cmdLookPath)
}

func cmdLookPath(args ...string) {
	p, err := cmd类.I查找路径(args[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "I查找路径 failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Print(p)
}

func cmdExec(args ...string) {
	cmd := cmd类.I设置命令(args[1])
	cmd.Cmd父类.Dir = args[0]
	if errors.Is(cmd.Cmd父类.Err, exec.ErrDot) {
		cmd.Cmd父类.Err = nil
	}
	output, err := cmd.I运行_带组合返回值()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Child: %s %s", err, string(output))
		os.Exit(1)
	}
	fmt.Printf("%s", string(output))
}

func installExe(t *testing.T, dest, src string) {
	fsrc, err := os.Open(src)
	if err != nil {
		t.Fatal("os.Open failed: ", err)
	}
	defer fsrc.Close()
	fdest, err := os.Create(dest)
	if err != nil {
		t.Fatal("os.Create failed: ", err)
	}
	defer fdest.Close()
	_, err = io.Copy(fdest, fsrc)
	if err != nil {
		t.Fatal("io.Copy failed: ", err)
	}
}

func installBat(t *testing.T, dest string) {
	f, err := os.Create(dest)
	if err != nil {
		t.Fatalf("failed to create batch file: %v", err)
	}
	defer f.Close()
	fmt.Fprintf(f, "@echo %s\n", dest)
}

func installProg(t *testing.T, dest, srcExe string) {
	err := os.MkdirAll(filepath.Dir(dest), 0700)
	if err != nil {
		t.Fatal("os.MkdirAll failed: ", err)
	}
	if strings.ToLower(filepath.Ext(dest)) == ".bat" {
		installBat(t, dest)
		return
	}
	installExe(t, dest, srcExe)
}

type lookPathTest struct {
	rootDir   string
	PATH      string
	PATHEXT   string
	files     []string
	searchFor string
	fails     bool // 预计测试将失败
}

func (test lookPathTest) runProg(t *testing.T, env []string, cmd *cmd类.Cmd) (string, error) {
	cmd.Cmd父类.Env = env
	cmd.Cmd父类.Dir = test.rootDir
	args := append([]string(nil), cmd.Cmd父类.Args...)
	args[0] = filepath.Base(args[0])
	cmdText := fmt.Sprintf("%q command", strings.Join(args, " "))
	out, err := cmd.I运行_带组合返回值()
	if (err != nil) != test.fails {
		if test.fails {
			t.Fatalf("test=%+v: %s succeeded, but expected to fail", test, cmdText)
		}
		t.Fatalf("test=%+v: %s failed, but expected to succeed: %v - %v", test, cmdText, err, string(out))
	}
	if err != nil {
		return "", fmt.Errorf("test=%+v: %s failed: %v - %v", test, cmdText, err, string(out))
	}
	// 标准化程序输出
	p := string(out)
	//trim终止\r\n和批处理文件输出
	for len(p) > 0 && (p[len(p)-1] == '\n' || p[len(p)-1] == '\r') {
		p = p[:len(p)-1]
	}
	if !filepath.IsAbs(p) {
		return p, nil
	}
	if p[:len(test.rootDir)] != test.rootDir {
		t.Fatalf("test=%+v: %s output is wrong: %q must have %q prefix", test, cmdText, p, test.rootDir)
	}
	return p[len(test.rootDir)+1:], nil
}

func updateEnv(env []string, name, value string) []string {
	for i, e := range env {
		if strings.HasPrefix(strings.ToUpper(e), name+"=") {
			env[i] = name + "=" + value
			return env
		}
	}
	return append(env, name+"="+value)
}

func createEnv(dir, PATH, PATHEXT string) []string {
	env := os.Environ()
	env = updateEnv(env, "PATHEXT", PATHEXT)
	// Add dir in front of every directory in the PATH.
	dirs := filepath.SplitList(PATH)
	for i := range dirs {
		dirs[i] = filepath.Join(dir, dirs[i])
	}
	path := strings.Join(dirs, ";")
	env = updateEnv(env, "PATH", os.Getenv("SystemRoot")+"/System32;"+path)
	return env
}

// createFiles将srcPath文件复制到多个文件中。
// 它使用dir作为所有目标文件的前缀。
func createFiles(t *testing.T, dir string, files []string, srcPath string) {
	for _, f := range files {
		installProg(t, filepath.Join(dir, f), srcPath)
	}
}

func (test lookPathTest) run(t *testing.T, tmpdir, printpathExe string) {
	test.rootDir = tmpdir
	createFiles(t, test.rootDir, test.files, printpathExe)
	env := createEnv(test.rootDir, test.PATH, test.PATHEXT)
	// 使用新的环境和工作目录集运行 "cmd.exe /c test.searchFor" 。所有候选文件都是printpath.exe的副本。这些文件将在运行时输出其程序路径。
	should, errCmd := test.runProg(t, env, cmd类.I设置命令("cmd", "/c", test.searchFor))
	// I运行 the lookpath program with new environment and work directory set.
	have, errLP := test.runProg(t, env, helperCommand(t, "lookpath", test.searchFor))
	// 比较结果。
	if errCmd == nil && errLP == nil {
		// 两者都成功了
		if should != have {
			t.Fatalf("test=%+v:\ncmd /c ran: %s\nlookpath found: %s", test, should, have)
		}
		return
	}
	if errCmd != nil && errLP != nil {
		// both failed -> continue
		return
	}
	if errCmd != nil {
		t.Fatal(errCmd)
	}
	if errLP != nil {
		t.Fatal(errLP)
	}
}

var lookPathTests = []lookPathTest{
	{
		PATHEXT:   `.COM;.EXE;.BAT`,
		PATH:      `p1;p2`,
		files:     []string{`p1\a.exe`, `p2\a.exe`, `p2\a`},
		searchFor: `a`,
	},
	{
		PATHEXT:   `.COM;.EXE;.BAT`,
		PATH:      `p1.dir;p2.dir`,
		files:     []string{`p1.dir\a`, `p2.dir\a.exe`},
		searchFor: `a`,
	},
	{
		PATHEXT:   `.COM;.EXE;.BAT`,
		PATH:      `p1;p2`,
		files:     []string{`p1\a.exe`, `p2\a.exe`},
		searchFor: `a.exe`,
	},
	{
		PATHEXT:   `.COM;.EXE;.BAT`,
		PATH:      `p1;p2`,
		files:     []string{`p1\a.exe`, `p2\b.exe`},
		searchFor: `b`,
	},
	{
		PATHEXT:   `.COM;.EXE;.BAT`,
		PATH:      `p1;p2`,
		files:     []string{`p1\b`, `p2\a`},
		searchFor: `a`,
		fails:     true, // TODO(brainman): do not know why this fails
	},
	// 如果命令名指定了路径，shell将在指定的路径中搜索与命令名匹配的可执行文件。如果找到匹配项，则执行外部命令（可执行文件）。
	{
		PATHEXT:   `.COM;.EXE;.BAT`,
		PATH:      `p1;p2`,
		files:     []string{`p1\a.exe`, `p2\a.exe`},
		searchFor: `p2\a`,
	},
	// If the command name specifies a path, the shell searches
	// the specified path for an executable file matching the command
	// name. ... If no match is found, the shell reports an error
	// and command processing completes.
	{
		PATHEXT:   `.COM;.EXE;.BAT`,
		PATH:      `p1;p2`,
		files:     []string{`p1\b.exe`, `p2\a.exe`},
		searchFor: `p2\b`,
		fails:     true,
	},
	// 如果命令名未指定路径, shell在当前目录中搜索与命令名匹配的可执行文件. 如果找到匹配项，则执行外部命令（可执行文件）。
	{
		PATHEXT:   `.COM;.EXE;.BAT`,
		PATH:      `p1;p2`,
		files:     []string{`a`, `p1\a.exe`, `p2\a.exe`},
		searchFor: `a`,
	},
	// shell现在按照列出的顺序搜索PATH环境变量指定的每个目录, 对于与命令名称匹配的可执行文件.
	//如果找到匹配项，则执行外部命令（可执行文件）。
	{
		PATHEXT:   `.COM;.EXE;.BAT`,
		PATH:      `p1;p2`,
		files:     []string{`p1\a.exe`, `p2\a.exe`},
		searchFor: `a`,
	},
	// shell现在搜索PATH环境变量指定的每个目录，
	//按照列出的顺序查找与命令名匹配的可执行文件。如果找不到匹配项，shell将报告错误并完成命令处理。
	{
		PATHEXT:   `.COM;.EXE;.BAT`,
		PATH:      `p1;p2`,
		files:     []string{`p1\a.exe`, `p2\a.exe`},
		searchFor: `b`,
		fails:     true,
	},
	// 如果命令名包含文件扩展名，shell将在每个目录中搜索命令名指定的确切文件名。
	{
		PATHEXT:   `.COM;.EXE;.BAT`,
		PATH:      `p1;p2`,
		files:     []string{`p1\a.exe`, `p2\a.exe`},
		searchFor: `a.exe`,
	},
	{
		PATHEXT:   `.COM;.EXE;.BAT`,
		PATH:      `p1;p2`,
		files:     []string{`p1\a.exe`, `p2\a.exe`},
		searchFor: `a.com`,
		fails:     true, // 包含扩展名，但文件名不完全匹配
	},
	{
		PATHEXT:   `.COM;.EXE;.BAT`,
		PATH:      `p1`,
		files:     []string{`p1\a.exe.exe`},
		searchFor: `a.exe`,
	},
	{
		PATHEXT:   `.COM;.BAT`,
		PATH:      `p1;p2`,
		files:     []string{`p1\a.exe`, `p2\a.exe`},
		searchFor: `a.exe`,
	},
	// 如果命令名不包含文件扩展名，则shell将添加PATHEXT环境变量中列出的扩展名，
	// 并在目录中搜索该文件名。请注意，在继续搜索下一个目录（如果有）之前，shell会尝试特定目录中所有可能的文件扩展名。
	{
		PATHEXT:   `.COM;.EXE`,
		PATH:      `p1;p2`,
		files:     []string{`p1\a.bat`, `p2\a.exe`},
		searchFor: `a`,
	},
	{
		PATHEXT:   `.COM;.EXE;.BAT`,
		PATH:      `p1;p2`,
		files:     []string{`p1\a.bat`, `p2\a.exe`},
		searchFor: `a`,
	},
	{
		PATHEXT:   `.COM;.EXE;.BAT`,
		PATH:      `p1;p2`,
		files:     []string{`p1\a.bat`, `p1\a.exe`, `p2\a.bat`, `p2\a.exe`},
		searchFor: `a`,
	},
	{
		PATHEXT:   `.COM`,
		PATH:      `p1;p2`,
		files:     []string{`p1\a.bat`, `p2\a.exe`},
		searchFor: `a`,
		fails:     true, // 尝试了PATHEXT中的所有扩展，但都不匹配
	},
}

func TestLookPathWindows(t *testing.T) {
	tmp := t.TempDir()
	printpathExe := buildPrintPathExe(t, tmp)

	// I运行 all tests.
	for i, test := range lookPathTests {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			dir := filepath.Join(tmp, "d"+strconv.Itoa(i))
			err := os.Mkdir(dir, 0700)
			if err != nil {
				t.Fatal("Mkdir failed: ", err)
			}
			test.run(t, dir, printpathExe)
		})
	}
}

type commandTest struct {
	PATH  string
	files []string
	dir   string
	arg0  string
	want  string
	fails bool // 预计测试将失败
}

func (test commandTest) isSuccess(rootDir, output string, err error) error {
	if err != nil {
		return fmt.Errorf("test=%+v: exec: %v %v", test, err, output)
	}
	path := output
	if path[:len(rootDir)] != rootDir {
		return fmt.Errorf("test=%+v: %q must have %q prefix", test, path, rootDir)
	}
	path = path[len(rootDir)+1:]
	if path != test.want {
		return fmt.Errorf("test=%+v: want %q, got %q", test, test.want, path)
	}
	return nil
}

func (test commandTest) runOne(t *testing.T, rootDir string, env []string, dir, arg0 string) {
	cmd := helperCommand(t, "exec", dir, arg0)
	cmd.Cmd父类.Dir = rootDir
	cmd.Cmd父类.Env = env
	output, err := cmd.I运行_带组合返回值()
	err = test.isSuccess(rootDir, string(output), err)
	if (err != nil) != test.fails {
		if test.fails {
			t.Errorf("test=%+v: succeeded, but expected to fail", test)
		} else {
			t.Error(err)
		}
	}
}

func (test commandTest) run(t *testing.T, rootDir, printpathExe string) {
	createFiles(t, rootDir, test.files, printpathExe)
	PATHEXT := `.COM;.EXE;.BAT`
	env := createEnv(rootDir, test.PATH, PATHEXT)
	test.runOne(t, rootDir, env, test.dir, test.arg0)
}

var commandTests = []commandTest{
	// 测试不带斜杠的命令，如`a.exe`
	{
		// 应在当前目录中找到.exe
		files: []string{`a.exe`},
		arg0:  `a.exe`,
		want:  `a.exe`,
	},
	{
		// 如上文所述，但添加PATH以尝试中断测试
		PATH:  `p2;p`,
		files: []string{`a.exe`, `p\a.exe`, `p2\a.exe`},
		arg0:  `a.exe`,
		want:  `a.exe`,
	},
	{
		//与上面类似，但命令使用“a”而不是“.exe”
		PATH:  `p2;p`,
		files: []string{`a.exe`, `p\a.exe`, `p2\a.exe`},
		arg0:  `a`,
		want:  `a.exe`,
	},
	//测试带有斜线的命令，如  `.\a.exe`
	{
		//应该找到 p\a.exe
		files: []string{`p\a.exe`},
		arg0:  `p\a.exe`,
		want:  `p\a.exe`,
	},
	{
		//同上，但添加了“.”在可执行文件前面应该还可以
		files: []string{`p\a.exe`},
		arg0:  `.\p\a.exe`,
		want:  `p\a.exe`,
	},
	{
		// 与上面一样，但添加了PATH以尝试打破它
		PATH:  `p2`,
		files: []string{`p\a.exe`, `p2\a.exe`},
		arg0:  `p\a.exe`,
		want:  `p\a.exe`,
	},
	{
		// 与上面类似，但确保即使对于带有斜杠的命令也尝试.exe
		PATH:  `p2`,
		files: []string{`p\a.exe`, `p2\a.exe`},
		arg0:  `p\a`,
		want:  `p\a.exe`,
	},
	// 使用c.Dir集测试命令，如“a.exe”
	{
		// 不应在p中找到.exe，因为LookPath（`a.exe`）将失败
		files: []string{`p\a.exe`},
		dir:   `p`,
		arg0:  `a.exe`,
		want:  `p\a.exe`,
		fails: true,
	},
	{
		// I查找路径(`a.exe`) will find `.\a.exe`, but prefixing that with
		// dir `p\a.exe` will refer to a non-existent file
		files: []string{`a.exe`, `p\not_important_file`},
		dir:   `p`,
		arg0:  `a.exe`,
		want:  `a.exe`,
		fails: true,
	},
	{
		// 如上所述，但通过在引用的目标中安装文件使测试成功 (so I查找路径(`a.exe`) will still
		// find `.\a.exe`, but we successfully execute `p\a.exe`)
		files: []string{`a.exe`, `p\a.exe`},
		dir:   `p`,
		arg0:  `a.exe`,
		want:  `p\a.exe`,
	},
	{
		// 如上文所述，但添加PATH以尝试中断测试
		PATH:  `p2;p`,
		files: []string{`a.exe`, `p\a.exe`, `p2\a.exe`},
		dir:   `p`,
		arg0:  `a.exe`,
		want:  `p\a.exe`,
	},
	{
		// 与上面类似，但命令使用“a”而不是“.exe”
		PATH:  `p2;p`,
		files: []string{`a.exe`, `p\a.exe`, `p2\a.exe`},
		dir:   `p`,
		arg0:  `a`,
		want:  `p\a.exe`,
	},
	{
		// 在PATH中查找“a.exe”，而不考虑目录集，因为LookPath在这种情况下返回完整路径
		PATH:  `p2;p`,
		files: []string{`p\a.exe`, `p2\a.exe`},
		dir:   `p`,
		arg0:  `a.exe`,
		want:  `p2\a.exe`,
	},
	//测试命令， like `.\a.exe`, with c.Dir set
	{
		//当命令为路径时，应使用dir，如 ".\a.exe"
		files: []string{`p\a.exe`},
		dir:   `p`,
		arg0:  `.\a.exe`,
		want:  `p\a.exe`,
	},
	{
		//与上面一样，但添加了PATH以尝试打破它
		PATH:  `p2`,
		files: []string{`p\a.exe`, `p2\a.exe`},
		dir:   `p`,
		arg0:  `.\a.exe`,
		want:  `p\a.exe`,
	},
	{
		// 与上面类似，但确保即使对于带有斜杠的命令也尝试.exe
		PATH:  `p2`,
		files: []string{`p\a.exe`, `p2\a.exe`},
		dir:   `p`,
		arg0:  `.\a`,
		want:  `p\a.exe`,
	},
}

func TestCommand(t *testing.T) {
	tmp := t.TempDir()
	printpathExe := buildPrintPathExe(t, tmp)

	// 运行所有测试。
	for i, test := range commandTests {
		dir := filepath.Join(tmp, "d"+strconv.Itoa(i))
		err := os.Mkdir(dir, 0700)
		if err != nil {
			t.Fatal("Mkdir failed: ", err)
		}
		test.run(t, dir, printpathExe)
	}
}

// buildPrintPathExe 创建一个Go程序，打印自己的路径。
// dir是创建可执行文件的临时目录。该函数返回所创建程序的完整路径。
func buildPrintPathExe(t *testing.T, dir string) string {
	const name = "printpath"
	srcname := name + ".go"
	err := os.WriteFile(filepath.Join(dir, srcname), []byte(printpathSrc), 0644)
	if err != nil {
		t.Fatalf("failed to create source: %v", err)
	}
	if err != nil {
		t.Fatalf("failed to execute template: %v", err)
	}
	outname := name + ".exe"
	cmd := cmd类.I设置命令(testenv.GoToolPath(t), "build", "-o", outname, srcname)
	cmd.Cmd父类.Dir = dir
	out, err := cmd.I运行_带组合返回值()
	if err != nil {
		t.Fatalf("failed to build executable: %v - %v", err, string(out))
	}
	return filepath.Join(dir, outname)
}

const printpathSrc = `
package main

import (
	"os"
	"syscall"
	"unicode/utf16"
	"unsafe"
)

func getMyName() (string, error) {
	var sysproc = syscall.MustLoadDLL("kernel32.dll").MustFindProc("GetModuleFileNameW")
	b := make([]uint16, syscall.MAX_PATH)
	r, _, err := sysproc.Call(0, uintptr(unsafe.Pointer(&b[0])), uintptr(len(b)))
	n := uint32(r)
	if n == 0 {
		return "", err
	}
	return string(utf16.Decode(b[0:n])), nil
}

func main() {
	path, err := getMyName()
	if err != nil {
		os.Stderr.Write([]byte("getMyName failed: " + err.Error() + "\n"))
		os.Exit(1)
	}
	os.Stdout.Write([]byte(path))
}
`
