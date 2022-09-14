package main

import (
	"os"

	"github.com/DelineaXPM/dsv-github-action/dga"
	"github.com/pterm/pterm"
)

const (
	// ExitFailure is exit code sent for failed task.
	exitFailure = 1
	// ExitSuccess is exit code sent for running without any error.
	exitSuccess = 0
)
//nolint:gochecknoglobals // ok for providing as version output
var (
    version = "dev"
    commit  = "none"
    date    = "unknown"
)
func main() {
	pterm.Info.Printf("version: %s\n"+"commit: %s\n" + "built: %s\n", version, commit, date)

	if err := dga.Run(); err != nil {
		pterm.Error.Printfln("run(): %v", err)
		os.Exit(exitFailure)
	}
	pterm.Success.Println("complete with success")
	os.Exit(exitSuccess)
}
