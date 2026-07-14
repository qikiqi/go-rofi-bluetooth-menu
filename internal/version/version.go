// Package version prints build/version information derived from the
// binary's embedded debug.BuildInfo.
package version

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"
)

// Print writes the program's version info to stdout. It returns an error only
// if the build info can't be read.
func Print() error {
	buildInfo, ok := debug.ReadBuildInfo()
	if !ok {
		return errors.New("no build info available")
	}

	// Module version, e.g. "v1.2.3" or "v0.0.0-20250806123456-abcd1234".
	// The Go toolchain fills this in when building a module.
	version := strings.TrimPrefix(buildInfo.Main.Version, "v")
	goVersion := buildInfo.GoVersion

	revision := "unknown"
	buildTime := "unknown"

	for _, s := range buildInfo.Settings {
		switch s.Key {
		case "vcs.revision":
			if len(s.Value) >= 8 {
				revision = s.Value[:8]
			} else {
				revision = s.Value
			}
		case "vcs.time":
			buildTime = s.Value
		}
	}

	prog := filepath.Base(os.Args[0])
	fmt.Printf(
		"%s version %s (built with %s, commit %s on %s)\n",
		prog, version, goVersion, revision, buildTime,
	)

	return nil
}
