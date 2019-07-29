package version

import (
	"flag"
	"log"
	"os"
)

// PrintVersionOrContinue will print git commit and exit with os.Exit(0) if CLI v flag is present
func PrintVersionOrContinue(version, gitCommit string) {
	versionFlag := flag.Bool("v", false, "Print the current version and exit")

	flag.Parse()

	switch {
	case *versionFlag:
		log.Printf("Current build version: %s (git commit: %s)", version, gitCommit)
		os.Exit(0)
	}
}
