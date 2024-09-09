package main

import (
	"fmt"
	"runtime/debug"
)

// PrintVersion prints the build version information.
func PrintVersion() {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		fmt.Println("No build info available")
		return
	}

	fmt.Println("Build Info:")
	fmt.Printf("Go Version: %s\n", info.GoVersion)
	for _, setting := range info.Settings {
		fmt.Printf("%s: %s\n", setting.Key, setting.Value)
	}
}
