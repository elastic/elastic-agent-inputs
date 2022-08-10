// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build mage
// +build mage

package main

import (
	"fmt"
	"path/filepath"

	"github.com/magefile/mage/mg"

	// mage:import
	"github.com/elastic/elastic-agent-libs/dev-tools/mage"

	devtools "github.com/elastic/elastic-agent-libs/dev-tools/mage"
	"github.com/elastic/elastic-agent-libs/dev-tools/mage/gotool"
)

// Aliases are shortcuts to long target names.
// nolint: deadcode // it's used by `mage`.
var Aliases = map[string]interface{}{
	"llc":  mage.Linter.LastChange,
	"lint": mage.Linter.All,
}

// Check runs all the checks
// nolint: deadcode,unparam // it's used as a `mage` target and requires returning an error
func Check() error {
	mg.Deps(devtools.Deps.CheckModuleTidy, CheckLicenseHeaders)
	mg.Deps(devtools.CheckNoChanges)
	return nil
}

// Fmt formats code and adds license headers.
func Fmt() {
	mg.Deps(devtools.GoImports.Run)
	mg.Deps(AddLicenseHeaders)
}

// AddLicenseHeaders adds Elastic V2 headers to .go files
func AddLicenseHeaders() error {
	fmt.Println(">> fmt - go-licenser: Adding missing headers")

	mg.Deps(devtools.InstallGoLicenser)

	licenser := gotool.Licenser

	return licenser(
		licenser.License("Elastic"),
	)
}

// CheckLicenseHeaders checks Elastic V2 headers in .go files
func CheckLicenseHeaders() error {
	mg.Deps(devtools.InstallGoLicenser)

	licenser := gotool.Licenser

	return licenser(
		licenser.Check(),
		licenser.License("Elastic"),
	)
}

// Notice generates a NOTICE.txt file for the module.
func Notice() error {
	return devtools.GenerateNotice(
		filepath.Join("dev-tools", "templates", "notice", "overrides.json"),
		filepath.Join("dev-tools", "templates", "notice", "rules.json"),
		filepath.Join("dev-tools", "templates", "notice", "NOTICE.txt.tmpl"),
	)
}
