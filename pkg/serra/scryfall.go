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

type PriceEntry struct {
	Date      primitive.DateTime `bson:"date"`
	Eur       float64            `json:"eur,string" bson:"eur,float64"`
	EurFoil   float64            `json:"eur_foil,string" bson:"eur_foil,float64"`
	Tix       float64            `json:"tix,string" bson:"tix,float64"`
	Usd       float64            `json:"usd,string" bson:"usd,float64"`
	UsdEtched float64            `json:"usd_etched,string" bson:"usd_etched,float64"`
	UsdFoil   float64            `json:"usd_foil,string" bson:"usd_foil,float64"`
}

// Getter for currency specific value
func (c Card) getValue() float64 {
	if getCurrency() == EUR {
		return c.Prices.Eur
	}
	return c.Prices.Usd
}

// Getter for currency specific value
func (c Card) getFoilValue() float64 {
	if getCurrency() == EUR {
		return c.Prices.EurFoil
	}
	return c.Prices.UsdFoil
}

// Getter for currency specific value
func (c Card) getColoredValue() string {
	var value float64
	if getCurrency() == EUR {
		value = c.Prices.Eur
	} else {
		value = c.Prices.Usd
	}

	if value > 10 {
		return Red("%.2f", value)
	}
	if value > 5 {
		return Yellow("%.2f", value)
	}
	if value > 1 {
		return Green("%.2f", value)
	}

	return fmt.Sprintf("%.2f", value)

}

// Getter for currency specific value
func (c Card) getColoredFoilValue() string {
	var value float64
	if getCurrency() == EUR {
		value = c.Prices.EurFoil
	} else {
		value = c.Prices.UsdFoil
	}

	if value > 10 {
		return Red("%.2f", value)
	}
	if value > 5 {
		return Yellow("%.2f", value)
	}
	if value > 1 {
		return Green("%.2f", value)
	}

	return fmt.Sprintf("%.2f", value)

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

	// Set created Time
	val.SerraCreated = primitive.NewDateTimeFromTime(time.Now())

	// Increase Price
	val.Prices.Date = primitive.NewDateTimeFromTime(time.Now())
	val.SerraPrices = append(val.SerraPrices, val.Prices)

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
