package serra

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Card struct {
	// Added by Serra
	SerraCount       int64              `bson:"serra_count"`
	SerraCountFoil   int64              `bson:"serra_count_foil"`
	SerraCountEtched int64              `bson:"serra_count_etched"`
	SerraPrices      []PriceEntry       `bson:"serra_prices"`
	SerraCreated     primitive.DateTime `bson:"serra_created"`
	SerraUpdated     primitive.DateTime `bson:"serra_updated"`

	Artist          string   `json:"artist"`
	ArtistIds       []string `json:"artist_ids"`
	Booster         bool     `json:"booster"`
	BorderColor     string   `json:"border_color"`
	CardBackID      string   `json:"card_back_id"`
	CardmarketID    float64  `json:"cardmarket_id"`
	Cmc             float64  `json:"cmc"`
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
	ID              string   `json:"id" bson:"_id"`
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
	ManaCost        string        `json:"mana_cost"`
	MultiverseIds   []interface{} `json:"multiverse_ids"`
	Name            string        `json:"name"`
	Nonfoil         bool          `json:"nonfoil"`
	Object          string        `json:"object"`
	OracleID        string        `json:"oracle_id"`
	OracleText      string        `json:"oracle_text"`
	Oversized       bool          `json:"oversized"`
	Prices          PriceEntry    `json:"prices"`
	PrintedName     string        `json:"printed_name"`
	PrintedText     string        `json:"printed_text"`
	PrintedTypeLine string        `json:"printed_type_line"`
	PrintsSearchURI string        `json:"prints_search_uri"`
	Promo           bool          `json:"promo"`
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
	ReleasedAt     string  `json:"released_at"`
	Reprint        bool    `json:"reprint"`
	Reserved       bool    `json:"reserved"`
	RulingsURI     string  `json:"rulings_uri"`
	ScryfallSetURI string  `json:"scryfall_set_uri"`
	ScryfallURI    string  `json:"scryfall_uri"`
	Set            string  `json:"set"`
	SetID          string  `json:"set_id"`
	SetName        string  `json:"set_name"`
	SetSearchURI   string  `json:"set_search_uri"`
	SetType        string  `json:"set_type"`
	SetURI         string  `json:"set_uri"`
	StorySpotlight bool    `json:"story_spotlight"`
	Textless       bool    `json:"textless"`
	TCGPlayerID    float64 `json:"tcgplayer_id"`
	TypeLine       string  `json:"type_line"`
	URI            string  `json:"uri"`
	Variation      bool    `json:"variation"`
}

// Getter for currency specific value
func (c Card) getValue(foil bool) float64 {
	if getCurrency() == EUR {
		if foil {
			return c.Prices.EurFoil
		}
		return c.Prices.Eur
	}
	if foil {
		return c.Prices.UsdFoil
	}
	return c.Prices.Usd
}

type PriceEntry struct {
	Date      primitive.DateTime `bson:"date"`
	Eur       float64            `json:"eur,string" bson:"eur,float64"`
	EurFoil   float64            `json:"eur_foil,string" bson:"eur_foil,float64"`
	Tix       float64            `json:"tix,string" bson:"tix,float64"`
	Usd       float64            `json:"usd,string" bson:"usd,float64"`
	UsdEtched float64            `json:"usd_etched,string" bson:"usd_etched,float64"`
	UsdFoil   float64            `json:"usd_foil,string" bson:"usd_foil,float64"`
}

// Sets

type SetList struct {
	Data []Set `json:"data"`
}

type Set struct {
	SerraPrices  []PriceEntry       `bson:"serra_prices"`
	SerraCreated primitive.DateTime `bson:"serra_created"`
	SerraUpdated primitive.DateTime `bson:"serra_updated"`
	CardCount    int64              `json:"card_count" bson:"cardcount"`
	Code         string             `json:"code"`
	Digital      bool               `json:"digital"`
	FoilOnly     bool               `json:"foil_only"`
	IconSvgURI   string             `json:"icon_svg_uri"`
	ID           string             `json:"id" bson:"_id"`
	Name         string             `json:"name"`
	NonfoilOnly  bool               `json:"nonfoil_only"`
	Object       string             `json:"object"`
	ReleasedAt   string             `json:"released_at"`
	ScryfallURI  string             `json:"scryfall_uri"`
	SearchURI    string             `json:"search_uri"`
	SetType      string             `json:"set_type"`
	TcgplayerID  int64              `json:"tcgplayer_id"`
	URI          string             `json:"uri"`
}

func fetchCard(setName, collectorNumber string) (*Card, error) {
	resp, err := http.Get(fmt.Sprintf("https://api.scryfall.com/cards/%s/%s/", setName, collectorNumber))
	if err != nil {
		log.Fatalln(err)
		return &Card{}, err
	}

	if resp.StatusCode != 200 {
		return &Card{}, fmt.Errorf("Card %s/%s not found", setName, collectorNumber)
	}

	//We Read the response body on the line below.
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("%s", err)
		return &Card{}, err
	}

	r := bytes.NewReader(body)
	decoder := json.NewDecoder(r)
	val := &Card{}

	err = decoder.Decode(val)
	if err != nil {
		log.Fatalf("%s", err)
	}

	// Set created Time
	val.SerraCreated = primitive.NewDateTimeFromTime(time.Now())

	// Increase Price
	val.Prices.Date = primitive.NewDateTimeFromTime(time.Now())
	val.SerraPrices = append(val.SerraPrices, val.Prices)

	return val, nil
}

func fetchSets() (*SetList, error) {
	// TODO better URL Building...
	resp, err := http.Get("https://api.scryfall.com/sets")
	if err != nil {
		log.Fatalln(err)
		return &SetList{}, err
	}

	if resp.StatusCode != 200 {
		return &SetList{}, fmt.Errorf("/sets not found")
	}

	//We Read the response body on the line below.
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalln(err)
		return &SetList{}, err
	}

	r := bytes.NewReader(body)
	decoder := json.NewDecoder(r)
	val := &SetList{}

	err = decoder.Decode(val)
	if err != nil {
		log.Fatalln(err)
	}

	return val, nil
}
