//go:build !windows

package main

import (
	"os"
	"os/exec"
)

func configureServerCommand(*exec.Cmd) {}

func tieServerToApp(*os.Process) error { return nil }
