package main

import (
	"fmt"
	"io/ioutil"
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

func TestRunRenameSuccess(t *testing.T) {
	testCases := []struct {
		name         string
		options      []string
		planPath     string
		wantUpPath   string
		wantDownPath string
	}{
		{
			"exact match",
			[]string{},
			"testdata/rename/01_exact-match.plan.txt",
			"testdata/rename/01_exact-match.up.sh",
			"testdata/rename/01_exact-match.down.sh",
		},
		{
			"q-gram fuzzy match simple",
			[]string{"--fuzzy-match"},
			"testdata/rename/02_fuzzy-match.plan.txt",
			"testdata/rename/02_fuzzy-match.up.sh",
			"testdata/rename/02_fuzzy-match.down.sh",
		},
		{
			"q-gram fuzzy match complicated",
			[]string{"--fuzzy-match"},
			"testdata/rename/03_fuzzy-match.plan.txt",
			"testdata/rename/03_fuzzy-match.up.sh",
			"testdata/rename/03_fuzzy-match.down.sh",
		},
		{
			"q-gram fuzzy match complicated (regression)",
			[]string{"--fuzzy-match"},
			"testdata/rename/07_fuzzy-match.plan.txt",
			"testdata/rename/07_fuzzy-match.up.sh",
			"testdata/rename/07_fuzzy-match.down.sh",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			args := []string{"terravalet", "rename", "--plan", tc.planPath}
			args = append(args, tc.options...)

			runSuccess(t, args, tc.wantUpPath, tc.wantDownPath)
		})
	}
}

func TestRunRenameFailure(t *testing.T) {
	testCases := []struct {
		name     string
		planPath string
		wantErr  string
	}{
		{"plan file doesn't exist",
			"nonexisting",
			"opening the terraform plan file: open nonexisting: no such file or directory",
		},
		{"matchExact failure",
			"testdata/rename/02_fuzzy-match.plan.txt",
			`matchExact:
unmatched create:
  aws_route53_record.localhostnames_public["artifactory"]
  aws_route53_record.loopback["artifactory"]
  aws_route53_record.private["artifactory"]
unmatched destroy:
  aws_route53_record.artifactory
  aws_route53_record.artifactory_loopback
  aws_route53_record.artifactory_private`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			args := []string{"terravalet", "rename", "--plan", tc.planPath}

			runFailure(t, args, tc.wantErr)
		})
	}
}

func TestRunMoveSuccess(t *testing.T) {
	testCases := []struct {
		name         string
		srcPlanPath  string
		dstPlanPath  string
		wantUpPath   string
		wantDownPath string
	}{
		{
			"exact match",
			"testdata/move/04_src-plan.txt",
			"testdata/move/04_dst-plan.txt",
			"testdata/move/04_up.sh",
			"testdata/move/04_down.sh",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			args := []string{"terravalet", "move",
				"--src-plan", tc.srcPlanPath, "--dst-plan", tc.dstPlanPath,
				"--src-state", "src-dummy", "--dst-state", "dst-dummy",
			}

			runSuccess(t, args, tc.wantUpPath, tc.wantDownPath)
		})
	}
}

func TestRunMoveFailure(t *testing.T) {
	testCases := []struct {
		name         string
		srcPlanPath  string
		dstPlanPath  string
		wantUpPath   string
		wantDownPath string
		wantErr      string
	}{
		{"non existing src-plan",
			"src-plan-path-dummy",
			"dst-plan-path-dummy",
			"want-up-path-dummy",
			"want-down-path-dummy",
			"opening the terraform plan file: open src-plan-path-dummy: no such file or directory",
		},
		{"src-plan must only destroy",
			"testdata/move/05_src-plan.txt",
			"testdata/move/05_dst-plan.txt",
			"want-up-path-dummy",
			"want-down-path-dummy",
			"src-plan contains resources to create: [aws_batch_job_definition.foo]",
		},
		{"dst-plan must only create",
			"testdata/move/06_src-plan.txt",
			"testdata/move/06_dst-plan.txt",
			"want-up-path-dummy",
			"want-down-path-dummy",
			"dst-plan contains resources to destroy: [aws_batch_job_definition.foo]",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			args := []string{"terravalet", "move",
				"--src-plan", tc.srcPlanPath, "--dst-plan", tc.dstPlanPath,
				"--src-state", "src-dummy", "--dst-state", "dst-dummy",
			}

			runFailure(t, args, tc.wantErr)
		})
	}
}

func runSuccess(t *testing.T, args []string, wantUpPath string, wantDownPath string) {
	wantUp, err := ioutil.ReadFile(wantUpPath)
	if err != nil {
		t.Fatalf("reading want up file: %v", err)
	}
	wantDown, err := ioutil.ReadFile(wantDownPath)
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

	args = append(args, "--up", tmpUpPath, "--down", tmpDownPath)
	os.Args = args

	if err := run(); err != nil {
		t.Fatalf("run: args: %s\ngot:  %q\nwant: no error", args, err)
	}

	tmpUp, err := ioutil.ReadFile(tmpUpPath)
	if err != nil {
		t.Fatalf("reading tmp up file: %v", err)
	}
	tmpDown, err := ioutil.ReadFile(tmpDownPath)
	if err != nil {
		t.Fatalf("reading tmp down file: %v", err)
	}

	if diff := cmp.Diff(wantUp, tmpUp); diff != "" {
		t.Errorf("\nup script: mismatch (-want +got):\n"+
			"(want path: %s)\n"+
			"%s", wantUpPath, diff)
	}
	if diff := cmp.Diff(wantDown, tmpDown); diff != "" {
		t.Errorf("\ndown script: mismatch (-want +got):\n"+
			"(want path: %s)\n"+
			"%s", wantDownPath, diff)
	}
}

func runFailure(t *testing.T, args []string, wantErr string) {
	tmpDir, err := ioutil.TempDir("", "terravalet")
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
		t.Fatalf("run: args: %s\ngot:  no error\nwant: %q", args, err)
	}
	if diff := cmp.Diff(wantErr, err.Error()); diff != "" {
		t.Errorf("error message mismatch (-want +have):\n%s", diff)
	}
}

// Used to compare sets.
var setCmp = cmp.Comparer(func(s1, s2 *strset.Set) bool {
	return s1.IsEqual(s2)
})

func TestParseSuccess(t *testing.T) {
	testCases := []struct {
		name        string
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
		t.Run(tc.name, func(t *testing.T) {
			rd := strings.NewReader(tc.line)

			gotCreate, gotDestroy, err := parse(rd)

			if err != nil {
				t.Fatalf("\ngot:  %q\nwant: no error", err)
			}
			if diff := cmp.Diff(tc.wantCreate, gotCreate, setCmp); diff != "" {
				t.Errorf("\ncreate: mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tc.wantDestroy, gotDestroy, setCmp); diff != "" {
				t.Errorf("\ndestroy: mismatch (-want +got):\n%s", diff)
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
			"vaporized is not an expected action",
			"  # aws_instance.bar will be vaporized",
			`line "  # aws_instance.bar will be vaporized", unexpected action "vaporized"`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			rd := strings.NewReader(tc.line)

			_, _, err := parse(rd)

			if err == nil {
				t.Fatalf("\ngot:  no error\nwant: %q", tc.wantErr)
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
		t.Run(tc.name, func(t *testing.T) {
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
		name        string
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
		t.Run(tc.name, func(t *testing.T) {
			matchExact(tc.create, tc.destroy)

			if diff := cmp.Diff(tc.wantCreate, tc.create, setCmp); diff != "" {
				t.Errorf("\nUnmatched create: (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tc.wantDestroy, tc.destroy, setCmp); diff != "" {
				t.Errorf("\nUnmatched destroy (-want +got):\n%s", diff)
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
		t.Run(tc.name, func(t *testing.T) {

			gotUpMatches, gotDownMatches, err := matchFuzzy(tc.create, tc.destroy)
			if err != nil {
				t.Fatalf("got: %s; want: no error", err)
			}

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

func TestMatchFuzzyError(t *testing.T) {
	create := set.NewStringSet(`abcde`, `abdecde`)
	destroy := set.NewStringSet(`abdcde`, `hfjabd`)
	_, _, err := matchFuzzy(create, destroy)
	if err == nil {
		t.Fatalf("got: no error; want: an ambiguous migration error")
	}

	gotMsg := err.Error()
	var msg string

	want := "ambiguous migration:"
	if !strings.HasPrefix(gotMsg, want) {
		msg += fmt.Sprintf("error message does not start with %q\n", want)
	}

	want = "{abcde} -> {abdcde}"
	if !strings.Contains(gotMsg, want) {
		msg += fmt.Sprintf("error message does not contain %q", want)
	}

	want = "{abdecde} -> {abdcde}"
	if !strings.Contains(gotMsg, want) {
		msg += fmt.Sprintf("error message does not contain %q", want)
	}

	if msg != "" {
		t.Fatal(msg)
	}
}

func TestRunImportSuccess(t *testing.T) {
	testCases := []struct {
		name         string
		resDefs      string
		srcPlanPath  string
		wantUpPath   string
		wantDownPath string
	}{
		{
			"import resources",
			"testdata/import/terravalet_imports_definitions.json",
			"testdata/import/08_import_src-plan.json",
			"testdata/import/08_import_up.sh",
			"testdata/import/08_import_down.sh",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			args := []string{"terravalet", "import",
				"--res-defs", tc.resDefs,
				"--src-plan", tc.srcPlanPath,
			}

			runSuccess(t, args, tc.wantUpPath, tc.wantDownPath)
		})
	}
}

func TestRunImportFailure(t *testing.T) {
	testCases := []struct {
		name        string
		resDefs     string
		srcPlanPath string
		wantErr     string
	}{
		{"non existing src-plan",
			"testdata/import/terravalet_imports_definitions.json",
			"src-plan-path-dummy",
			"opening the terraform plan file: open src-plan-path-dummy: no such file or directory",
		},
		{"src-plan is invalid json",
			"testdata/import/terravalet_imports_definitions.json",
			"testdata/import/09_import_empty_src-plan.json",
			"parse src-plan: parsing the plan: unexpected end of JSON input",
		},
		{"src-plan must create resource",
			"testdata/import/terravalet_imports_definitions.json",
			"testdata/import/10_import_no-new-resources.json",
			"parse src-plan: src-plan doesn't contains resources to create",
		},
		{"src-plan contains only undefined resources",
			"testdata/import/terravalet_imports_definitions.json",
			"testdata/import/11_import_src-plan_undefined_resources.json",
			"parse src-plan: src-plan contains only undefined resources",
		},
		{"src-plan contains a not existing resource parameter",
			"testdata/import/terravalet_imports_definitions.json",
			"testdata/import/12_import_src-plan_invalid_resource_param.json",
			"parse src-plan: error in resources definition dummy_resource2: field 'long_name' doesn't exist in plan",
		},
		{"terravalet missing resources definitions file",
			"testdata/import/missing.file",
			"testdata/import/08_import_src-plan.json",
			"opening the definitions file: open testdata/import/missing.file: no such file or directory",
		},
		{"terravalet invalid resources definitions file",
			"testdata/import/invalid_imports_definitions.json",
			"testdata/import/08_import_src-plan.json",
			"parse src-plan: parsing resources definitions: invalid character '}' after object key",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			args := []string{"terravalet", "import",
				"--res-defs", tc.resDefs,
				"--src-plan", tc.srcPlanPath,
			}

			runFailure(t, args, tc.wantErr)
		})
	}
}
