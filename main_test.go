package main

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/scylladb/go-set"
	"github.com/scylladb/go-set/strset"
)

// (Ab)use the "Example" feature of the testing package to assert on the
// output of the program. See https://pkg.go.dev/testing#hdr-Examples
func Example_version() {
	os.Args = []string{"terravalet", "version"}
	if err := run(); err != nil {
		panic(err)
	}
	// Output:
	// terravalet unknown
}

func TestParseSuccess(t *testing.T) {
	testCases := []struct {
		name        string
		line        string
		wantCreate  *strset.Set
		wantDestroy *strset.Set
	}{
		{
			name:        "destroyed is recorded",
			line:        "  # aws_instance.bar will be destroyed",
			wantCreate:  set.NewStringSet(),
			wantDestroy: set.NewStringSet("aws_instance.bar"),
		},
		{
			name:        "created is recorded",
			line:        "  # aws_instance.bar will be created",
			wantCreate:  set.NewStringSet("aws_instance.bar"),
			wantDestroy: set.NewStringSet(),
		},
		{
			name:        "read is skipped",
			line:        "  # data.foo.bar will be read during apply",
			wantCreate:  set.NewStringSet(),
			wantDestroy: set.NewStringSet(),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			rd := strings.NewReader(tc.line)

			haveCreate, haveDestroy, err := parse(rd)

			if err != nil {
				t.Fatalf("\nhave: %q\nwant: no error", err)
			}
			if diff := cmp.Diff(tc.wantCreate, haveCreate, setCmp); diff != "" {
				t.Errorf("\ncreate: mismatch (-want +have):\n%s", diff)
			}
			if diff := cmp.Diff(tc.wantDestroy, haveDestroy, setCmp); diff != "" {
				t.Errorf("\ndestroy: mismatch (-want +have):\n%s", diff)
			}
		})
	}
}

func TestParseFailure(t *testing.T) {
	testCases := []struct {
		name    string
		line    string
		wantErr string
	}{
		{
			name:    "vaporized is not an expected action",
			line:    "  # aws_instance.bar will be vaporized",
			wantErr: `line "  # aws_instance.bar will be vaporized", unexpected action "vaporized"`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			rd := strings.NewReader(tc.line)

			_, _, err := parse(rd)

			if err == nil {
				t.Fatalf("\nhave: no error\nwant: %q", tc.wantErr)
			}
			if diff := cmp.Diff(tc.wantErr, err.Error()); diff != "" {
				t.Errorf("error message mismatch (-want +have):\n%s", diff)
			}
		})
	}
}

func TestMatchExactZeroUnmatched(t *testing.T) {
	testCases := []struct {
		name            string
		create          *strset.Set
		destroy         *strset.Set
		wantUpMatches   map[string]string
		wantDownMatches map[string]string
	}{
		{
			name:            "increase depth, len 1",
			create:          set.NewStringSet("a.b"),
			destroy:         set.NewStringSet("b"),
			wantUpMatches:   map[string]string{"b": "a.b"},
			wantDownMatches: map[string]string{"a.b": "b"},
		},
		{
			name:            "decrease depth, len 1",
			create:          set.NewStringSet("b"),
			destroy:         set.NewStringSet("a.b"),
			wantUpMatches:   map[string]string{"a.b": "b"},
			wantDownMatches: map[string]string{"b": "a.b"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			haveUpMatches, haveDownMatches := matchExact(tc.create, tc.destroy)

			if diff := cmp.Diff(tc.wantUpMatches, haveUpMatches); diff != "" {
				t.Errorf("\nupMatches: mismatch (-want +have):\n%s", diff)
			}
			if diff := cmp.Diff(tc.wantDownMatches, haveDownMatches); diff != "" {
				t.Errorf("\ndownMatches: mismatch (-want +have):\n%s", diff)
			}
			if have := tc.create.Size(); have != 0 {
				t.Errorf("\nsize(create): have: %d; want: 0", have)
			}
			if have := tc.destroy.Size(); have != 0 {
				t.Errorf("\nsize(destroy): have: %d; want: 0", have)
			}
		})
	}
}

func TestMatchExactSomeUnmatched(t *testing.T) {
	testCases := []struct {
		name        string
		create      *strset.Set
		destroy     *strset.Set
		wantCreate  *strset.Set
		wantDestroy *strset.Set
	}{
		{
			name:        "len(create) == len(destroy), no match",
			create:      set.NewStringSet("a.b"),
			destroy:     set.NewStringSet("j.k"),
			wantCreate:  set.NewStringSet("a.b"),
			wantDestroy: set.NewStringSet("j.k"),
		},
		{
			name:        "len(create) > len(destroy), match",
			create:      set.NewStringSet("a.b", "a.j.k"),
			destroy:     set.NewStringSet("j.k"),
			wantCreate:  set.NewStringSet("a.b"),
			wantDestroy: set.NewStringSet(),
		},
		{
			name:        "len(create) < len(destroy), match",
			create:      set.NewStringSet("a.b"),
			destroy:     set.NewStringSet("j.k", "x.a.b"),
			wantCreate:  set.NewStringSet(),
			wantDestroy: set.NewStringSet("j.k"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			matchExact(tc.create, tc.destroy)

			if diff := cmp.Diff(tc.wantCreate, tc.create, setCmp); diff != "" {
				t.Errorf("\nUnmatched create: (-want +have):\n%s", diff)
			}
			if diff := cmp.Diff(tc.wantDestroy, tc.destroy, setCmp); diff != "" {
				t.Errorf("\nUnmatched destroy (-want +have):\n%s", diff)
			}
		})
	}
}

func TestMatchFuzzyZeroUnmatched(t *testing.T) {
	testCases := []struct {
		name            string
		create          *strset.Set
		destroy         *strset.Set
		wantUpMatches   map[string]string
		wantDownMatches map[string]string
	}{
		{
			name:            "1 fuzzy match",
			create:          set.NewStringSet(`foo.loopback["bar"]`),
			destroy:         set.NewStringSet(`foo.bar_loopback`),
			wantUpMatches:   map[string]string{`foo.bar_loopback`: `foo.loopback["bar"]`},
			wantDownMatches: map[string]string{`foo.loopback["bar"]`: `foo.bar_loopback`},
		},
		{
			name: "3 fuzzy matches",
			create: set.NewStringSet(
				`foo.loopback["bar"]`,
				`foo.private["bar"]`,
				`foo.public["bar"]`),
			destroy: set.NewStringSet(
				`foo.bar_loopback`,
				`foo.bar_private`,
				`foo.bar`),
			wantUpMatches: map[string]string{
				`foo.bar_loopback`: `foo.loopback["bar"]`,
				`foo.bar_private`:  `foo.private["bar"]`,
				`foo.bar`:          `foo.public["bar"]`},
			wantDownMatches: map[string]string{
				`foo.loopback["bar"]`: `foo.bar_loopback`,
				`foo.private["bar"]`:  `foo.bar_private`,
				`foo.public["bar"]`:   `foo.bar`},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {

			haveUpMatches, haveDownMatches, err := matchFuzzy(tc.create, tc.destroy)
			if err != nil {
				t.Fatalf("have: %s; want: no error", err)
			}

			if diff := cmp.Diff(tc.wantUpMatches, haveUpMatches); diff != "" {
				t.Errorf("\nupMatches: mismatch (-want +have):\n%s", diff)
			}
			if diff := cmp.Diff(tc.wantDownMatches, haveDownMatches); diff != "" {
				t.Errorf("\ndownMatches: mismatch (-want +have):\n%s", diff)
			}
			if have := tc.create.Size(); have != 0 {
				t.Errorf("\nsize(create): have: %d; want: 0", have)
			}
			if have := tc.destroy.Size(); have != 0 {
				t.Errorf("\nsize(destroy): have: %d; want: 0", have)
			}
		})
	}
}

func TestMatchFuzzyError(t *testing.T) {
	create := set.NewStringSet(`abcde`, `abdecde`)
	destroy := set.NewStringSet(`abdcde`, `hfjabd`)
	_, _, err := matchFuzzy(create, destroy)
	if err == nil {
		t.Fatalf("have: no error; want: an ambiguous migration error")
	}

	haveMsg := err.Error()
	var msg string

	want := "ambiguous migration:"
	if !strings.HasPrefix(haveMsg, want) {
		msg += fmt.Sprintf("error message does not start with %q\n", want)
	}

	want = "{abcde} -> {abdcde}"
	if !strings.Contains(haveMsg, want) {
		msg += fmt.Sprintf("error message does not contain %q", want)
	}

	want = "{abdecde} -> {abdcde}"
	if !strings.Contains(haveMsg, want) {
		msg += fmt.Sprintf("error message does not contain %q", want)
	}

	if msg != "" {
		t.Fatal(msg)
	}
}

// Used to compare sets.
var setCmp = cmp.Comparer(func(s1, s2 *strset.Set) bool {
	return s1.IsEqual(s2)
})

func runSuccess(t *testing.T, args []string, wantUpPath string, wantDownPath string) {
	wantUp, err := os.ReadFile(wantUpPath)
	if err != nil {
		t.Fatalf("reading want up file: %v", err)
	}
	wantDown, err := os.ReadFile(wantDownPath)
	if err != nil {
		t.Fatalf("reading want down file: %v", err)
	}

	tmpDir, err := os.MkdirTemp("", "terravalet")
	if err != nil {
		t.Fatalf("creating temporary dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	tmpUpPath := tmpDir + "/up"
	tmpDownPath := tmpDir + "/down"

	args = append(args, "--up", tmpUpPath, "--down", tmpDownPath)
	os.Args = args

	if err := run(); err != nil {
		t.Fatalf("run: args: %s\nhave: %q\nwant: no error", args, err)
	}

	tmpUp, err := os.ReadFile(tmpUpPath)
	if err != nil {
		t.Fatalf("reading tmp up file: %v", err)
	}
	tmpDown, err := os.ReadFile(tmpDownPath)
	if err != nil {
		t.Fatalf("reading tmp down file: %v", err)
	}

	if diff := cmp.Diff(string(wantUp), string(tmpUp)); diff != "" {
		t.Errorf("\nup script: mismatch (-want +have):\n"+
			"(want path: %s)\n"+
			"%s", wantUpPath, diff)
	}
	if diff := cmp.Diff(string(wantDown), string(tmpDown)); diff != "" {
		t.Errorf("\ndown script: mismatch (-want +have):\n"+
			"(want path: %s)\n"+
			"%s", wantDownPath, diff)
	}
}

func runFailure(t *testing.T, args []string, wantErr string) {
	tmpDir, err := os.MkdirTemp("", "terravalet")
	if err != nil {
		t.Fatalf("creating temporary dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	tmpUpPath := tmpDir + "/up"
	tmpDownPath := tmpDir + "/down"

	args = append(args, "--up", tmpUpPath, "--down", tmpDownPath)
	os.Args = args

	err = run()

	if err == nil {
		t.Fatalf("run: args: %s\nhave: no error\nwant: %q", args, wantErr)
	}
	if diff := cmp.Diff(wantErr, err.Error()); diff != "" {
		t.Errorf("error message mismatch (-want +have):\n%s", diff)
	}
}
