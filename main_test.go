package main

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/pkg/diff"
	"github.com/scylladb/go-set"
	"github.com/scylladb/go-set/strset"
)

func TestSuccess(t *testing.T) {
	testCases := []struct {
		planPath     string
		wantUpPath   string
		wantDownPath string
	}{
		{
			"testdata/01_exact-match.plan.txt",
			"testdata/01_exact-match.up.sh",
			"testdata/01_exact-match.down.sh",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.planPath, func(t *testing.T) {
			wantUp, err := ioutil.ReadFile(tc.wantUpPath)
			if err != nil {
				t.Fatalf("reading want up file: %v", err)
			}
			wantDown, err := ioutil.ReadFile(tc.wantDownPath)
			if err != nil {
				t.Fatalf("reading want down file: %v", err)
			}

			tmpDir, err := ioutil.TempDir("", "terravalet")
			if err != nil {
				t.Fatalf("creating temporary dir: %v", err)
			}
			defer os.RemoveAll(tmpDir)

			tmpUpPath := tmpDir + "/up"
			tmpDownPath := tmpDir + "/down"

			args := []string{"-plan", tc.planPath, "-up", tmpUpPath, "-down", tmpDownPath}
			if err := run(args); err != nil {
				t.Fatalf("\ngot:  %q\nwant: no error", err)
			}

			tmpUp, err := ioutil.ReadFile(tmpUpPath)
			if err != nil {
				t.Fatalf("reading tmp up file: %v", err)
			}
			tmpDown, err := ioutil.ReadFile(tmpDownPath)
			if err != nil {
				t.Fatalf("reading tmp down file: %v", err)
			}

			if !bytes.Equal(tmpUp, wantUp) {
				var outDiff bytes.Buffer
				diff.Text("got", tc.wantUpPath, tmpUp, wantUp, &outDiff)
				t.Errorf("up script: got the following differences:\n%v", outDiff.String())
			}
			if !bytes.Equal(tmpDown, wantDown) {
				var outDiff bytes.Buffer
				diff.Text("got", tc.wantDownPath, tmpDown, wantDown, &outDiff)
				t.Errorf("down script: got the following differences:\n%v", outDiff.String())
			}
		})
	}
}

func TestFailure(t *testing.T) {
	testCases := []struct {
		args      []string
		wantError string
	}{
		{
			[]string{},
			"missing value for -plan",
		},
		{
			[]string{"-plan=nonexisting", "-up=up", "-down=down"},
			"opening the terraform plan file: open nonexisting: no such file or directory",
		},
	}

	for _, tc := range testCases {
		t.Run(strings.Join(tc.args, "_"), func(t *testing.T) {
			err := run(tc.args)

			if err == nil {
				t.Fatalf("\ngot:  no error\nwant: %q", tc.wantError)
			}
			if err.Error() != tc.wantError {
				t.Fatalf("\ngot:  %q\nwant: %q", err, tc.wantError)
			}
		})
	}
}

var cmpOpt = cmp.Comparer(func(s1, s2 *strset.Set) bool {
	return s1.IsEqual(s2)
})

func TestParseSuccess(t *testing.T) {
	testCases := []struct {
		description string
		line        string
		wantCreate  *strset.Set
		wantDestroy *strset.Set
	}{
		{
			"destroyed is recorded",
			"  # aws_instance.bar will be destroyed",
			set.NewStringSet(),
			set.NewStringSet("aws_instance.bar"),
		},
		{
			"created is recorded",
			"  # aws_instance.bar will be created",
			set.NewStringSet("aws_instance.bar"),
			set.NewStringSet(),
		},
		{
			"read is skipped",
			"  # data.foo.bar will be read during apply",
			set.NewStringSet(),
			set.NewStringSet(),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			rd := strings.NewReader(tc.line)

			gotCreate, gotDestroy, err := parse(rd)

			if err != nil {
				t.Fatalf("\ngot:  %q\nwant: no error", err)
			}
			if diff := cmp.Diff(tc.wantCreate, gotCreate, cmpOpt); diff != "" {
				t.Errorf("\ncreate: mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tc.wantDestroy, gotDestroy, cmpOpt); diff != "" {
				t.Errorf("\ndestroy: mismatch (-want +got):\n%s", diff)
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
				t.Fatalf("\ngot:  no error\nwant: %q", tc.wantError)
			}
			if err.Error() != tc.wantError.Error() {
				t.Fatalf("\ngot:  %q\nwant: %q", err, tc.wantError)
			}
		})
	}
}

func TestMatchSuccess(t *testing.T) {
	testCases := []struct {
		description     string
		create          *strset.Set
		destroy         *strset.Set
		wantUpMatches   map[string]string
		wantDownMatches map[string]string
	}{
		{"increase depth, len 1",
			set.NewStringSet("a.b"),
			set.NewStringSet("b"),
			map[string]string{"b": "a.b"},
			map[string]string{"a.b": "b"},
		},
		{"decrease depth, len 1",
			set.NewStringSet("b"),
			set.NewStringSet("a.b"),
			map[string]string{"a.b": "b"},
			map[string]string{"b": "a.b"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			gotUpMatches, gotDownMatches, err := match(tc.create, tc.destroy)

			if err != nil {
				t.Fatalf("\ngot:  %q\nwant: no error", err)
			}
			if diff := cmp.Diff(tc.wantUpMatches, gotUpMatches); diff != "" {
				t.Errorf("upMatches: mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tc.wantDownMatches, gotDownMatches); diff != "" {
				t.Errorf("downMatches: mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestMatchFailure(t *testing.T) {
	testCases := []struct {
		description string
		create      *strset.Set
		destroy     *strset.Set
		wantErr     error
	}{
		{"len(create) == len(destroy), no match",
			set.NewStringSet("a.b"),
			set.NewStringSet("j.k"),
			fmt.Errorf("1 unmatched create 1 unmatched destroy"),
		},
		{"len(create) > len(destroy), match",
			set.NewStringSet("a.b", "a.j.k"),
			set.NewStringSet("j.k"),
			fmt.Errorf("1 unmatched create"),
		},
		{"len(create) < len(destroy), match",
			set.NewStringSet("a.b"),
			set.NewStringSet("j.k", "x.a.b"),
			fmt.Errorf("1 unmatched destroy"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			_, _, gotErr := match(tc.create, tc.destroy)

			if gotErr == nil {
				t.Fatalf("\ngot:  no error\nwant: %q", tc.wantErr)
			}
			if gotErr.Error() != tc.wantErr.Error() {
				t.Fatalf("\ngot:  %q\nwant: %q", gotErr, tc.wantErr)
			}
		})
	}
}
