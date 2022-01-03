package serra

import (
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
)

func increase_count_of_card(coll *Collection, c *Card) error {

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
		"$set": bson.M{"serra_count": stored_card.SerraCount + 1},
	}
	coll.storage_update(update_filter, update)

	LogMessage(fmt.Sprintf("Updating Card \"%s\" amount to %d", stored_card.Name, stored_card.SerraCount), "purple")
	return nil
}
