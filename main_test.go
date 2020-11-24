package main

import (
	"bytes"
	"io/ioutil"
	"testing"

	"github.com/pkg/diff"
)

func TestHappyPath(t *testing.T) {
	testCases := []struct {
		input      string
		wantScript string
	}{
		{
			"testdata/plan-synthetic-01.txt",
			"testdata/001_synthetic.up.sh",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			var got bytes.Buffer
			if err := run([]string{tc.input}, &got); err != nil {
				t.Fatalf("got error: %v; want: no error", err)
			}
			want, err := ioutil.ReadFile(tc.wantScript)
			if err != nil {
				t.Fatalf("reading want file %q: error: %v", tc.wantScript, err)
			}

			if !bytes.Equal(got.Bytes(), want) {
				var outDiff bytes.Buffer
				diff.Text("got", tc.wantScript, got.Bytes(), want, &outDiff)
				t.Fatalf("got the following differences:\n%v", outDiff.String())
			}
		})
	}
}
