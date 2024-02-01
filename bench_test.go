// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cmd类

import (
	"testing"
)

func BenchmarkExecHostname(b *testing.B) {
	b.ReportAllocs()
	path, err := I查找路径("hostname")
	if err != nil {
		b.Fatalf("找不到主机名: %v", err)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := I设置命令(path).I运行(); err != nil {
			b.Fatalf("主机名: %v", err)
		}
	}
}
