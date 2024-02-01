// Copyright 2017 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cmd类

// skipStdinCopyError 可选地指定一个函数，该函数报告是否应忽略提供的stdin复制错误。
//func skipStdinCopyError(err error) bool {
//	// 如果程序成功完成，则忽略复制到stdin的ERROR_BROKEN_PIPE和ERROR_NO_DATA错误。见第20445期.
//	const _ERROR_NO_DATA = syscall.Errno(0xe8)
//	pe, ok := err.(*fs.PathError)
//	return ok &&
//		pe.Op == "write" && pe.Path == "|1" &&
//		(pe.Err == syscall.ERROR_BROKEN_PIPE || pe.Err == _ERROR_NO_DATA)
//}
