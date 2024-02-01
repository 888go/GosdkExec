// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cmd类

import (
	"bytes"
	"io"
	"strconv"
	"testing"
)

// prefixSuffixSaver 是一个io.Writer，它保留了写入它的前N个字节和最后N个字节。bytes（）方法用一条错误消息重新构建它。
type prefixSuffixSaver struct {
	N         int //前缀或后缀的最大大小
	prefix    []byte
	suffix    []byte //环形缓冲区一次 len(suffix) == N
	suffixOff int    // 要写入的偏移量 suffix
	skipped   int64

	// TODO(bradfitz): we could keep one large []byte and use part of it for
	// the prefix, reserve space for the '... Omitting N bytes ...' message,
	// then the ring buffer suffix, and just rearrange the ring buffer
	// suffix when Bytes() is called, but it doesn't seem worth it for
	// now just for error messages. It's only ~64KB anyway.
}

func (w *prefixSuffixSaver) Write(p []byte) (n int, err error) {
	lenp := len(p)
	p = w.fill(&w.prefix, p)

	// Only keep the last w.N bytes of suffix data.
	if overage := len(p) - w.N; overage > 0 {
		p = p[overage:]
		w.skipped += int64(overage)
	}
	p = w.fill(&w.suffix, p)

	// w.suffix is full now if p is non-empty. Overwrite it in a circle.
	for len(p) > 0 { // 0, 1, or 2 iterations.
		n := copy(w.suffix[w.suffixOff:], p)
		p = p[n:]
		w.skipped += int64(n)
		w.suffixOff += n
		if w.suffixOff == w.N {
			w.suffixOff = 0
		}
	}
	return lenp, nil
}

// fill 将p的最大len（p）字节附加到dst，这样dst不会增长到大于w.N。它返回未附加的p后缀。
func (w *prefixSuffixSaver) fill(dst *[]byte, p []byte) (pRemain []byte) {
	if remain := w.N - len(*dst); remain > 0 {
		add := minInt(len(p), remain)
		*dst = append(*dst, p[:add]...)
		p = p[add:]
	}
	return p
}
func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (w *prefixSuffixSaver) Bytes() []byte {
	if w.suffix == nil {
		return w.prefix
	}
	if w.skipped == 0 {
		return append(w.prefix, w.suffix...)
	}
	var buf bytes.Buffer
	buf.Grow(len(w.prefix) + len(w.suffix) + 50)
	buf.Write(w.prefix)
	buf.WriteString("\n... omitting ")
	buf.WriteString(strconv.FormatInt(w.skipped, 10))
	buf.WriteString(" bytes ...\n")
	buf.Write(w.suffix[w.suffixOff:])
	buf.Write(w.suffix[:w.suffixOff])
	return buf.Bytes()
}
func TestPrefixSuffixSaver(t *testing.T) {
	tests := []struct {
		N      int
		writes []string
		want   string
	}{
		{
			N:      2,
			writes: nil,
			want:   "",
		},
		{
			N:      2,
			writes: []string{"a"},
			want:   "a",
		},
		{
			N:      2,
			writes: []string{"abc", "d"},
			want:   "abcd",
		},
		{
			N:      2,
			writes: []string{"abc", "d", "e"},
			want:   "ab\n... omitting 1 bytes ...\nde",
		},
		{
			N:      2,
			writes: []string{"ab______________________yz"},
			want:   "ab\n... omitting 22 bytes ...\nyz",
		},
		{
			N:      2,
			writes: []string{"ab_______________________y", "z"},
			want:   "ab\n... omitting 23 bytes ...\nyz",
		},
	}
	for i, tt := range tests {
		w := &prefixSuffixSaver{N: tt.N}
		for _, s := range tt.writes {
			n, err := io.WriteString(w, s)
			if err != nil || n != len(s) {
				t.Errorf("%d. WriteString(%q) = %v, %v; want %v, %v", i, s, n, err, len(s), nil)
			}
		}
		if got := string(w.Bytes()); got != tt.want {
			t.Errorf("%d. Bytes = %q; want %q", i, got, tt.want)
		}
	}
}
