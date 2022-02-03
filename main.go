// This code is released under the MIT License
// Copyright (c) 2020 Pix4D and the terravalet contributors.

package main

import (
	"fmt"
	"os"

	"github.com/alexflint/go-arg"
)

var (
	// Filled by the linker.
	fullVersion = "unknown" // example: v0.0.9-8-g941583d027-dirty
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

type args struct {
	Rename     *RenameCmd     `arg:"subcommand:rename" help:"rename resources in the same root environment"`
	MoveAfter  *MoveAfterCmd  `arg:"subcommand:move-after" help:"move resources from one root environment to AFTER another"`
	Import     *ImportCmd     `arg:"subcommand:import" help:"import resources generated out-of-band of Terraform"`
	Version    *struct{}      `arg:"subcommand:version" help:"show version"`
}

func (args) Description() string {
	return "terravalet - helps with advanced Terraform operations\n"
}

type UpDown struct {
	Up   string `arg:"required" help:"path of the up script to generate (NNN_TITLE.up.sh)"`
	Down string `arg:"required" help:"path of the down script to generate (NNN_TITLE.down.sh)"`
}

type RenameCmd struct {
	UpDown
	PlanPath       string `arg:"--plan,required" help:"path to the terraform plan"`
	LocalStatePath string `arg:"--local-state" help:"path to the local state to modify (both src and dst)" default:"local.tfstate"`
	FuzzyMatch     bool   `arg:"--fuzzy-match" help:"enable q-gram distance fuzzy matching. WARNING: You must validate by hand the output!"`
}

type MoveAfterCmd struct {
	UpDown
	SrcPlanPath  string `arg:"--src-plan,required" help:"path to the SRC terraform plan"`
	DstPlanPath  string `arg:"--dst-plan,required" help:"path to the DST terraform plan"`
	SrcStatePath string `arg:"--src-state,required" help:"path to the SRC local state to modify"`
	DstStatePath string `arg:"--dst-state,required" help:"path to the DST local state to modify"`
}

type ImportCmd struct {
	UpDown
	ResourceDefs string `arg:"--res-defs,required" help:"path to resource definitions"`
	SrcPlanPath  string `arg:"--src-plan,required" help:"path to the SRC terraform plan in JSON format"`
}

func run() error {
	var args args

	parser := arg.MustParse(&args)
	if parser.Subcommand() == nil {
		parser.Fail("missing subcommand")
	}

	switch {
	case args.Rename != nil:
		return doRename(args.Rename.Up, args.Rename.Down,
			args.Rename.PlanPath, args.Rename.LocalStatePath, args.Rename.FuzzyMatch)
	case args.MoveAfter != nil:
		cmd := args.MoveAfter
		return doMoveAfter(cmd.Up, cmd.Down,
			cmd.SrcPlanPath, cmd.DstPlanPath,
			cmd.SrcStatePath, cmd.DstStatePath)
	case args.Import != nil:
		return doImport(args.Import.Up, args.Import.Down,
			args.Import.SrcPlanPath, args.Import.ResourceDefs)
	case args.Version != nil:
		fmt.Println("terravalet", fullVersion)
		return nil
	default:
		return fmt.Errorf("internal error: unwired command: %s", parser.SubcommandNames()[0])
	}
}
