// Package serra
//
// It implements base functions and also cli wrappers
// The entire tool consists only of this one package.
package serra

import (
	"github.com/spf13/cobra"
)

var (
	Version         = "unknown"
	address         string
	allCards        bool
	artist          string
	cardType        string
	setType         string
	color           string
	cmc             int64
	condition       string
	count           int64
	detail          bool
	etched          bool
	foil            bool
	format          string
	interactive     bool
	is              string
	isNot           string
	language        string
	legal           string
	limit           float64
	name            string
	oracle          string
	port            uint64
	rarity          string
	reserved        bool
	set             string
	sinceBeginning  bool
	sinceLastUpdate bool
	sortBy          string
	unique          bool
	otoCtx          string
)

var rootCmd = &cobra.Command{
	Version:               Version,
	Long:                  `serra - Magic: The Gathering Collection Tracker`,
	Use:                   "serra",
	DisableFlagsInUseLine: true,
	SilenceErrors:         true,
}

func Execute() {

	l := Logger()
	if err := rootCmd.Execute(); err != nil {
		l.Fatal(err)
	}
}
