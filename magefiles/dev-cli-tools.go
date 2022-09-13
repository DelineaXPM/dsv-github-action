package main

// ToolList is a list of tools that are installed as binaries for development usage.
// This list gets installed to go bin directory once `mage init` is run.
// This is for binaries that need to be invoked as cli tools, not packages.
var ToolList = []string{ //nolint:gochecknoglobals // ok to be global for tooling setup
	"mvdan.cc/gofumpt@latest",
	"github.com/iwittkau/mage-select@latest",
	"github.com/mfridman/tparse@latest", // Tparse provides nice formatted go test console output.
	"github.com/rakyll/gotest@latest",   // Gotest is a wrapper for running Go tests via command line with support for colors to make it more readable.
	"github.com/gechr/yamlfmt@latest",   // Yamlfmt provides formatting standards for yaml files.
}

// CIToolList is the minimum needed for CI invocation like GitHub actions.
var CIToolList = []string{ //nolint:gochecknoglobals // ok to be global for tooling setup
	"github.com/golangci/golangci-lint/cmd/golangci-lint@latest",
	"github.com/goreleaser/goreleaser@latest",
	"gotest.tools/gotestsum@latest", // Gotestsum provides improved console output for tests as well as additional test output for CI systems.

}
