package serra

import (
	"github.com/spf13/cobra"
)

var (
	Version         = "unknown"
	address         string
	cardType        string
	color           string
	cmc             int64
	count           int64
	detail          bool
	foil            bool
	format          string
	interactive     bool
	limit           float64
	name            string
	oracle          string
	port            uint64
	rarity          string
	reserved        bool
	set             string
	sinceBeginning  bool
	sinceLastUpdate bool
	sortby          string
	unique          bool
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
