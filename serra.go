// Package main provides a typing test
package main

import (
	"fmt"

	"github.com/docopt/docopt-go"
	"github.com/noqqe/serra/src/serra"
)

//[--count:1 <card>:[mmq/1] <set>:<nil> add:true cards:false remove:false set:false sets:false stats:false update:false]

var opts struct {
	Add     bool     `docopt:"add"`
	Remove  bool     `docopt:"remove"`
	Cards   bool     `docopt:"cards"`
	Card    bool     `docopt:"card"`
	Set     bool     `docopt:"set"`
	Sets    bool     `docopt:"sets"`
	Stats   bool     `docopt:"stats"`
	Missing bool     `docopt:"missing"`
	Update  bool     `docopt:"update"`
	CardId  []string `docopt:"<cardid>"`
	SetCode string   `docopt:"<setcode>,--set"`
	Count   int64    `docopt:"--count"`
	Sort    string   `docopt:"--sort"`
	Rarity  string   `docopt:"--rarity"`
}

// Main Loop
func main() {

	usage := `Serra

Usage:
  serra add <cardid>... [--count=<number>]
  serra remove <cardid>...
  serra cards [--rarity=<rarity>] [--set=<setcode>] [--sort=<sort>]
  serra card <cardid>...
  serra missing <setcode>
  serra set <setcode>
  serra sets
  serra update
  serra stats

Options:
  -h --help					Show this screen.
	--count=<number>	Count of card to add.  [default: 1].
  --version					Show version.
`

	args, _ := docopt.ParseDoc(usage)
	err := args.Bind(&opts)
	if err != nil {
		fmt.Println(err)
	}

	serra.Banner()
	if opts.Add {
		serra.Add(opts.CardId, opts.Count)
	} else if opts.Remove {
		serra.Remove(opts.CardId)
	} else if opts.Cards {
		serra.Cards(opts.Rarity, opts.SetCode, opts.Sort)
	} else if opts.Card {
		serra.ShowCard(opts.CardId)
	} else if opts.Sets {
		serra.Sets()
	} else if opts.Missing {
		serra.Missing(opts.SetCode)
	} else if opts.Set {
		serra.ShowSet(opts.SetCode)
	} else if opts.Update {
		serra.Update()
	} else if opts.Stats {
		serra.Stats()
	}

}
