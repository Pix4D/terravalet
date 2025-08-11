package tfstate

import "encoding/json"

type State struct {
	FormatVersion    string `json:"format_version"`
	TerraformVersion string `json:"terraform_version"`
	Values           Values `json:"values"`
}

type Values struct {
	// Outputs
	RootModule Module `json:"root_module"`
}

type Module struct {
	Address      string     `json:"address"`
	Resources    []Resource `json:"resources"`
	ChildModules []Module   `json:"child_modules"`
}

type Resource struct {
	ResourceFields
	// The shape of 'Values' depends on '[ResourceFields.Type]'.
	Values json.RawMessage `json:"values"`
}

type ResourceFields struct {
	Address       string `json:"address"`
	Mode          string `json:"mode"`
	Type          string `json:"type"`
	Name          string `json:"name"`
	ProviderName  string `json:"provider_name"`
	SchemaVersion int    `json:"schema_version"`
}
