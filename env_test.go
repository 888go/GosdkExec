// Copyright 2017 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cmd类

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

// dedupEnvCase 是dudupEnv，带有用于测试的case选项。
// 如果caseInsensitive为true，则忽略键的大小写。
// 如果nulOK为false，则允许包含NUL字符的项。
func dedupEnvCase(caseInsensitive, nulOK bool, env []string) ([]string, error) {
	//以相反的顺序构造输出，以保留每个键的最后一次出现。
	var err error
	out := make([]string, 0, len(env))
	saw := make(map[string]bool, len(env))
	for n := len(env); n > 0; n-- {
		kv := env[n-1]

		//拒绝环境变量中的NUL以防止安全问题 (#56284);
		// 除计划9外，计划9使用NUL作为os.PathListSeparator (#56544).
		if !nulOK && strings.IndexByte(kv, 0) != -1 {
			err = errors.New("exec: environment variable contains NUL")
			continue
		}

		i := strings.Index(kv, "=")
		if i == 0 {
			// 我们在实践中观察到，Windows上的键只有一个前导“=”。
			// TODO(#49886): 我们是否应该只使用开头的“=”作为一部分
			// 或者解析任意多个键，直到非“=”？
			i = strings.Index(kv[1:], "=") + 1
		}
		if i < 0 {
			if kv != "" {
				//该条目的格式不是“key=value”（按要求）。
				// 暂时保持原样。
				// TODO(#52436): 我们应该剥离或拒绝这些虚假条目吗？
				out = append(out, kv)
			}
			continue
		}
		k := kv[:i]
		if caseInsensitive {
			k = strings.ToLower(k)
		}
		if saw[k] {
			continue
		}

		saw[k] = true
		out = append(out, kv)
	}

	// 现在反转切片以恢复原始顺序。
	for i := 0; i < len(out)/2; i++ {
		j := len(out) - i - 1
		out[i], out[j] = out[j], out[i]
	}

	return out, err
}
func TestDedupEnv(t *testing.T) {
	tests := []struct {
		noCase  bool
		nulOK   bool
		in      []string
		want    []string
		wantErr bool
	}{
		{
			noCase: true,
			in:     []string{"k1=v1", "k2=v2", "K1=v3"},
			want:   []string{"k2=v2", "K1=v3"},
		},
		{
			noCase: false,
			in:     []string{"k1=v1", "K1=V2", "k1=v3"},
			want:   []string{"K1=V2", "k1=v3"},
		},
		{
			in:   []string{"=a", "=b", "foo", "bar"},
			want: []string{"=b", "foo", "bar"},
		},
		{
			// #49886: 保留带有前导“=”符号的奇怪Windows键。
			noCase: true,
			in:     []string{`=C:=C:\golang`, `=D:=D:\tmp`, `=D:=D:\`},
			want:   []string{`=C:=C:\golang`, `=D:=D:\`},
		},
		{
			// #52436: 保留无效的键值条目（暂时）。
			// （可能会过滤掉它们，或者在某个时候出错。）
			in:   []string{"dodgy", "entries"},
			want: []string{"dodgy", "entries"},
		},
		{
			// 筛选出包含NUL的条目。
			in:      []string{"A=a\x00b", "B=b", "C\x00C=c"},
			want:    []string{"B=b"},
			wantErr: true,
		},
		{
			// 计划9需要使用NUL保存环境变量(#56544).
			nulOK: true,
			in:    []string{"path=one\x00two"},
			want:  []string{"path=one\x00two"},
		},
	}
	for _, tt := range tests {
		got, err := dedupEnvCase(tt.noCase, tt.nulOK, tt.in)
		if !reflect.DeepEqual(got, tt.want) || (err != nil) != tt.wantErr {
			t.Errorf("Dedup(%v, %q) = %q, %v; want %q, error:%v", tt.noCase, tt.in, got, err, tt.want, tt.wantErr)
		}
	}
}
