package serra

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	Version         = "unknown"
	address         string
	cardType        string
	count           int64
	detail          bool
	foil            bool
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
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
