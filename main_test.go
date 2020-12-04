package main

import (
	"bytes"
	"errors"
	"io/ioutil"
	"os"
	"strings"
	"testing"

	"github.com/pkg/diff"
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
				t.Fatalf("got error: %v; want: no error", err)
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
