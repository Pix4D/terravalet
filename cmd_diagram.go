package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/pix4d/terravalet/pkg/tfstate"
)

func doDiagram(statePath string) error {
	errorf := MakeErrorf("diagram")

	data, err := os.ReadFile(statePath)
	if err != nil {
		return errorf("reading state file: %s", err)
	}

	if err := process(data); err != nil {
		return errorf("processing state file: %s", err)
	}
	return nil
}

// core  (flat):            values / root_module / resources
// infra (with tf modules): values / root_module /child_modules / resources

// "type": "aws_security_group",
// values
//     "id": "sg-<hex-digits>",
//     ingress: [...]
//     egress:  [...]

// "type": "aws_security_group_rule"
// values:
//     "security_group_id":      "sg-<hex-digits>",
//     "security_group_rule_id": "sgr-<hex-digits>",

func process(data []byte) error {
	var state tfstate.State
	if err := json.Unmarshal(data, &state); err != nil {
		return fmt.Errorf("parsing state file: %s", err)
	}
	traverse(state.Values.RootModule)
	return nil
}

func traverse(module tfstate.Module) {
	fmt.Println("RESOURCES")
	for _, res := range module.Resources {
		if err := print(res); err != nil {
			fmt.Println(err)
		}
	}
	fmt.Println("CHILDMODULES")
	for _, module := range module.ChildModules {
		traverse(module)
	}
}

func print(res tfstate.Resource) error {
	if res.Type == tfstate.TypeAwsSecurityGroup {
		var asg tfstate.AwsSecurityGroup
		if err := json.Unmarshal(res.Values, &asg); err != nil {
			return fmt.Errorf("unmarshaling to ASG: %s", err)
		}
		fmt.Printf("ASG %+v\n", asg)
		return nil
	}
	fmt.Println("TYPE", res.Type)
	return nil
}
