package main

import "muster/cmd"

// Version can be set during build with -ldflags
var version = "0.0.71"

func main() {
	cmd.SetVersion(version)
	cmd.Execute()
}
