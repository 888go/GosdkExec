// 版权所有2009 The Go Authors。保留所有权利。此源代码的使用受BSD风格许可证的约束，该许可证可以在license文件中找到。

// Package cmd类 Package exec 包exec运行外部命令。它包装os.StartProcess，以使重新映射stdin和stdout、将IO与管道连接以及进行其他调整更加容易。
//
// 与来自C和其他语言的“系统”库调用不同，osexec包故意不调用系统shell，也不扩展任何glob模式或处理通常由shell执行的其他扩展、管道或重定向。
// 该包的行为更像C的“exec”函数家族。要扩展glob模式，可以直接调用shell，小心避开任何危险的输入，或者使用pathfilepath包的glob函数。
// 要扩展环境变量，请使用包os的ExpandEnv。
//
// 请注意，此包中的示例假定为Unix系统。它们可能不会在Windows上运行，也不会在golang.org和godoc.org使用的Go Playground上运行。
//
// # 当前目录中的可执行文件
//
// 函数Command和LookPath在当前路径中列出的目录中查找程序，遵循主机操作系统的惯例。
// 几十年来，操作系统一直将当前目录包含在搜索中，有时默认情况下隐式配置，有时显式配置。
// 现代实践表明，包含当前目录通常是意外的，并且常常会导致安全问题。
//
// 为了避免这些安全问题，从Go 1.19开始, 此包不会使用相对于当前目录的隐式或显式路径项来解析程序。
// 也就是说，如果运行exec.I查找路径（“go”）， 它不会返回 \Windows上的go.exe,无论路径如何配置。
// 如果通常的路径算法将得到该答案，
// 这些函数返回满足错误的错误err.Is（err，ErrDot）。
//
// 例如，考虑以下两个程序片段：
//
//	path, err := exec.I查找路径("prog")
//	if err != nil {
//		log.Fatal(err)
//	}
//	use(path)
//
// and
//
//	cmd := exec.I设置命令("prog")
//	if err := cmd.I运行(); err != nil {
//		log.Fatal(err)
//	}
//
// 这些将无法找到并运行 ./prog 或 .\prog.exe, 无论路径如何配置。
//
// 始终希望从当前目录运行程序的代码
// 可以重写为“./prog”而不是“prog”。
//
// 坚持包含相对路径项的结果的代码可以使用错误替代错误。是否检查：
//
//	path, err := exec.I查找路径("prog")
//	if errors.Is(err, exec.ErrDot) {
//		err = nil
//	}
//	if err != nil {
//		log.Fatal(err)
//	}
//	use(path)
//
// and
//
//	cmd := exec.I设置命令("prog")
//	if errors.Is(cmd.Cmd父类.Err, exec.ErrDot) {
//		cmd.Cmd父类.Err = nil
//	}
//	if err := cmd.I运行(); err != nil {
//		log.Fatal(err)
//	}
//
// 设置环境变量GODEBUG=exerrdot=0将完全禁用ErrDot的生成, 对于无法应用更多目标修复的程序，暂时恢复Go 1.19之前的行为。
// 未来版本的Go可能会删除对该变量的支持。
//
// 在添加此类重写之前，请确保您了解这样做的安全含义。
// 见 https://go.dev/blog/path-security 了解更多信息。
package cmd类

import (
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"strconv"
)

// Error LookPath无法将文件分类为可执行文件时返回。
type Error struct {
	// Name是发生错误的文件名。
	Name string
	// Err是基本错误。
	Err error
}

func (e *Error) Error() string {
	return "exec: " + strconv.Quote(e.Name) + ": " + e.Err.Error()
}

func (e *Error) Unwrap() error { return e.Err }

// Cmd 表示正在准备或运行的外部命令。
//
// Cmd在调用其Run、Output或CombinedOutput方法后无法重用。
type Cmd struct {
	Cmd父类 *exec.Cmd
}

// I设置命令 返回Cmd结构以使用给定参数执行命名程序。
//
// 它只设置返回结构中的Path和Args。
//
// 若名称不包含路径分隔符，则命令使用LookPath将名称解析为完整的路径（如果可能）。否则，它直接使用名称作为路径。
//
// 返回的Cmd的Args字段由命令名后跟arg元素构成， 因此arg不应包含命令名本身. 例如, I设置命令("echo", "hello").
// Args[0] 始终是名称，而不是可能解析的路径。
//
// 在Windows上，进程以单个字符串的形式接收整个命令行，并执行自己的解析.
// 命令使用与使用CommandLineToArgvW的应用程序兼容的算法（这是最常见的方法）将Args组合并引用到命令行字符串中.
// 值得注意的例外是msiexec.exe和cmd.exe（以及所有批处理文件）， 它们具有不同的去激励算法。
// 在这些或其他类似情况下, 您可以自己引用，并在SysProcAttr.CmdLine中提供完整的命令行，将Args留空。
func I设置命令(进程名 string, 命令参数 ...string) *Cmd {
	c := exec.Command(进程名, 命令参数...)
	if c == nil {
		return nil
	}
	//
	return &Cmd{c}
}

// I设置命令_上下文 与 I设置命令 类似，但包含上下文。
//
// 如果上下文在命令自身完成之前完成，则提供的上下文用于终止进程（通过调用os.ProcessKill）。
func I设置命令_上下文(上下文 context.Context, 进程名 string, 命令参数 ...string) *Cmd {
	c := exec.CommandContext(上下文, 进程名, 命令参数...)
	if c == nil {
		return nil
	}
	return &Cmd{c}
}

// I取命令 返回c的可读描述。
// 它仅用于调试。
// 特别是，它不适合用作外壳的输入。
// String的输出可能因Go版本而异。
func (c *Cmd) I取命令() string {
	if c == nil {
		return ""
	}
	return c.Cmd父类.String()
}

// String 返回c的可读描述。
// 它仅用于调试。
// 特别是，它不适合用作外壳的输入。
// String的输出可能因Go版本而异。
func (c *Cmd) String() string {
	if c == nil {
		return ""
	}
	return c.Cmd父类.String()
}

// I运行 启动指定的命令并等待其完成。
//
// 如果命令运行，复制stdin、stdout和stderr没有问题，并且以零退出状态退出，则返回的错误为零。
//
// I如果命令启动但未成功完成，则错误类型为*ExitError。对于其他情况，可能会返回其他错误类型。
//
// 如果调用goroutine使用runtime.LockOSThread锁定了操作系统线程，并修改了任何可继承的OS级线程状态（例如，Linux或Plan 9名称空间），则新进程将继承调用者的线程状态。
func (c *Cmd) I运行() error {
	if c == nil {
		return errors.New("cmd类对象为nil")
	}
	return c.Cmd父类.Run()
} //I运行

// I运行_异步 启动指定的命令，但不等待它完成。
//
// 如果Start成功返回，将设置c.Process字段。
//
// 成功调用Start后，必须调用Wait方法才能释放相关的系统资源。
func (c *Cmd) I运行_异步() error {
	if c == nil {
		return errors.New("cmd类对象为nil")
	}
	return c.Cmd父类.Start()
}

// ExitError 报告命令退出失败。
type ExitError struct {
	*os.ProcessState

	// Stderr如果未收集标准错误，则保存Cmd.output方法中标准错误输出的子集。
	//
	// 如果错误输出很长，则Stderr可能只包含输出的前缀和后缀，中间用省略字节数的文本替换。
	//
	// 提供Stderr用于调试，以包含在错误消息中。有其他需求的用户应根据需要重定向Cmd.Stderr。
	Stderr []byte
}

func (e *ExitError) Error() string {
	return e.ProcessState.String()
}

// I等待运行完成 等待命令退出，并等待任何复制到stdin或从stdout或stderr复制完成。
//
// 命令必须已由Start启动。
//
// 如果命令运行，复制stdin、stdout和stderr没有问题，并且以零退出状态退出，则返回的错误为零。
//
// 如果命令无法运行或未成功完成，则错误类型为*ExitError。IO问题可能会返回其他错误类型。
//
// 如果c.Stdin、c.Stdout或c.Stderr中的任何一个不是 *os.File, 等待还等待各个IO循环复制到进程或从进程复制完成。
//
// I等待运行完成 释放与Cmd关联的任何资源。
func (c *Cmd) I等待运行完成() error {
	if c == nil {
		return errors.New("cmd类对象为nil")
	}
	return c.Cmd父类.Wait()
}

// I运行_带返回值 运行命令并返回其标准输出。
// 任何返回的错误通常为*ExitError类型。
// 如果c.Stderr为零，则输出填充ExitError.Stderr。
func (c *Cmd) I运行_带返回值() ([]byte, error) {
	if c == nil {
		return nil, errors.New("cmd类对象为nil")
	}
	return c.Cmd父类.Output()
}

// I运行_带组合返回值 运行该命令并返回其组合的标准输出和标准错误。
// 简单点理解, 这个命令连错误提示都一并给你返回了.
//
// 比如:
// 返回2 := cmd类.I设置命令("go", "111")
// 返回值2, _ := 返回2.I运行_带组合返回值()
// fmt.Println(string(返回值2))
// 返回如下:
// go 111: unknown command
// Run 'go help' for usage.
func (c *Cmd) I运行_带组合返回值() ([]byte, error) {
	if c == nil {
		return nil, errors.New("cmd类对象为nil")
	}
	return c.Cmd父类.CombinedOutput()
}

// I取Stdin管道 StdinPipe方法返回一个在命令Start后与命令标准输入关联的管道。Wait方法获知命令结束后会关闭这个管道。
// 必要时调用者可以调用Close方法来强行关闭管道，例如命令在输入关闭后才会执行返回时需要显式关闭管道。
func (c *Cmd) I取Stdin管道() (io.WriteCloser, error) {
	if c == nil {
		return nil, errors.New("cmd类对象为nil")
	}
	return c.Cmd父类.StdinPipe()
	//Stdin管道
}

// I取标准管道 返回一个管道，该管道将在命令启动时连接到命令的标准输出。
//
// StdoutPipe方法返回一个在命令Start后与命令标准输出关联的管道。Wait方法获知命令结束后会关闭这个管道，一般不需要显式的关闭该管道。
// 但是在从管道读取完全部数据之前调用Wait是错误的；同样使用StdoutPipe方法时调用Run函数也是错误的。
func (c *Cmd) I取标准管道() (io.ReadCloser, error) {
	if c == nil {
		return nil, errors.New("cmd类对象为nil")
	}
	return c.Cmd父类.StdoutPipe()
}

// I取Stderr管道 返回一个管道，该管道将在命令启动时连接到命令的标准错误。
//
// 方法返回一个在命令Start后与命令标准错误输出关联的管道。Wait方法获知命令结束后会关闭这个管道，一般不需要显式的关闭该管道。
// 但是在从管道读取完全部数据之前调用Wait是错误的；同样使用StderrPipe方法时调用Run函数也是错误的。请参照StdoutPipe的例子。
func (c *Cmd) I取Stderr管道() (io.ReadCloser, error) {
	if c == nil {
		return nil, errors.New("cmd类对象为nil")
	}
	return c.Cmd父类.StderrPipe() //
}

// I取环境变量数组 返回当前配置的命令运行环境的副本。
func (c *Cmd) I取环境变量数组() []string {
	if c == nil {
		return nil
	}
	return c.Cmd父类.Environ()
}
