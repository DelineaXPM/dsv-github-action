package main

// ToolList is a list of tools that are installed as binaries for development usage.
// This list gets installed to go bin directory once `mage init` is run.
// This is for binaries that need to be invoked as cli tools, not packages.
var ToolList = []string{
	"mvdan.cc/gofumpt@latest",
	"github.com/iwittkau/mage-select@latest",
	"github.com/mfridman/tparse@latest", // Tparse provides nice formatted go test console output.
	"github.com/rakyll/gotest@latest",   // Gotest is a wrapper for running Go tests via command line with support for colors to make it more readable.
	"github.com/gechr/yamlfmt@latest",   // Yamlfmt provides formatting standards for yaml files.
}

// CIToolList is the minimum needed for CI invocation like GitHub actions.
// Will remove later, as most of these are installed on demand now instead of requiring upfront.
var CIToolList = []string{
	// "github.com/git-chglog/git-chglog/cmd/git-chglog@latest", // Git-chglog provides changelog automation.
	// "github.com/golangci/golangci-lint/cmd/golangci-lint@latest",. // installed by trunk
	// "github.com/goreleaser/goreleaser@latest", // installed on demand
	// "gotest.tools/gotestsum@latest", // Gotestsum provides improved console output for tests as well as additional test output for CI systems.

}
