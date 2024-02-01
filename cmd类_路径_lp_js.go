// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build js && wasm

package cmd类

import (
	"errors"
)

// ErrNotFound 是路径搜索未能找到可执行文件时产生的错误。
var ErrNotFound = errors.New("executable file not found in $PATH")

// I查找路径 在环境变量PATH指定的目录中搜索可执行文件，如file中有斜杠，则只在当前目录搜索。
// 返回完整路径或者相对于当前目录的一个相对路径。
func I查找路径(文件 string) (string, error) {
	// Wasm不能执行进程，所以就好像根本没有可执行文件一样。
	return "", &Error{文件, ErrNotFound}
}
