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
	os.Exit(Main())
}

func Main() int {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return 1
	}
	return 0
}

type Args struct {
	Rename     *RenameCmd     `arg:"subcommand:rename" help:"rename resources in the same root environment"`
	MoveAfter  *MoveAfterCmd  `arg:"subcommand:move-after" help:"move resources from one root environment to AFTER another"`
	MoveBefore *MoveBeforeCmd `arg:"subcommand:move-before" help:"move resources from one root environment to BEFORE another"`
	Import     *ImportCmd     `arg:"subcommand:import" help:"import resources generated out-of-band of Terraform"`
	Remove     *RemoveCmd     `arg:"subcommand:remove" help:"remove resources"`
	Version    *struct{}      `arg:"subcommand:version" help:"show version"`
}

func (Args) Description() string {
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
	Script string `arg:"required" help:"the migration scripts; will generate SCRIPT_up.sh and SCRIPT_down.sh"`
	Before string `arg:"required" help:"the before root directory; will look for BEFORE.tfplan and BEFORE.tfstate"`
	After  string `arg:"required" help:"the after root directory; will look for AFTER.tfplan and AFTER.tfstate"`
}

type MoveBeforeCmd struct {
	Script string `arg:"required" help:"the migration scripts; will generate SCRIPT_up.sh and SCRIPT_down.sh"`
	Before string `arg:"required" help:"the before root directory; will look for BEFORE.tfplan and BEFORE.tfstate"`
	After  string `arg:"required" help:"the after root directory; will look for AFTER.tfstate"`
}

type ImportCmd struct {
	UpDown
	ResourceDefs string `arg:"--res-defs,required" help:"path to resource definitions"`
	SrcPlanPath  string `arg:"--src-plan,required" help:"path to the SRC terraform plan in JSON format"`
}

type RemoveCmd struct {
	Up   string `arg:"required" help:"path of the up script to generate (NNN_TITLE.up.sh)"`
	Plan string `arg:"required" help:"path to to the output of 'terraform plan -no-color'"`
}

func run() error {
	var args Args

	parser := arg.MustParse(&args)
	if parser.Subcommand() == nil {
		parser.Fail("missing subcommand")
	}

	switch {
	case args.Rename != nil:
		cmd := args.Rename
		return doRename(cmd.Up, cmd.Down, cmd.PlanPath, cmd.LocalStatePath,
			cmd.FuzzyMatch)
	case args.MoveAfter != nil:
		cmd := args.MoveAfter
		return doMoveAfter(cmd.Script, cmd.Before, cmd.After)
	case args.MoveBefore != nil:
		cmd := args.MoveBefore
		return doMoveBefore(cmd.Script, cmd.Before, cmd.After)
	case args.Import != nil:
		cmd := args.Import
		return doImport(cmd.Up, cmd.Down, cmd.SrcPlanPath, cmd.ResourceDefs)
	case args.Remove != nil:
		cmd := args.Remove
		return doRemove(cmd.Plan, cmd.Up)
	case args.Version != nil:
		fmt.Println("terravalet", fullVersion)
		return nil
	default:
		return fmt.Errorf("internal error: unwired command: %s",
			parser.SubcommandNames()[0])
	}
}
