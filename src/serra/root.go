package serra

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var version = "unknown"
var count int64
var limit float64
var rarity, set, sort string

var rootCmd = &cobra.Command{
	Version:               version,
	Long:                  `serra - Personal Magic: The Gathering Collection Tracker`,
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
