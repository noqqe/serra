package serra

import (
	"errors"
	"fmt"
	"strconv"

	"go.mongodb.org/mongo-driver/bson"
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
