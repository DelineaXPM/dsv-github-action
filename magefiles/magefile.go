// ⚡ Core Mage Tasks.
package main

import (
	"os"
	"os/exec"
	"path/filepath"

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

	"github.com/bitfield/script"
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
		(InstallSyft),
	)
	return nil
}

func InstallSyft() error {
	_, err := script.Exec("curl -sSfL https://raw.githubusercontent.com/anchore/syft/main/install.sh").Exec("sh -s -- -b /usr/local/bin").Stdout()
	if err != nil {
		return err
	}
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

// ActIntegration runs local act cli to test the integration.
func (Test) ActIntegration() error {
	binary, err := exec.LookPath("act")
	if err != nil {
		pterm.Error.Println("unable to resolve act as installed. please install via directions here: https://github.com/nektos/act#installation")
		return err
	}

	return sh.RunV(
		binary,
		"--job", "integration",
		"--secret-file", constants.SecretFile,
	)
}

// Unit runs go unit tests.
func (Test) Unit() {
	mg.Deps(gotools.Go{}.TestSum("./..."))
}

// PullImage pulls the latest docker image published to the hub for testing.
func PullImage() error {
	if err := sh.RunV("docker", "pull", constants.DockerRegistryPathQualified+":"+constants.DockerTag); err != nil {
		pterm.Error.Printfln("unable to pull docker image: %v", err)
		return err
	}
	return nil
}

// TestIntegration runs local act cli to test the integration.
func (Test) Integration() error {
	// Mg.Deps(PullImage).
	currentDescribedTag, err := sh.Output("git", "describe", "--tags", "--abbrev=0")
	if err != nil {
		pterm.Error.Printfln("unable to retrieve tag: %v", err)
		return err
	}
	pterm.Info.Printfln("using tag: %s", currentDescribedTag)
	pterm.Info.Println("running docker now")

	wd, err := os.Getwd()
	if err != nil {
		return err
	}
	return sh.RunV(
		"docker",
		"run",
		"--env-file", constants.SecretFile,
		// NOTE: Optional way to invoke: 	"--env", "DSV_RETRIEVE="+"{"secretPath": "ci:tests:dsv-github-action", "secretKey": "secret-01", "outputVariable": "RETURN_VALUE_1"}",.
		"--rm",
		"-v", filepath.Join(wd, ".cache")+":"+"/app/.cache:rw",
		constants.LocalDockerRegistryPathQualified+":"+currentDescribedTag,
	)
}
