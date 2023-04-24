package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
)

type ResourcesBundle struct {
	ResourceChanges []ResourceChange `json:"resource_changes"`
}

type ResourceChange struct {
	Address      string `json:"address"`
	Type         string `json:"type"`
	ProviderName string `json:"provider_name"`
	Change       struct {
		Actions []string    `json:"actions"`
		After   interface{} `json:"after"`
	} `json:"change"`
}

type Definitions struct {
	Separator string   `json:"separator"`
	Priority  int      `json:"priority"`
	Variables []string `json:"variables"`
}

// Keep track of the asymmetry of import subcommand.
// When importing, the up direction wants two parameters:
//
//	terraform import res-address res-id
//
// while the down direction wants only one parameter:
//
//	terraform state rm res-address
type ImportElement struct {
	Addr string
	ID   string
}

func doImport(upPath, downPath, srcPlanPath, resourcesDefinitions string) error {
	definitionsFile, err := os.Open(resourcesDefinitions)
	if err != nil {
		return fmt.Errorf("opening the definitions file: %v", err)
	}
	defer definitionsFile.Close()

	srcPlanFile, err := os.Open(srcPlanPath)
	if err != nil {
		return fmt.Errorf("opening the terraform plan file: %v", err)
	}
	defer srcPlanFile.Close()

	upFile, err := os.Create(upPath)
	if err != nil {
		return fmt.Errorf("creating the up file: %v", err)
	}
	defer upFile.Close()

	downFile, err := os.Create(downPath)
	if err != nil {
		return fmt.Errorf("creating the down file: %v", err)
	}
	defer downFile.Close()

	imports, removals, err := Import(srcPlanFile, definitionsFile)
	if err != nil {
		return fmt.Errorf("parse src-plan: %v", err)
	}

	if err := importUpScript(imports, upFile); err != nil {
		return fmt.Errorf("writing the up script: %v", err)
	}
	if err := importDownScript(removals, downFile); err != nil {
		return fmt.Errorf("writing the down script: %v", err)
	}

	return nil
}

func Import(rd, definitionsFile io.Reader) ([]ImportElement, []ImportElement, error) {
	var imports []ImportElement
	var removals []ImportElement
	var configs map[string]Definitions
	var resourcesBundle ResourcesBundle
	var filteredResources []ResourceChange

	plan, err := io.ReadAll(rd)
	if err != nil {
		return imports, removals,
			fmt.Errorf("reading the plan file: %s", err)
	}
	if err = json.Unmarshal(plan, &resourcesBundle); err != nil {
		return imports, removals,
			fmt.Errorf("parsing the plan: %s", err)
	}

	defs, err := io.ReadAll(definitionsFile)
	if err != nil {
		return imports, removals,
			fmt.Errorf("reading the definitions file: %s", err)
	}
	if err = json.Unmarshal(defs, &configs); err != nil {
		return imports, removals,
			fmt.Errorf("parsing resources definitions: %s", err)
	}

	// Return objects in the correct order if 'priority' parameter is set in provider configuration.
	// The remove order is reversed (LIFO logic).

	// Filter all "create" resources before going further
	for _, resource := range resourcesBundle.ResourceChanges {
		if resource.Change.Actions[0] == "create" {
			filteredResources = append(filteredResources, resource)
		}
	}

	if len(filteredResources) == 0 {
		return imports, removals,
			fmt.Errorf("src-plan doesn't contains resources to create")
	}

	for _, resource := range filteredResources {
		// Proceed only if type is declared in resources definitions
		if _, ok := configs[resource.Type]; !ok {
			msg := fmt.Sprintf("Warning: resource %s is not defined. Check %s documentation\n",
				resource.Type, resource.ProviderName)
			fmt.Printf("\033[1;33m%s\033[0m", msg)
			continue
		}
		resourceParams := configs[resource.Type]
		var resID []string
		after := resource.Change.After.(map[string]interface{})
		for _, field := range resourceParams.Variables {
			if _, ok := after[field]; !ok {
				return imports, removals,
					fmt.Errorf(
						"error in resources definition %s: field '%s' doesn't exist in plan",
						resource.Type, field)
			}
			subID, ok := after[field].(string)
			if !ok {
				return imports, removals,
					fmt.Errorf("resource_changes: after: %s: type is %T; want: string",
						field, after[field])
			}
			resID = append(resID, subID)
		}

		elem := ImportElement{
			Addr: resource.Address,
			ID:   strings.Join(resID, resourceParams.Separator)}

		if resourceParams.Priority == 1 {
			// Prepend
			imports = append([]ImportElement{elem}, imports...)
		} else {
			// Append
			imports = append(imports, elem)
		}
	}

	if len(imports) == 0 {
		return imports, removals,
			fmt.Errorf("src-plan contains only undefined resources")
	}

	// The removals are the reverse of the imports.
	removals = make([]ImportElement, 0, len(imports))
	for i := len(imports) - 1; i >= 0; i-- {
		removals = append(removals, imports[i])
	}

	return imports, removals, nil
}

func importUpScript(elements []ImportElement, out io.Writer) error {
	cmd := "terraform import"
	fmt.Fprintf(out, importScriptHeader, cmd, len(elements))
	for _, elem := range elements {
		fmt.Fprintf(out, "%s \\\n    %q %q\n\n", cmd, elem.Addr, elem.ID)
	}
	return nil
}

func importDownScript(elements []ImportElement, out io.Writer) error {
	cmd := "terraform state rm"
	fmt.Fprintf(out, importScriptHeader, cmd, len(elements))
	for _, elem := range elements {
		fmt.Fprintf(out, "%s \\\n    %q\n\n", cmd, elem.Addr)
	}
	return nil
}

const importScriptHeader = `#! /bin/sh
# DO NOT EDIT. Generated by terravalet.
# WARNING: check the order of resources before running this script.
#
# This script will %q %d items.

# Uncomment this if you want to stop the script at first error
# set -e
set -x

`
