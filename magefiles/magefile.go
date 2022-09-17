// ⚡ Core Mage Tasks.
package main

import (
	"os"

	"github.com/DelineaXPM/dsv-github-action/magefiles/constants"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
	"github.com/pterm/pterm"
	"github.com/sheldonhull/magetools/ci"
	"github.com/sheldonhull/magetools/tooling"

	// mage:import
	"github.com/sheldonhull/magetools/gittools"
	// mage:import
	"github.com/sheldonhull/magetools/gotools"
	// mage:import
	"github.com/sheldonhull/magetools/precommit"
	//mage:import
	_ "github.com/sheldonhull/magetools/secrets"
)

// Test contains mage tasks for testing.
type Test mg.Namespace

// createDirectories creates the local working directories for build artifacts and tooling.
func createDirectories() error {
	for _, dir := range []string{constants.ArtifactDirectory, constants.CacheDirectory} {
		if err := os.MkdirAll(dir, constants.PermissionUserReadWriteExecute); err != nil {
			pterm.Error.Printf("failed to create dir: [%s] with error: %v\n", dir, err)

			return err
		}
		pterm.Success.Printf("✅ [%s] dir created\n", dir)
	}

	return nil
}

// Init runs multiple tasks to initialize all the requirements for running a project for a new contributor.
func Init() error {
	pterm.DefaultHeader.Println("running Init()")
	mg.SerialDeps(
		Clean,
		createDirectories,
	)

	mg.Deps(
		(gotools.Go{}.Tidy),
	)
	if err := tooling.SilentInstallTools(CIToolList); err != nil {
		return err
	}
	if ci.IsCI() {
		pterm.Debug.Println("CI detected, done with init")
		return nil
	}

	pterm.DefaultSection.Println("Setup Project Specific Tools")
	if err := tooling.SilentInstallTools(ToolList); err != nil {
		return err
	}
	// These can run in parallel as different toolchains.
	mg.Deps(
		(gotools.Go{}.Init),
		(gittools.Gittools{}.Init),
		(precommit.Precommit{}.Init),
	)
	return nil
}

// Clean up after yourself.
func Clean() {
	pterm.Success.Println("Cleaning...")
	for _, dir := range []string{constants.ArtifactDirectory, constants.CacheDirectory} {
		err := os.RemoveAll(dir)
		if err != nil {
			pterm.Error.Printf("failed to removeall: [%s] with error: %v\n", dir, err)
		}
		pterm.Success.Printf("🧹 [%s] dir removed\n", dir)
	}
	mg.Deps(createDirectories)
}

// TestIntegration runs local act cli to test the integration.
func (Test) Integration() error {
	return sh.RunV(
		"act",
		"--job", "integration",
		"--secret-file", constants.SecretFile,
	)
}

// Unit runs go unit tests.
func (Test) Unit() {
	mg.Deps(gotools.Go{}.TestSum("./..."))
}
