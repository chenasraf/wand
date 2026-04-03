package main

import (
	_ "embed"
	"fmt"
	"os"
	"strings"

	"github.com/chenasraf/wand/cmd"
)

//go:embed version.txt
var version string

func main() {
	cmd.Version = strings.TrimSpace(version)
	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
