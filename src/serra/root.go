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
	LogMessage(fmt.Sprintf("Serra %v\n", version), "green")

	client := storage_connect()
	coll := &Collection{client.Database("serra").Collection("cards")}

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
	LogMessage(fmt.Sprintf("Serra %v\n", version), "green")

	client := storage_connect()
	coll := &Collection{client.Database("serra").Collection("cards")}

	// Loop over different cards
	for _, card := range cards {
		// Fetch card from scryfall
		c, err := find_card_by_setcollectornumber(coll, strings.Split(card, "/")[0], strings.Split(card, "/")[1])
		if err != nil {
			LogMessage(fmt.Sprintf("%v", err), "red")
			continue
		}

		if c.SerraCount > 1 {
			// update
			modify_count_of_card(coll, c, -1)
		} else {
			coll.storage_remove(bson.M{"_id": c.ID})
			LogMessage(fmt.Sprintf("\"%s\" (%.2f Eur) removed from the Collection.", c.Name, c.Prices.Eur), "green")
		}
		// delete

	}
	storage_disconnect(client)
}

func Cards() {
	LogMessage(fmt.Sprintf("Serra %v\n", version), "green")

	client := storage_connect()
	coll := &Collection{client.Database("serra").Collection("cards")}

	sort := bson.D{{"name", 1}}
	filter := bson.D{{}}
	cards, _ := coll.storage_find(filter, sort)

	for _, card := range cards {
		fmt.Printf("%s (%s) %.2f\n", card.Name, card.Set, card.Prices.Eur)
	}
	storage_disconnect(client)
}

func Sets() {
	LogMessage(fmt.Sprintf("Serra %v\n", version), "green")

	client := storage_connect()
	coll := &Collection{client.Database("serra").Collection("cards")}

	groupStage := bson.D{
		{"$group", bson.D{
			{"_id", "$setname"},
			{"value", bson.D{{"$sum", bson.D{{"$multiply", bson.A{"$prices.eur", "$serra_count"}}}}}},
			{"count", bson.D{{"$sum", bson.D{{"$multiply", bson.A{1.0, "$serra_count"}}}}}},
			{"release", bson.D{{"$last", "$releasedat"}}},
		}},
	}
	sortStage := bson.D{
		{"$sort", bson.D{
			{"release", 1},
		}}}

	sets, _ := coll.storage_aggregate(mongo.Pipeline{groupStage, sortStage})
	for _, set := range sets {
		fmt.Printf("* %s %s (%.2f Eur) %.0f\n", set["release"].(string)[0:4], set["_id"], set["value"], set["count"])
	}
	storage_disconnect(client)

}

func ShowSet(setname string) error {
	LogMessage(fmt.Sprintf("Serra %v\n", version), "green")

	client := storage_connect()
	coll := &Collection{client.Database("serra").Collection("cards")}

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
		fmt.Printf("%dx %s (%s/%d) %.2f EUR\n", card.SerraCount, card.Name, sets[0].Code, card.CollectorNumber, card.Prices.Eur)
	}

	storage_disconnect(client)
	return nil

}

func Update() error {
	LogMessage(fmt.Sprintf("Serra %v\n", version), "green")

	client := storage_connect()

	// update sets
	setscoll := &Collection{client.Database("serra").Collection("sets")}

	sets, _ := fetch_sets()
	for _, set := range sets.Data {
		// setscoll.storage_remove(bson.M{"_id": ""})
		// TODO: lol, no errorhandling, no dup key handling. but its fine. for now.
		setscoll.storage_add_set(&set)
	}

	return nil

	// update cards
	coll := &Collection{client.Database("serra").Collection("cards")}
	sort := bson.D{{"_id", 1}}
	filter := bson.D{{}}
	cards, _ := coll.storage_find(filter, sort)

	for i, card := range cards {
		fmt.Printf("Updating (%d/%d): %s (%s)...\n", i+1, len(cards), card.Name, card.SetName)

		updated_card, err := fetch_card(fmt.Sprintf("%s/%d", card.Set, card.CollectorNumber))
		if err != nil {
			LogMessage(fmt.Sprintf("%v", err), "red")
			continue
		}

		filter := bson.M{"_id": bson.M{"$eq": card.ID}}

		update := bson.M{
			"$set": bson.M{"serra_updated": primitive.NewDateTimeFromTime(time.Now()), "prices": updated_card.Prices},
			"$push": bson.M{"serra_prices": bson.M{"date": primitive.NewDateTimeFromTime(time.Now()),
				"value": updated_card.Prices.Eur}}}

		coll.storage_update(filter, update)
	}

	storage_disconnect(client)
	return nil
}

func Stats() {
	LogMessage(fmt.Sprintf("Serra %v\n", version), "green")

	LogMessage(fmt.Sprintf("Color distribution in Collection"), "green")
	client := storage_connect()
	coll := &Collection{client.Database("serra").Collection("cards")}

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
