// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cmd类

// skipStdinCopyError 可选地指定一个函数，该函数报告是否应忽略提供的stdin复制错误。
//func skipStdinCopyError(err error) bool {
//	// 如果程序成功完成，则忽略复制到stdin的挂起错误，否则将忽略。
//	// See Issue 35753.
//	pe, ok := err.(*fs.PathError)
//	return ok &&
//		pe.Op == "write" && pe.Path == "|1" &&
//		pe.Err.Error() == "i/o on hungup channel"
//}
