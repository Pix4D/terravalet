package main

import (
	"fmt"
	"io"
	"os"
	"strings"
)

func doRemove(planPath string, upPath string) error {
	planFile, err := os.Open(planPath)
	if err != nil {
		return fmt.Errorf("remove: opening the plan file: %s", err)
	}
	defer planFile.Close()

	upFile, err := os.Create(upPath)
	if err != nil {
		return fmt.Errorf("remove: creating the up file: %s", err)
	}
	defer upFile.Close()

	toCreate, toDestroy, err := parse(planFile)
	if err != nil {
		return fmt.Errorf("remove: parsing plan: %s", err)
	}
	if toCreate.Size() > 0 {
		return fmt.Errorf("remove: plan contains resources to create: %v",
			sorted(toCreate.List()))
	}

	var bld strings.Builder
	generateRemoveScript(&bld, sorted(toDestroy.List()))
	_, err = upFile.WriteString(bld.String())
	if err != nil {
		return fmt.Errorf("remove: writing script file: %s", err)
	}

	return nil
}

func generateRemoveScript(wr io.Writer, addresses []string) {
	fmt.Fprintf(wr, `#! /bin/sh
# DO NOT EDIT. Generated by https://github.com/pix4D/terravalet
# This script will remove %d items.

set -e

`, len(addresses))
	for _, addr := range addresses {
		fmt.Fprintf(wr, "terraform state rm '%s'\n", addr)
	}
	fmt.Fprintln(wr)
}
