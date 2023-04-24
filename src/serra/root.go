package serra

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	Version         = "unknown"
	count           int64
	detail          bool
	limit           float64
	interactive     bool
	name            string
	oracle          string
	rarity          string
	set             string
	sinceBeginning  bool
	sinceLastUpdate bool
	sortby          string
	cardType        string
	unique          bool
	foil            bool
	address         string
	port            uint64
)

var rootCmd = &cobra.Command{
	Version:               Version,
	Long:                  `serra - Magic: The Gathering Collection Tracker`,
	Use:                   "serra",
	DisableFlagsInUseLine: true,
	SilenceErrors:         true,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
