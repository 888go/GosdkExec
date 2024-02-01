// Copyright 2012 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cmd类_test

import (
	"bytes"
	"context"
	"e.coding.net/gogit/go/gosdk/core/os_exec_cn"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"
)

func ExampleLookPath() {
	path, err := cmd类.I查找路径("fortune")
	if err != nil {
		log.Fatal("安装fortune是您的未来")
	}
	fmt.Printf("fortune在%s可用\n", path)
}

func ExampleCommand() {
	cmd := cmd类.I设置命令("tr", "a-z", "A-Z")
	cmd.Cmd父类.Stdin = strings.NewReader("some input")
	var out bytes.Buffer
	cmd.Cmd父类.Stdout = &out
	err := cmd.I运行()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("in all caps: %q\n", out.String())
}

func ExampleCommand_environment() {
	cmd := cmd类.I设置命令("prog")
	cmd.Cmd父类.Env = append(os.Environ(),
		"FOO=duplicate_value", // ignored
		"FOO=actual_value",    // this value is used
	)
	if err := cmd.I运行(); err != nil {
		log.Fatal(err)
	}
}

func ExampleCmd_Output() {
	out, err := cmd类.I设置命令("date").I运行_带返回值()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("The date is %s\n", out)
}

func ExampleCmd_Run() {
	cmd := cmd类.I设置命令("sleep", "1")
	log.Printf("正在运行命令并等待其完成...")
	err := cmd.I运行()
	log.Printf("命令已完成，但有错误: %v", err)
}

func ExampleCmd_Start() {
	cmd := cmd类.I设置命令("sleep", "5")
	err := cmd.I运行_异步()
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("等待命令完成...")
	err = cmd.I等待运行完成()
	log.Printf("命令已完成，但有错误: %v", err)
}

func ExampleCmd_StdoutPipe() {
	cmd := cmd类.I设置命令("echo", "-n", `{"Name": "Bob", "Age": 32}`)
	stdout, err := cmd.I取标准管道()
	if err != nil {
		log.Fatal(err)
	}
	if err := cmd.I运行_异步(); err != nil {
		log.Fatal(err)
	}
	var person struct {
		Name string
		Age  int
	}
	if err := json.NewDecoder(stdout).Decode(&person); err != nil {
		log.Fatal(err)
	}
	if err := cmd.I等待运行完成(); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%s是%d岁\n", person.Name, person.Age)
}

func ExampleCmd_StdinPipe() {
	cmd := cmd类.I设置命令("cat")
	stdin, err := cmd.I取Stdin管道()
	if err != nil {
		log.Fatal(err)
	}

	go func() {
		defer stdin.Close()
		io.WriteString(stdin, "写入stdin的值传递给cmd的标准输入")
	}()

	out, err := cmd.I运行_带组合返回值()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("%s\n", out)
}

func ExampleCmd_StderrPipe() {
	cmd := cmd类.I设置命令("sh", "-c", "echo stdout; echo 1>&2 stderr")
	stderr, err := cmd.I取Stderr管道()
	if err != nil {
		log.Fatal(err)
	}

	if err := cmd.I运行_异步(); err != nil {
		log.Fatal(err)
	}

	slurp, _ := io.ReadAll(stderr)
	fmt.Printf("%s\n", slurp)

	if err := cmd.I等待运行完成(); err != nil {
		log.Fatal(err)
	}
}

func ExampleCmd_CombinedOutput() {
	cmd := cmd类.I设置命令("sh", "-c", "echo stdout; echo 1>&2 stderr")
	stdoutStderr, err := cmd.I运行_带组合返回值()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%s\n", stdoutStderr)
}

func ExampleCmd_Environ() {
	cmd := cmd类.I设置命令("pwd")

	// 在调用cmd.Environ之前设置Dir，使其包含更新的PWD变量（在使用该变量的平台上）。
	cmd.Cmd父类.Dir = ".."
	cmd.Cmd父类.Env = append(cmd.I取环境变量数组(), "POSIXLY_CORRECT=1")

	out, err := cmd.I运行_带返回值()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%s\n", out)
}

func ExampleCommandContext() {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	if err := cmd类.I设置命令_上下文(ctx, "sleep", "5").I运行(); err != nil {
		// 这将在100毫秒后失败。5秒钟的睡眠将被中断。
	}
}
