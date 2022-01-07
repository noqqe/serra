package serra

import (
	"fmt"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

const (
	version string = "0.0.1"
)

// Add
func Add(cards []string, count int64) error {

	client := storage_connect()
	coll := &Collection{client.Database("serra").Collection("cards")}
	defer storage_disconnect(client)

	// Loop over different cards
	for _, card := range cards {
		// Fetch card from scryfall
		c, err := fetch_card(card)
		if err != nil {
			LogMessage(fmt.Sprintf("%v", err), "red")
			continue
		}

		// Write card to mongodb
		c.SerraCount = count
		err = coll.storage_add(c)

		// If duplicate key, increase count of card
		if mongo.IsDuplicateKeyError(err) {
			modify_count_of_card(coll, c, count)
			continue
		}

		// If error, print error and continue
		if err != nil {
			LogMessage(fmt.Sprintf("%v", err), "red")
			continue
		}

		// Give feedback of successfully added card
		LogMessage(fmt.Sprintf("%dx \"%s\" (%.2f Eur) added to Collection.", c.SerraCount, c.Name, c.Prices.Eur), "green")
	}
	storage_disconnect(client)
	return nil
}

// Remove
func Remove(cards []string) {

	client := storage_connect()
	coll := &Collection{client.Database("serra").Collection("cards")}
	defer storage_disconnect(client)

	// Loop over different cards
	for _, card := range cards {
		// Fetch card from scryfall
		c, err := find_card_by_setcollectornumber(coll, strings.Split(card, "/")[0], strings.Split(card, "/")[1])
		if err != nil {
			LogMessage(fmt.Sprintf("%v", err), "red")
			continue
		}

		if c.SerraCount > 1 {
			modify_count_of_card(coll, c, -1)
		} else {
			coll.storage_remove(bson.M{"_id": c.ID})
			LogMessage(fmt.Sprintf("\"%s\" (%.2f Eur) removed from the Collection.", c.Name, c.Prices.Eur), "green")
		}

	}
}

func Cards() {

	client := storage_connect()
	coll := &Collection{client.Database("serra").Collection("cards")}
	defer storage_disconnect(client)

	sort := bson.D{{"name", 1}}
	filter := bson.D{{}}
	cards, _ := coll.storage_find(filter, sort)

	for _, card := range cards {
		LogMessage(fmt.Sprintf("* %dx %s%s%s (%s/%s) %s%.2f EUR%s", card.SerraCount, Purple, card.Name, Reset, card.Set, card.CollectorNumber, Yellow, card.Prices.Eur, Reset), "normal")
	}
}

func ShowCard(cardids []string) {

	client := storage_connect()
	coll := &Collection{client.Database("serra").Collection("cards")}
	defer storage_disconnect(client)

	for _, v := range cardids {

		cards, _ := coll.storage_find(bson.D{{"set", strings.Split(v, "/")[0]}, {"collectornumber", strings.Split(v, "/")[1]}}, bson.D{{"name", 1}})

		for _, card := range cards {
			show_card_details(&card)
		}
	}
}

func Sets() {

	client := storage_connect()
	coll := &Collection{client.Database("serra").Collection("cards")}
	defer storage_disconnect(client)

	groupStage := bson.D{
		{"$group", bson.D{
			{"_id", "$setname"},
			{"value", bson.D{{"$sum", bson.D{{"$multiply", bson.A{"$prices.eur", "$serra_count"}}}}}},
			{"count", bson.D{{"$sum", bson.D{{"$multiply", bson.A{1.0, "$serra_count"}}}}}},
			{"unique", bson.D{{"$sum", 1}}},
			{"code", bson.D{{"$last", "$set"}}},
			{"release", bson.D{{"$last", "$releasedat"}}},
		}},
	}
	sortStage := bson.D{
		{"$sort", bson.D{
			{"release", 1},
		}}}

	sets, _ := coll.storage_aggregate(mongo.Pipeline{groupStage, sortStage})
	for _, set := range sets {
		fmt.Printf("* %s %s%s%s (%s%s%s)\n", set["release"].(string)[0:4], Purple, set["_id"], Reset, Cyan, set["code"], Reset)
		fmt.Printf("  Cards: %s%d/350%s Total: %.0f \n", Yellow, set["unique"], Reset, set["count"])
		fmt.Printf("  Value: %s%.2f EUR%s\n", Pink, set["value"], Reset)
		fmt.Println()
	}

}

func ShowSet(setname string) error {

	client := storage_connect()
	coll := &Collection{client.Database("serra").Collection("cards")}
	defer storage_disconnect(client)

	// fetch all cards in set
	cards, err := coll.storage_find(bson.D{{"set", setname}}, bson.D{{"prices.eur", -1}})
	if (err != nil) || len(cards) == 0 {
		LogMessage(fmt.Sprintf("Error: Set %s not found or no card in your collection.", setname), "red")
		return err
	}

	// fetch set informations
	setcoll := &Collection{client.Database("serra").Collection("sets")}
	sets, _ := setcoll.storage_find_set(bson.D{{"code", setname}}, bson.D{{"_id", 1}})

	// set values
	matchStage := bson.D{
		{"$match", bson.D{
			{"set", setname},
		}},
	}
	groupStage := bson.D{
		{"$group", bson.D{
			{"_id", "$setname"},
			{"value", bson.D{{"$sum", bson.D{{"$multiply", bson.A{"$prices.eur", "$serra_count"}}}}}},
			{"count", bson.D{{"$sum", bson.D{{"$multiply", bson.A{1.0, "$serra_count"}}}}}},
		}},
	}
	stats, _ := coll.storage_aggregate(mongo.Pipeline{matchStage, groupStage})

	LogMessage(fmt.Sprintf("%s", sets[0].Name), "green")
	LogMessage(fmt.Sprintf("Set Cards: %d/%d", len(cards), sets[0].CardCount), "normal")
	LogMessage(fmt.Sprintf("Total Cards: %.0f", stats[0]["count"]), "normal")
	LogMessage(fmt.Sprintf("Total Value: %.2f EUR", stats[0]["value"]), "normal")
	LogMessage(fmt.Sprintf("Released: %s", sets[0].ReleasedAt), "normal")

	LogMessage(fmt.Sprintf("\nMost valuable cards"), "purple")
	for i := 0; i < 10; i++ {
		card := cards[i]
		fmt.Printf("%dx %s (%s/%s) %.2f EUR\n", card.SerraCount, card.Name, sets[0].Code, card.CollectorNumber, card.Prices.Eur)
	}

	return nil
}

func Update() error {

	client := storage_connect()
	defer storage_disconnect(client)

	// update sets
	setscoll := &Collection{client.Database("serra").Collection("sets")}

	sets, _ := fetch_sets()
	for i, set := range sets.Data {
		fmt.Printf("Updating (%d/%d): %s...\n", i+1, len(sets.Data), set.Name)
		// TODO: lol, no errorhandling, no dup key handling. but its fine. for now.
		setscoll.storage_add_set(&set)
	}

	// update cards
	coll := &Collection{client.Database("serra").Collection("cards")}
	cards, _ := coll.storage_find(bson.D{{}}, bson.D{{"_id", 1}})

	for i, card := range cards {
		fmt.Printf("Updating (%d/%d): %s (%s)...\n", i+1, len(cards), card.Name, card.SetName)

		updated_card, err := fetch_card(fmt.Sprintf("%s/%s", card.Set, card.CollectorNumber))
		if err != nil {
			LogMessage(fmt.Sprintf("%v", err), "red")
			continue
		}

		update := bson.M{
			"$set": bson.M{"serra_updated": primitive.NewDateTimeFromTime(time.Now()), "prices": updated_card.Prices, "collectornumber": updated_card.CollectorNumber},
			"$push": bson.M{"serra_prices": bson.M{"date": primitive.NewDateTimeFromTime(time.Now()),
				"value": updated_card.Prices.Eur}},
		}

		coll.storage_update(bson.M{"_id": bson.M{"$eq": card.ID}}, update)
	}
	return nil
}

func Stats() {

	LogMessage(fmt.Sprintf("Color distribution in Collection"), "green")
	client := storage_connect()
	coll := &Collection{client.Database("serra").Collection("cards")}
	defer storage_disconnect(client)

	groupStage := bson.D{
		{"$group", bson.D{
			{"_id", "$coloridentity"},
			{"count", bson.D{{"$sum", 1}}},
		}}}

	sets, _ := coll.storage_aggregate(mongo.Pipeline{groupStage})
	for _, set := range sets {
		// TODO fix primitiveA Problem with loop and reflect
		fmt.Printf("* %s %d\n", set["_id"], set["count"])
	}

	// LogMessage(fmt.Sprintf("Mana costs in Collection"), "green")
	// groupStage = bson.D{
	// 	{"$group", bson.D{
	// 		{"_id", "$manacost"},
	// 		{"count", bson.D{{"$sum", 1}}},
	// 	}}}
	// m, _ := coll.storage_aggregate(groupStage)

	// for _, manacosts := range m {
	// 	// TODO fix primitiveA Problem with loop and reflect
	// 	fmt.Printf("* %s %d\n", manacosts["_id"], manacosts["count"])
	// }
}
