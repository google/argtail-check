// Copyright 2017 Google Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package main

import (
	"io/ioutil"
	"path"
	"testing"
)

func TestFiles(t *testing.T) {
	for _, test := range []struct {
		fn  string
		err error
	}{
		{
			fn: "normal",
		},
		{
			fn:  "nomain",
			err: errFuncNotFound,
		},
		{
			fn:  "noparse",
			err: errNoFlagParseCalls,
		},
		{
			fn:  "already_checking",
			err: errAlreadyChecking,
		},
	} {
		const basePath = "testdata"

		var got string
		{
			fn := path.Join(basePath, test.fn+"_in.go")
			s, err := ioutil.ReadFile(fn)
			if err != nil {
				t.Fatalf("%s: %v", test.fn, err)
			}
			got, err = fix(fn, string(s))
			if err != test.err {
				t.Fatalf("%s: Got error %q, want %q", test.fn, err, test.err)
			}
			if err != nil {
				continue
			}
		}

		var want string
		{
			fn := path.Join(basePath, test.fn+"_out.go")
			wantb, err := ioutil.ReadFile(fn)
			if err != nil {
				t.Fatalf("%s: %v", test.fn, err)
			}
			want = string(wantb)
		}

		if got != want {
			t.Errorf("Got:\n-----\n%s\n-----\nWant:\n-----\n%s\n-----\n", got, want)
		}
	}
}
