package serra

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func modify_count_of_card(coll *Collection, c *Card, amount int64) error {

	// find already existing card
	sort := bson.D{{"_id", 1}}
	search_filter := bson.D{{"_id", c.ID}}
	stored_cards, err := coll.storage_find(search_filter, sort)
	if err != nil {
		return err
	}
	stored_card := stored_cards[0]

	// update card amount
	update_filter := bson.M{"_id": bson.M{"$eq": c.ID}}
	update := bson.M{
		"$set": bson.M{"serra_count": stored_card.SerraCount + amount},
	}
	coll.storage_update(update_filter, update)

	LogMessage(fmt.Sprintf("Updating Card \"%s\" amount to %d", stored_card.Name, stored_card.SerraCount+amount), "purple")
	return nil
}

func find_card_by_setcollectornumber(coll *Collection, setcode string, collectornumber string) (*Card, error) {

	sort := bson.D{{"_id", 1}}
	c, _ := strconv.ParseInt(collectornumber, 10, 64)
	search_filter := bson.D{{"set", setcode}, {"collectornumber", c}}
	stored_cards, err := coll.storage_find(search_filter, sort)
	if err != nil {
		return &Card{}, err
	}

	if len(stored_cards) < 1 {
		return &Card{}, errors.New("Card not found")
	}

	return &stored_cards[0], nil
}

func stringToTime(s primitive.DateTime) string {
	return time.UnixMilli(int64(s)).Format("2006-01-02")
}

func show_card_details(card *Card) error {
	fmt.Printf("* %dx %s%s%s (%s/%s)\n", card.SerraCount, Purple, card.Name, Reset, card.Set, card.CollectorNumber)
	fmt.Printf("  Added: %s\n", stringToTime(card.SerraCreated))
	fmt.Printf("  Current Value: %s%.2f EUR%s\n", Yellow, card.Prices.Eur, Reset)
	fmt.Printf("  History:\n")
	for _, e := range card.SerraPrices {
		fmt.Printf("  * %s %.2f EUR\n", stringToTime(e.Date), e.Value)
	}
	fmt.Println()
	return nil
}

func convert_mana_symbols(sym []interface{}) string {
	var mana string

	if len(sym) == 0 {
		mana = mana + "\U0001F6AB" //probibited sign for lands
	}

	for _, v := range sym {
		switch v {
		case "B":
			mana = mana + "\U000026AB" //black
		case "R":
			mana = mana + "\U0001F534" //red
		case "G":
			mana = mana + "\U0001F7E2" //green
		case "U":
			mana = mana + "\U0001F535" //blue
		case "W":
			mana = mana + "\U000026AA" //white
		}
	}
	return mana

}
