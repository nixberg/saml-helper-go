//go:build darwin

package main

import (
	"os/exec"
)

func tryInitiateLogin(url string) {
	exec.Command("open", "--background", url).Run()
}
