package archivar

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
)

type Card struct {
	Artist          string   `json:"artist"`
	ArtistIds       []string `json:"artist_ids"`
	Booster         bool     `json:"booster"`
	BorderColor     string   `json:"border_color"`
	CardBackID      string   `json:"card_back_id"`
	Cmc             int64    `json:"cmc"`
	CollectorNumber string   `json:"collector_number"`
	ColorIdentity   []string `json:"color_identity"`
	Colors          []string `json:"colors"`
	Digital         bool     `json:"digital"`
	EdhrecRank      int64    `json:"edhrec_rank"`
	Finishes        []string `json:"finishes"`
	Foil            bool     `json:"foil"`
	Frame           string   `json:"frame"`
	FullArt         bool     `json:"full_art"`
	Games           []string `json:"games"`
	HighresImage    bool     `json:"highres_image"`
	ID              string   `json:"id"`
	IllustrationID  string   `json:"illustration_id"`
	ImageStatus     string   `json:"image_status"`
	ImageUris       struct {
		ArtCrop    string `json:"art_crop"`
		BorderCrop string `json:"border_crop"`
		Large      string `json:"large"`
		Normal     string `json:"normal"`
		Png        string `json:"png"`
		Small      string `json:"small"`
	} `json:"image_uris"`
	Keywords   []interface{} `json:"keywords"`
	Lang       string        `json:"lang"`
	Layout     string        `json:"layout"`
	Legalities struct {
		Alchemy         string `json:"alchemy"`
		Brawl           string `json:"brawl"`
		Commander       string `json:"commander"`
		Duel            string `json:"duel"`
		Future          string `json:"future"`
		Gladiator       string `json:"gladiator"`
		Historic        string `json:"historic"`
		Historicbrawl   string `json:"historicbrawl"`
		Legacy          string `json:"legacy"`
		Modern          string `json:"modern"`
		Oldschool       string `json:"oldschool"`
		Pauper          string `json:"pauper"`
		Paupercommander string `json:"paupercommander"`
		Penny           string `json:"penny"`
		Pioneer         string `json:"pioneer"`
		Premodern       string `json:"premodern"`
		Standard        string `json:"standard"`
		Vintage         string `json:"vintage"`
	} `json:"legalities"`
	ManaCost      string        `json:"mana_cost"`
	MultiverseIds []interface{} `json:"multiverse_ids"`
	Name          string        `json:"name"`
	Nonfoil       bool          `json:"nonfoil"`
	Object        string        `json:"object"`
	OracleID      string        `json:"oracle_id"`
	OracleText    string        `json:"oracle_text"`
	Oversized     bool          `json:"oversized"`
	Prices        struct {
		Eur       interface{} `json:"eur"`
		EurFoil   interface{} `json:"eur_foil"`
		Tix       interface{} `json:"tix"`
		Usd       interface{} `json:"usd"`
		UsdEtched interface{} `json:"usd_etched"`
		UsdFoil   interface{} `json:"usd_foil"`
	} `json:"prices"`
	PrintedName     string `json:"printed_name"`
	PrintedText     string `json:"printed_text"`
	PrintedTypeLine string `json:"printed_type_line"`
	PrintsSearchURI string `json:"prints_search_uri"`
	Promo           bool   `json:"promo"`
	PurchaseUris    struct {
		Cardhoarder string `json:"cardhoarder"`
		Cardmarket  string `json:"cardmarket"`
		Tcgplayer   string `json:"tcgplayer"`
	} `json:"purchase_uris"`
	Rarity      string `json:"rarity"`
	RelatedUris struct {
		Edhrec                    string `json:"edhrec"`
		Mtgtop8                   string `json:"mtgtop8"`
		TcgplayerInfiniteArticles string `json:"tcgplayer_infinite_articles"`
		TcgplayerInfiniteDecks    string `json:"tcgplayer_infinite_decks"`
	} `json:"related_uris"`
	ReleasedAt     string `json:"released_at"`
	Reprint        bool   `json:"reprint"`
	Reserved       bool   `json:"reserved"`
	RulingsURI     string `json:"rulings_uri"`
	ScryfallSetURI string `json:"scryfall_set_uri"`
	ScryfallURI    string `json:"scryfall_uri"`
	Set            string `json:"set"`
	SetID          string `json:"set_id"`
	SetName        string `json:"set_name"`
	SetSearchURI   string `json:"set_search_uri"`
	SetType        string `json:"set_type"`
	SetURI         string `json:"set_uri"`
	StorySpotlight bool   `json:"story_spotlight"`
	Textless       bool   `json:"textless"`
	TypeLine       string `json:"type_line"`
	URI            string `json:"uri"`
	Variation      bool   `json:"variation"`
}

func fetch(path string) *Card {
	resp, err := http.Get(fmt.Sprintf("https://api.scryfall.com/cards/%s/", path))
	if err != nil {
		log.Fatalln(err)
	}

	//We Read the response body on the line below.
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatalln(err)
	}

	r := bytes.NewReader(body)
	decoder := json.NewDecoder(r)
	val := &Card{}
	err = decoder.Decode(val)
	return val
}
