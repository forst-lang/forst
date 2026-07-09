package main

import "os/exec"

func main() {
	argv := []string{"true", "extra"}
	cmd := exec.Command(argv[0], argv[1:]...)
	cmd.Run()
	cmd.ProcessState.ExitCode()
}
