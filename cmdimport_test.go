package main

import "testing"

func TestRunImportSuccess(t *testing.T) {
	testCases := []struct {
		name         string
		resDefs      string
		srcPlanPath  string
		wantUpPath   string
		wantDownPath string
	}{
		{
			name:         "import resources",
			resDefs:      "testdata/import/terravalet_imports_definitions.json",
			srcPlanPath:  "testdata/import/08_import_src-plan.json",
			wantUpPath:   "testdata/import/08_import_up.sh",
			wantDownPath: "testdata/import/08_import_down.sh",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			args := []string{"terravalet", "import",
				"--res-defs", tc.resDefs,
				"--src-plan", tc.srcPlanPath,
			}

			runSuccess(t, args, tc.wantUpPath, tc.wantDownPath)
		})
	}
}

func TestRunImportFailure(t *testing.T) {
	testCases := []struct {
		name        string
		resDefs     string
		srcPlanPath string
		wantErr     string
	}{
		{
			name:        "non existing src-plan",
			resDefs:     "testdata/import/terravalet_imports_definitions.json",
			srcPlanPath: "src-plan-path-dummy",
			wantErr:     "opening the terraform plan file: open src-plan-path-dummy: no such file or directory",
		},
		{
			name:        "src-plan is invalid json",
			resDefs:     "testdata/import/terravalet_imports_definitions.json",
			srcPlanPath: "testdata/import/09_import_empty_src-plan.json",
			wantErr:     "parse src-plan: parsing the plan: unexpected end of JSON input",
		},
		{
			name:        "src-plan must create resource",
			resDefs:     "testdata/import/terravalet_imports_definitions.json",
			srcPlanPath: "testdata/import/10_import_no-new-resources.json",
			wantErr:     "parse src-plan: src-plan doesn't contains resources to create",
		},
		{
			name:        "src-plan contains only undefined resources",
			resDefs:     "testdata/import/terravalet_imports_definitions.json",
			srcPlanPath: "testdata/import/11_import_src-plan_undefined_resources.json",
			wantErr:     "parse src-plan: src-plan contains only undefined resources",
		},
		{
			name:        "src-plan contains a not existing resource parameter",
			resDefs:     "testdata/import/terravalet_imports_definitions.json",
			srcPlanPath: "testdata/import/12_import_src-plan_invalid_resource_param.json",
			wantErr:     "parse src-plan: error in resources definition dummy_resource2: field 'long_name' doesn't exist in plan",
		},
		{
			name:        "terravalet missing resources definitions file",
			resDefs:     "testdata/import/missing.file",
			srcPlanPath: "testdata/import/08_import_src-plan.json",
			wantErr:     "opening the definitions file: open testdata/import/missing.file: no such file or directory",
		},
		{
			name:        "terravalet invalid resources definitions file",
			resDefs:     "testdata/import/invalid_imports_definitions.json",
			srcPlanPath: "testdata/import/08_import_src-plan.json",
			wantErr:     "parse src-plan: parsing resources definitions: invalid character '}' after object key",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			args := []string{"terravalet", "import",
				"--res-defs", tc.resDefs,
				"--src-plan", tc.srcPlanPath,
			}

			runFailure(t, args, tc.wantErr)
		})
	}
}
