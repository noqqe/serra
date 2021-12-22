// Package main provides a typing test
package main

import (
	"flag"

	"github.com/noqqe/archivar/src/archivar"
)

// Main Loop
func main() {

	var set_file string

	// flags declaration using flag package
	flag.StringVar(&set_file, "f", "inventory/set-1.yml", "Specify set file")
	flag.Parse()

	archivar.Start(set_file)
}
