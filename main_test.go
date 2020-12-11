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
		description  string
		options      []string
		planPath     string
		wantUpPath   string
		wantDownPath string
	}{
		{
			"exact match",
			[]string{},
			"testdata/01_exact-match.plan.txt",
			"testdata/01_exact-match.up.sh",
			"testdata/01_exact-match.down.sh",
		},
		{
			"q-gram fuzzy match simple",
			[]string{"-fuzzy-match"},
			"testdata/02_fuzzy-match.plan.txt",
			"testdata/02_fuzzy-match.up.sh",
			"testdata/02_fuzzy-match.down.sh",
		},
		{
			"q-gram fuzzy match complicated",
			[]string{"-fuzzy-match"},
			"testdata/03_fuzzy-match.plan.txt",
			"testdata/03_fuzzy-match.up.sh",
			"testdata/03_fuzzy-match.down.sh",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
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
			args = append(args, tc.options...)

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
				t.Errorf("\nup script: got the following differences:\n%v", outDiff.String())
			}
			if !bytes.Equal(tmpDown, wantDown) {
				var outDiff bytes.Buffer
				diff.Text("got", tc.wantDownPath, tmpDown, wantDown, &outDiff)
				t.Errorf("\ndown script: got the following differences:\n%v", outDiff.String())
			}
		})
	}
}

func TestFailure(t *testing.T) {
	testCases := []struct {
		planPath  string
		wantError error
	}{
		{
			"",
			fmt.Errorf("missing value for -plan"),
		},
		{
			"nonexisting",
			fmt.Errorf("opening the terraform plan file: open nonexisting: no such file or directory"),
		},
		{
			"testdata/02_fuzzy-match.plan.txt",
			fmt.Errorf(`matchExact:
unmatched create:
  aws_route53_record.localhostnames_public["artifactory"]
  aws_route53_record.loopback["artifactory"]
  aws_route53_record.private["artifactory"]
unmatched destroy:
  aws_route53_record.artifactory
  aws_route53_record.artifactory_loopback
  aws_route53_record.artifactory_private`),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.planPath, func(t *testing.T) {
			tmpDir, err := ioutil.TempDir("", "terravalet")
			if err != nil {
				t.Fatalf("creating temporary dir: %v", err)
			}
			defer os.RemoveAll(tmpDir)

			tmpUpPath := tmpDir + "/up"
			tmpDownPath := tmpDir + "/down"

			args := []string{"-plan", tc.planPath, "-up", tmpUpPath, "-down", tmpDownPath}

			err = run(args)

			if err == nil {
				t.Fatalf("\ngot:  no error\nwant: %q", err)
			}
			if err.Error() != tc.wantError.Error() {
				t.Fatalf("\ngot:  %q\nwant: %q", err, tc.wantError)
			}
		})
	}
}

// Used to compare sets.
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

func TestMatchExactZeroUnmatched(t *testing.T) {
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
			gotUpMatches, gotDownMatches := matchExact(tc.create, tc.destroy)

			if diff := cmp.Diff(tc.wantUpMatches, gotUpMatches); diff != "" {
				t.Errorf("\nupMatches: mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tc.wantDownMatches, gotDownMatches); diff != "" {
				t.Errorf("\ndownMatches: mismatch (-want +got):\n%s", diff)
			}
			if got := tc.create.Size(); got != 0 {
				t.Errorf("\nsize(create): got: %d; want: 0", got)
			}
			if got := tc.destroy.Size(); got != 0 {
				t.Errorf("\nsize(destroy): got: %d; want: 0", got)
			}
		})
	}
}

func TestMatchExactSomeUnmatched(t *testing.T) {
	testCases := []struct {
		description string
		create      *strset.Set
		destroy     *strset.Set
		wantCreate  *strset.Set
		wantDestroy *strset.Set
	}{
		{"len(create) == len(destroy), no match",
			set.NewStringSet("a.b"),
			set.NewStringSet("j.k"),
			set.NewStringSet("a.b"),
			set.NewStringSet("j.k"),
		},
		{"len(create) > len(destroy), match",
			set.NewStringSet("a.b", "a.j.k"),
			set.NewStringSet("j.k"),
			set.NewStringSet("a.b"),
			set.NewStringSet(),
		},
		{"len(create) < len(destroy), match",
			set.NewStringSet("a.b"),
			set.NewStringSet("j.k", "x.a.b"),
			set.NewStringSet(),
			set.NewStringSet("j.k"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			matchExact(tc.create, tc.destroy)

			if diff := cmp.Diff(tc.wantCreate, tc.create, cmpOpt); diff != "" {
				t.Errorf("\nUnmatched create: (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tc.wantDestroy, tc.destroy, cmpOpt); diff != "" {
				t.Errorf("\nUnmatched destroy (-want +got):\n%s", diff)
			}
		})
	}
}

func TestMatchFuzzyZeroUnmatched(t *testing.T) {
	testCases := []struct {
		description     string
		create          *strset.Set
		destroy         *strset.Set
		wantUpMatches   map[string]string
		wantDownMatches map[string]string
	}{
		{"1 fuzzy match",
			set.NewStringSet(`foo.loopback["bar"]`),
			set.NewStringSet(`foo.bar_loopback`),
			map[string]string{`foo.bar_loopback`: `foo.loopback["bar"]`},
			map[string]string{`foo.loopback["bar"]`: `foo.bar_loopback`},
		},
		{"3 fuzzy matches",
			set.NewStringSet(
				`foo.loopback["bar"]`,
				`foo.private["bar"]`,
				`foo.public["bar"]`),
			set.NewStringSet(
				`foo.bar_loopback`,
				`foo.bar_private`,
				`foo.bar`),
			map[string]string{
				`foo.bar_loopback`: `foo.loopback["bar"]`,
				`foo.bar_private`:  `foo.private["bar"]`,
				`foo.bar`:          `foo.public["bar"]`},
			map[string]string{
				`foo.loopback["bar"]`: `foo.bar_loopback`,
				`foo.private["bar"]`:  `foo.bar_private`,
				`foo.public["bar"]`:   `foo.bar`},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {

			gotUpMatches, gotDownMatches := matchFuzzy(tc.create, tc.destroy)

			if diff := cmp.Diff(tc.wantUpMatches, gotUpMatches); diff != "" {
				t.Errorf("\nupMatches: mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tc.wantDownMatches, gotDownMatches); diff != "" {
				t.Errorf("\ndownMatches: mismatch (-want +got):\n%s", diff)
			}
			if got := tc.create.Size(); got != 0 {
				t.Errorf("\nsize(create): got: %d; want: 0", got)
			}
			if got := tc.destroy.Size(); got != 0 {
				t.Errorf("\nsize(destroy): got: %d; want: 0", got)
			}
		})
	}
}
