package import_resources

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"strings"
)

type Resource struct {
	ResourceChanges []struct {
		Res
	} `json:"resource_changes"`
}

type Res struct {
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

	add := make([]string, 0)
	remove := make([]string, 0)
	configs := make(map[string]Definitions)
	var planParsed Resource
	var filteredResources []Res

	plan, err := ioutil.ReadAll(rd)
	if err != nil {
		return add, remove,
			fmt.Errorf("reading the plan file: %s", err)
	}
	if err = json.Unmarshal([]byte(plan), &planParsed); err != nil {
		return add, remove,
			fmt.Errorf("parsing the plan: %s", err)
	}

	defs, err := ioutil.ReadAll(definitionsFile)
	if err != nil {
		return add, remove,
			fmt.Errorf("reading the definitions file: %s", err)
	}
	if err = json.Unmarshal([]byte(defs), &configs); err != nil {
		return add, remove,
			fmt.Errorf("parsing resources definitions: %s", err)
	}

	// Return objects in the correct order if 'priority' parameter is set in provider configuration.
	// The remove order is reversed (LIFO logic).
	resources := planParsed.ResourceChanges

	// Filter all "create" resources before going further
	for _, resource := range resources {
		action := resource.Change.Actions[0]
		if action == "create" {
			filteredResources = append(filteredResources, resource.Res)
		}
	}

	if len(filteredResources) == 0 {
		return add, remove,
			fmt.Errorf("src-plan doesn't contains resources to create")
	}

	for _, resource := range filteredResources {
		// Get resource address
		a := fmt.Sprintf("'%s'", resource.Address)
		p := resource.ProviderName
		t := resource.Type
		// Proceed only if type is declared in resources definitions
		if _, ok := configs[t]; !ok {
			msg := fmt.Sprintf("Warning: resource %s is not defined. Check %s documentation\n", t, p)
			fmt.Printf("\033[1;33m%s\033[0m", msg)
			break
		}
		resourceParams := configs[t]
		variables := resourceParams.Variables
		var id []string
		v := resource.Change.After.(map[string]interface{})
		for _, field := range variables {
			if _, ok := v[field]; !ok {
				return add, remove,
					fmt.Errorf("error in resources definition %s: field '%s' doesn't exist in plan", t, field)
			}
			id = append(id, fmt.Sprintf("%s", v[field]))
		}
		separator := resourceParams.Separator
		arg := fmt.Sprintf("%s %s", a, strings.Join(id, separator))
		if resourceParams.Priority == 1 {
			// Prepend
			add = append([]string{arg}, add...)
			// Append
			remove = append(remove, a)
		} else {
			// Append
			add = append(add, arg)
			// Prepend
			remove = append([]string{a}, remove...)
		}
	}

	if len(add) == 0 {
		return add, remove,
			fmt.Errorf("src-plan contains only undefined resources")
	}

	return add, remove, nil
}
