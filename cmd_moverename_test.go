package main

import (
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestRunRenameSuccess(t *testing.T) {
	testCases := []struct {
		name         string
		options      []string
		planPath     string
		wantUpPath   string
		wantDownPath string
	}{
		{
			name:         "exact match",
			options:      []string{},
			planPath:     "testdata/rename/01_exact-match.plan.txt",
			wantUpPath:   "testdata/rename/01_exact-match.up.sh",
			wantDownPath: "testdata/rename/01_exact-match.down.sh",
		},
		{
			name:         "q-gram fuzzy match simple",
			options:      []string{"--fuzzy-match"},
			planPath:     "testdata/rename/02_fuzzy-match.plan.txt",
			wantUpPath:   "testdata/rename/02_fuzzy-match.up.sh",
			wantDownPath: "testdata/rename/02_fuzzy-match.down.sh",
		},
		{
			name:         "q-gram fuzzy match complicated",
			options:      []string{"--fuzzy-match"},
			planPath:     "testdata/rename/03_fuzzy-match.plan.txt",
			wantUpPath:   "testdata/rename/03_fuzzy-match.up.sh",
			wantDownPath: "testdata/rename/03_fuzzy-match.down.sh",
		},
		{
			name:         "q-gram fuzzy match complicated (regression)",
			options:      []string{"--fuzzy-match"},
			planPath:     "testdata/rename/07_fuzzy-match.plan.txt",
			wantUpPath:   "testdata/rename/07_fuzzy-match.up.sh",
			wantDownPath: "testdata/rename/07_fuzzy-match.down.sh",
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
		{
			name:     "plan file doesn't exist",
			planPath: "nonexisting",
			wantErr:  "opening the terraform plan file: open nonexisting: no such file or directory",
		},
		{
			name:     "matchExact failure",
			planPath: "testdata/rename/02_fuzzy-match.plan.txt",
			wantErr: `matchExact:
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

func TestRunMoveAfterSuccess(t *testing.T) {
	testCases := []struct {
		name       string
		before     string
		after      string
		wantScript string
	}{
		{
			name:       "exact match",
			before:     "testdata/move-after/04-before",
			after:      "testdata/move-after/04-after",
			wantScript: "testdata/move-after/04-want",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			args := []string{"terravalet", "move-after"}

			runMoveSuccess(t, args, tc.before, tc.after, tc.wantScript)
		})
	}
}

func TestRunMoveAfterFailure(t *testing.T) {
	testCases := []struct {
		name    string
		before  string // special value: "non-existing"
		after   string // special value: "non-existing"
		wantErr string
	}{
		{
			name:    "non existing before tfplan",
			before:  "non-existing",
			after:   "non-existing",
			wantErr: "opening the terraform BEFORE plan file: open non-existing.tfplan: no such file or directory",
		},
		{
			name:    "non existing after tfplan",
			before:  "testdata/move-after/05-before",
			after:   "non-existing",
			wantErr: "opening the terraform AFTER plan file: open non-existing.tfplan: no such file or directory",
		},
		{
			name:    "before tfplan must only destroy",
			before:  "testdata/move-after/05-before",
			after:   "testdata/move-after/05-after",
			wantErr: "BEFORE plan contains resources to create: [aws_batch_job_definition.foo]",
		},
		{
			name:    "after tfplan must only create",
			before:  "testdata/move-after/06-before",
			after:   "testdata/move-after/06-after",
			wantErr: "AFTER plan contains resources to destroy: [aws_batch_job_definition.foo]",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			args := []string{"terravalet", "move-after"}

			runMoveFailure(t, args, tc.before, tc.after, tc.wantErr)
		})
	}
}

func TestRunMoveBeforeSuccess(t *testing.T) {
	testCases := []struct {
		name       string
		before     string
		after      string // special prefix: dummy
		wantScript string
	}{
		{
			name:       "happy path simple",
			before:     "testdata/move-before/01-before",
			after:      "dummy-01-after",
			wantScript: "testdata/move-before/01-want",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			args := []string{"terravalet", "move-before"}

			runMoveSuccess(t, args, tc.before, tc.after, tc.wantScript)
		})
	}
}

func TestRunMoveBeforeFailure(t *testing.T) {
	testCases := []struct {
		name    string
		before  string // special value: "non-existing"
		wantErr string
	}{
		{
			name:    "non existing BEFORE plan",
			before:  "non-existing",
			wantErr: "opening the terraform BEFORE plan file: open non-existing.tfplan: no such file or directory",
		},
		{
			name:    "BEFORE plan must not contain resources to destroy",
			before:  "testdata/move-before/02-before",
			wantErr: "BEFORE plan contains resources to destroy: [aws_batch_compute_environment.foo_batch]",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			args := []string{"terravalet", "move-before",
				"--before=" + tc.before, "--after=testdata/move-before/dummy-after",
			}

			runMoveFailure(t, args, tc.before, "non-existing", tc.wantErr)
		})
	}
}

// If after has special prefix "dummy", it will not attempt to copy the
// corresponding tfplan files, to accomodate for move-before.
func runMoveSuccess(t *testing.T, args []string, before, after, wantScript string) {
	wantUpPath := wantScript + "_up.sh"
	wantUp, err := os.ReadFile(wantUpPath)
	if err != nil {
		t.Fatalf("reading want up file: %v", err)
	}

	wantDownPath := wantScript + "_down.sh"
	wantDown, err := os.ReadFile(wantDownPath)
	if err != nil {
		t.Fatalf("reading want down file: %v", err)
	}

	tmpDir, err := os.MkdirTemp("", "terravalet")
	if err != nil {
		t.Fatalf("creating temporary dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Copy the required input files to the tmpdir.
	if err := copyfile(before+".tfplan",
		filepath.Join(tmpDir, path.Base(before)+".tfplan")); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(after, "dummy") {
		if err := copyfile(after+".tfplan",
			filepath.Join(tmpDir, path.Base(after)+".tfplan")); err != nil {
			t.Fatal(err)
		}
	}

	// Change directory to the tmpdir.
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal("getwd:", err)
	}
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal("chdir:", err)
	}
	defer func() {
		if err := os.Chdir(cwd); err != nil {
			panic(err)
		}
	}()

	tmpScript := path.Base(wantScript)
	tmpUpPath := tmpScript + "_up.sh"
	tmpDownPath := tmpScript + "_down.sh"

	args = append(args, "--before="+path.Base(before), "--after="+path.Base(after),
		"--script="+tmpScript)
	os.Args = args

	if err := run(); err != nil {
		t.Fatalf("run: args: %s\nhave: %q\nwant: no error", args, err)
	}
	t.Log("SUT ran successfully")

	tmpUp, err := os.ReadFile(tmpUpPath)
	if err != nil {
		t.Fatalf("reading tmp up file: %s", err)
	}
	tmpDown, err := os.ReadFile(tmpDownPath)
	if err != nil {
		t.Fatalf("reading tmp down file: %s", err)
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

// If before or after have the special value "non-existing", it will not attempt to copy the
// corresponding tfplan files, allowing to test a missing file.
func runMoveFailure(t *testing.T, args []string, before, after, wantErr string) {
	tmpDir, err := os.MkdirTemp("", "terravalet")
	if err != nil {
		t.Fatalf("creating temporary dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Copy the required input files to the tmpdir.
	if before != "non-existing" {
		if err := copyfile(before+".tfplan",
			filepath.Join(tmpDir, path.Base(before)+".tfplan")); err != nil {
			t.Fatal(err)
		}
	}
	if after != "non-existing" {
		if err := copyfile(after+".tfplan",
			filepath.Join(tmpDir, path.Base(after)+".tfplan")); err != nil {
			t.Fatal(err)
		}
	}

	// Change directory to the tmpdir.
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal("getwd:", err)
	}
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal("chdir:", err)
	}
	defer func() {
		if err := os.Chdir(cwd); err != nil {
			panic(err)
		}
	}()

	args = append(args, "--before="+path.Base(before), "--after="+path.Base(after),
		"--script=dummy-script")
	os.Args = args

	err = run()

	if err == nil {
		t.Fatalf("run: args: %s\nhave: no error\nwant: %q", args, wantErr)
	}
	if diff := cmp.Diff(wantErr, err.Error()); diff != "" {
		t.Errorf("error message mismatch (-want +have):\n%s", diff)
	}
}

// copyfile copies file src to dst. It is not robust for production use (a lot of OS-dependent
// corner cases) but is good enough for tests.
func copyfile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("opening src file: %s", err)
	}
	defer srcFile.Close()

	// Create (or truncate) the dst file
	dstFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("creating dst file: %s", err)
	}
	defer dstFile.Close()

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return err
	}

	return nil
}
