// Copyright 2020 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build linux && cgo

// 在使用glibc的系统上，调用malloc可以创建一个新的竞技场，而创建新的竞技场可以读取/sys/devices/system/cpu/online。
// 如果我们使用cgo，我们将在创建新线程时调用malloc。
// 如果我们在检查打开的文件描述符时创建一个新的线程来创建新的竞技场并打开/sys文件，这可能会破坏TestExtraFiles.
//通过提前创建线程来解决问题。
// See issue 25628.

package cmd类_test

import (
	"os"
	"sync"
	"syscall"
	"time"
)

func init() {
	if os.Getenv("GO_EXEC_TEST_PID") == "" {
		return
	}

	// 启动一些线程。10是任意的，但足以确保代码本身不必创建任何线程。
	//特别是，这应该大于垃圾收集器可能创建的线程数。
	const threads = 10

	var wg sync.WaitGroup
	wg.Add(threads)
	ts := syscall.NsecToTimespec((100 * time.Microsecond).Nanoseconds())
	for i := 0; i < threads; i++ {
		go func() {
			defer wg.Done()
			syscall.Nanosleep(&ts, nil)
		}()
	}
	wg.Wait()
}
