package main

import (
	"bytes"
	"io/ioutil"
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
