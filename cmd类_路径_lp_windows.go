// Copyright 2010 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cmd类

import (
	"errors"
	"os/exec"
)

// ErrNotFound 是路径搜索未能找到可执行文件时产生的错误。
var ErrNotFound = errors.New("executable file not found in %PATH%")

// I查找路径 在环境变量PATH指定的目录中搜索可执行文件，如file中有斜杠，则只在当前目录搜索。
// 返回完整路径或者相对于当前目录的一个相对路径。
// 一旦成功，结果就是一条绝对的路径。
//
// 在旧版本的Go中，LookPath可以返回相对于当前目录的路径。
// 从Go 1.19开始，LookPath将返回该路径以及满足 errors.Is(err, ErrDot) 的错误。有关详细信息，请参阅软件包文档。
func I查找路径(file string) (string, error) {
	return exec.LookPath(file)
}
