package main

import (
	"bytes"
	"errors"
	"io/ioutil"
	"strings"
	"testing"

	"github.com/pkg/diff"
)

func TestSuccess(t *testing.T) {
	testCases := []struct {
		planPath       string
		wantScriptPath string
	}{
		{
			"testdata/plan-synthetic-01.txt",
			"testdata/001_synthetic.up.sh",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.planPath, func(t *testing.T) {
			want, err := ioutil.ReadFile(tc.wantScriptPath)
			if err != nil {
				t.Fatalf("reading want file %q: error: %v", tc.wantScriptPath, err)
			}

			var got bytes.Buffer
			if err := run([]string{tc.planPath}, &got); err != nil {
				t.Fatalf("got error: %v; want: no error", err)
			}

			if !bytes.Equal(got.Bytes(), want) {
				var outDiff bytes.Buffer
				diff.Text("got", tc.wantScriptPath, got.Bytes(), want, &outDiff)
				t.Fatalf("got the following differences:\n%v", outDiff.String())
			}
		})
	}
}

func TestFailure(t *testing.T) {
	testCases := []struct {
		planPath  string
		wantError string
	}{
		{
			"testdata/plan-non-existing.txt",
			"reading the tf plan file: open testdata/plan-non-existing.txt: no such file or directory",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.planPath, func(t *testing.T) {
			var got bytes.Buffer

			err := run([]string{tc.planPath}, &got)

			if err == nil {
				t.Fatalf("\ngot:  no error\nwant: %v", tc.wantError)
			}
			if err.Error() != tc.wantError {
				t.Fatalf("\ngot:  %v\nwant: %v", err, tc.wantError)
			}
		})
	}
}

func TestParseSuccess(t *testing.T) {
	testCases := []struct {
		description string
		line        string
		wantCreate  []string
		wantDestroy []string
	}{
		{
			"destroyed is recorded",
			"  # aws_instance.bar will be destroyed",
			[]string{},
			[]string{"aws_instance.bar"},
		},
		{
			"created is recorded",
			"  # aws_instance.bar will be created",
			[]string{"aws_instance.bar"},
			[]string{},
		},
		{
			"read is skipped",
			"  # data.foo.bar will be read during apply",
			[]string{},
			[]string{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			rd := strings.NewReader(tc.line)

			gotCreate, gotDestroy, err := parse(rd)

			if err != nil {
				t.Fatalf("\ngot:  %v\nwant: no error", err)
			}
			if !stringEqual(gotCreate, tc.wantCreate) {
				t.Errorf("\ngotCreate:  %v\nwantCreate: %v", gotCreate, tc.wantCreate)
			}
			if !stringEqual(gotDestroy, tc.wantDestroy) {
				t.Errorf("\ngotDestroy:  %v\nwantDestroy: %v", gotDestroy, tc.wantDestroy)
			}
		})
	}
}

func TestParseFailure(t *testing.T) {
	testCases := []struct {
		description string
		line        string
		wantError   error
	}{
		{
			"vaporized is not an expected action",
			"  # aws_instance.bar will be vaporized",
			errors.New(`line "  # aws_instance.bar will be vaporized", unexpected action "vaporized"`),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			rd := strings.NewReader(tc.line)

			_, _, err := parse(rd)

			if err == nil {
				t.Fatalf("\ngot:  no error\nwant: %v", tc.wantError)
			}
			if err.Error() != tc.wantError.Error() {
				t.Fatalf("\ngot:  %v\nwant: %v", err, tc.wantError)
			}
		})
	}
}

func stringEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i, v := range a {
		if v != b[i] {
			return false
		}
	}
	return true
}
