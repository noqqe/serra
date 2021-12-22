// Package main provides a typing test
package main

import (
	"github.com/docopt/docopt-go"
	"github.com/noqqe/archivar/src/archivar"
)

// Main Loop
func main() {

	usage := `Archivar

Usage:
  archivar new <path>...
	archivar update <path>...
  archivar value <path>...

Options:
  -h --help     Show this screen.
  --version     Show version.
`

	args, _ := docopt.ParseDoc(usage)

	if args["new"].(bool) {
		archivar.New(args["<path>"].(string))
	}
	if args["update"].(bool) {
		for _, i := range args["<path>"].([]string) {
			archivar.Update(i)
		}
	}

}
