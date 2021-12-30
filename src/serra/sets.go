package serra

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
)

type Set struct {
	ArenaCode   string `json:"arena_code"`
	Block       string `json:"block"`
	BlockCode   string `json:"block_code"`
	CardCount   int64  `json:"card_count"`
	Code        string `json:"code"`
	Digital     bool   `json:"digital"`
	FoilOnly    bool   `json:"foil_only"`
	IconSvgURI  string `json:"icon_svg_uri"`
	ID          string `json:"id"`
	MtgoCode    string `json:"mtgo_code"`
	Name        string `json:"name"`
	NonfoilOnly bool   `json:"nonfoil_only"`
	Object      string `json:"object"`
	PrintedSize int64  `json:"printed_size"`
	ReleasedAt  string `json:"released_at"`
	ScryfallURI string `json:"scryfall_uri"`
	SearchURI   string `json:"search_uri"`
	SetType     string `json:"set_type"`
	TcgplayerID int64  `json:"tcgplayer_id"`
	URI         string `json:"uri"`
}

func fetch_set(path string) (*Set, error) {
	// TODO better URL Building...
	resp, err := http.Get(fmt.Sprintf("https://api.scryfall.com/sets/%s/", path))
	if err != nil {
		log.Fatalln(err)
		return &Set{}, err
	}

	if resp.StatusCode != 200 {
		err := errors.New(fmt.Sprintf("set: %s not found", path))
		return &Set{}, err
	}

	//We Read the response body on the line below.
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatalln(err)
		return &Set{}, err
	}

	r := bytes.NewReader(body)
	decoder := json.NewDecoder(r)
	val := &Set{}
	err = decoder.Decode(val)

	return val, nil
}
