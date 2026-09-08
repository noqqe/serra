package serra

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type PriceEntry struct {
	Date      primitive.DateTime `bson:"date"`
	Eur       float64            `json:"eur,string" bson:"eur,float64"`
	EurFoil   float64            `json:"eur_foil,string" bson:"eur_foil,float64"`
	Tix       float64            `json:"tix,string" bson:"tix,float64"`
	Usd       float64            `json:"usd,string" bson:"usd,float64"`
	UsdEtched float64            `json:"usd_etched,string" bson:"usd_etched,float64"`
	UsdFoil   float64            `json:"usd_foil,string" bson:"usd_foil,float64"`
}

// valueForFinish returns the currency specific value for a given finish
// (nonfoil/foil/etched) of the card. Scryfall does not provide a EUR price
// for etched cards, so that combination always returns 0.
func (c Card) valueForFinish(finish string) float64 {
	eur := getCurrency() == EUR
	switch finish {
	case FinishFoil:
		if eur {
			return c.Prices.EurFoil
		}
		return c.Prices.UsdFoil
	case FinishEtched:
		if eur {
			return 0
		}
		return c.Prices.UsdEtched
	default:
		if eur {
			return c.Prices.Eur
		}
		return c.Prices.Usd
	}
}

// priceEntryForFinish extracts the value relevant to a given finish from a
// full Scryfall price snapshot and normalizes it into the Eur/Usd fields, so
// that a single finish-specific value history can be read back without
// having to know which finish it belongs to.
func priceEntryForFinish(p PriceEntry, finish string) PriceEntry {
	entry := PriceEntry{Date: p.Date}
	switch finish {
	case FinishFoil:
		entry.Eur, entry.Usd = p.EurFoil, p.UsdFoil
	case FinishEtched:
		entry.Usd = p.UsdEtched
	default:
		entry.Eur, entry.Usd = p.Eur, p.Usd
	}
	return entry
}

// priceHistoryForFinish narrows a full Scryfall price history (as cached for
// every card, owned or not) down to the values relevant to a single finish,
// so it can be displayed the same way an inventory entry's value history is.
func priceHistoryForFinish(history []PriceEntry, finish string) []PriceEntry {
	narrowed := make([]PriceEntry, len(history))
	for i, p := range history {
		narrowed[i] = priceEntryForFinish(p, finish)
	}
	return narrowed
}

// Getter for currency specific value
func (c Card) getValue() float64 {
	return c.valueForFinish(FinishNonfoil)
}

// Getter for currency specific value
func (c Card) getFoilValue() float64 {
	return c.valueForFinish(FinishFoil)
}

// Getter for currency specific value
func (c Card) getEtchedValue() float64 {
	return c.valueForFinish(FinishEtched)
}

// colorizeValue formats a value, color coded by how expensive it is.
func colorizeValue(value float64) string {
	if value > 10 {
		go playSoundCash()
		return Red("%.2f", value)
	}
	if value > 5 {
		go playSoundCash()
		return Yellow("%.2f", value)
	}
	if value > 1 {
		go playSoundCash()
		return Green("%.2f", value)
	}

	return fmt.Sprintf("%.2f", value)
}

// Getter for currency specific, colored value
func (c Card) getColoredValueForFinish(finish string) string {
	return colorizeValue(c.valueForFinish(finish))
}

// Getter for currency specific, colored value
func (c Card) getColoredValue() string {
	return c.getColoredValueForFinish(FinishNonfoil)
}

// Getter for currency specific, colored value
func (c Card) getColoredFoilValue() string {
	return c.getColoredValueForFinish(FinishFoil)
}

// http getter for scryfall api with custom headers
func queryScryfall(url string) (*http.Response, error) {
	client := &http.Client{}
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json;q=0.9,*/*;q=0.8")
	req.Header.Set("User-Agent", fmt.Sprintf("Serra/%s", Version))
	return client.Do(req)
}

// fetchCard fetches a card from scryfall api and return a Card struct
func fetchCard(setName, collectorNumber string) (*Card, error) {
	resp, err := queryScryfall(fmt.Sprintf("https://api.scryfall.com/cards/%s/%s", setName, collectorNumber))
	if err != nil {
		log.Fatalln(err)
		return &Card{}, err
	}

	if resp.StatusCode != 200 {
		return &Card{}, fmt.Errorf("Card %s/%s not found", setName, collectorNumber)
	}

	//we read the response body on the line below.
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

	return val, nil
}

func fetchSets() (*SetList, error) {
	resp, err := queryScryfall("https://api.scryfall.com/sets")
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
