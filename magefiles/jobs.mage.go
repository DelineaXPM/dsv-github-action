package main

import (
	"github.com/magefile/mage/mg"
)

// Job is a namespace to contain chained sets of automation actions, to reduce the need to chain many commands together for common workflows.
type Job mg.Namespace
