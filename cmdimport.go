package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
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

func Import(rd, definitionsFile io.Reader) ([]string, []string, error) {
	var imports []string
	var removals []string
	var configs map[string]Definitions
	var resourcesBundle ResourcesBundle
	var filteredResources []ResourceChange

	plan, err := ioutil.ReadAll(rd)
	if err != nil {
		return imports, removals,
			fmt.Errorf("reading the plan file: %s", err)
	}
	if err = json.Unmarshal([]byte(plan), &resourcesBundle); err != nil {
		return imports, removals,
			fmt.Errorf("parsing the plan: %s", err)
	}

	defs, err := ioutil.ReadAll(definitionsFile)
	if err != nil {
		return imports, removals,
			fmt.Errorf("reading the definitions file: %s", err)
	}
	if err = json.Unmarshal([]byte(defs), &configs); err != nil {
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
			msg := fmt.Sprintf("Warning: resource %s is not defined. Check %s documentation\n", resource.Type, resource.ProviderName)
			fmt.Printf("\033[1;33m%s\033[0m", msg)
			break
		}
		resourceParams := configs[resource.Type]
		var id []string
		after := resource.Change.After.(map[string]interface{})
		for _, field := range resourceParams.Variables {
			if _, ok := after[field]; !ok {
				return imports, removals,
					fmt.Errorf("error in resources definition %s: field '%s' doesn't exist in plan", resource.Type, field)
			}
			id = append(id, fmt.Sprintf("%s", after[field]))
		}

		resAddr := fmt.Sprintf("'%s'", resource.Address)
		arg := fmt.Sprintf("%s %s", resAddr, strings.Join(id, resourceParams.Separator))
		if resourceParams.Priority == 1 {
			// Prepend
			imports = append([]string{arg}, imports...)
			// Append
			removals = append(removals, resAddr)
		} else {
			// Append
			imports = append(imports, arg)
			// Prepend
			removals = append([]string{resAddr}, removals...)
		}
	}

	if len(imports) == 0 {
		return imports, removals,
			fmt.Errorf("src-plan contains only undefined resources")
	}

	return imports, removals, nil
}
