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
