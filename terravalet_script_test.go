// This file runs tests using the 'testscript' package.
// To understand, see:
// - https://github.com/rogpeppe/go-internal
// - https://bitfieldconsulting.com/golang/test-scripts

package main

import (
	"testing"

	"github.com/rogpeppe/go-internal/testscript"
)

func TestMain(m *testing.M) {
	testscript.Main(m, map[string]func(){
		"terravalet": func() { Main() },
	})
}

func TestScript(t *testing.T) {
	testscript.Run(t, testscript.Params{
		Dir: "testdata/script",
	})
}
