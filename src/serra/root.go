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

func Cards(rarity, set string) {

	client := storage_connect()
	coll := &Collection{client.Database("serra").Collection("cards")}
	defer storage_disconnect(client)

	filter := bson.D{}

	switch rarity {
	case "uncommon":
		filter = append(filter, bson.E{"rarity", "uncommon"})
	case "common":
		filter = append(filter, bson.E{"rarity", "common"})
	case "rare":
		filter = append(filter, bson.E{"rarity", "rare"})
	}

	if len(set) > 0 {
		filter = append(filter, bson.E{"set", set})
	}

	cards, _ := coll.storage_find(filter, bson.D{{"name", 1}})

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
	setscoll := &Collection{client.Database("serra").Collection("sets")}
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
		setobj, _ := find_set_by_code(setscoll, set["code"].(string))
		fmt.Printf("* %s %s%s%s (%s%s%s)\n", set["release"].(string)[0:4], Purple, set["_id"], Reset, Cyan, set["code"], Reset)
		fmt.Printf("  Cards: %s%d/%d%s Total: %.0f \n", Yellow, set["unique"], setobj.CardCount, Reset, set["count"])
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

	// set values
	matchStage = bson.D{
		{"$match", bson.D{
			{"set", setname},
		}},
	}
	groupStage = bson.D{
		{"$group", bson.D{
			{"_id", "$rarity"},
			{"count", bson.D{{"$sum", bson.D{{"$multiply", bson.A{1.0, "$serra_count"}}}}}},
		}}}

	sortStage := bson.D{
		{"$sort", bson.D{
			{"_id", 1},
		}}}
	rar, _ := coll.storage_aggregate(mongo.Pipeline{matchStage, groupStage, sortStage})

	LogMessage(fmt.Sprintf("%s", sets[0].Name), "green")
	LogMessage(fmt.Sprintf("Set Cards: %d/%d", len(cards), sets[0].CardCount), "normal")
	LogMessage(fmt.Sprintf("Total Cards: %.0f", stats[0]["count"]), "normal")
	LogMessage(fmt.Sprintf("Total Value: %.2f EUR", stats[0]["value"]), "normal")
	LogMessage(fmt.Sprintf("Released: %s", sets[0].ReleasedAt), "normal")
	LogMessage(fmt.Sprintf("Rares: %.0f", rar[1]["count"]), "normal")
	LogMessage(fmt.Sprintf("Uncommons: %.0f", rar[2]["count"]), "normal")
	LogMessage(fmt.Sprintf("Commons: %.0f", rar[0]["count"]), "normal")
	fmt.Printf("\n%sPrice History:%s\n", Pink, Reset)
	for _, e := range sets[0].SerraPrices {
		fmt.Printf("* %s %.2f EUR\n", stringToTime(e.Date), e.Value)
	}

	fmt.Printf("\n%sMost valuable cards%s\n", Pink, Reset)
	for i := 0; i < 10; i++ {
		card := cards[i]
		fmt.Printf("* %dx %s%s%s (%s/%s) %s%.2f EUR%s\n", card.SerraCount, Purple, card.Name, Reset, sets[0].Code, card.CollectorNumber, Yellow, card.Prices.Eur, Reset)
	}

	return nil
}

func Update() error {

	client := storage_connect()
	defer storage_disconnect(client)

	// update sets
	setscoll := &Collection{client.Database("serra").Collection("sets")}
	coll := &Collection{client.Database("serra").Collection("cards")}

	sets, _ := fetch_sets()
	for i, set := range sets.Data {
		setscoll.storage_add_set(&set)
		cards, _ := coll.storage_find(bson.D{{"set", set.Code}}, bson.D{{"_id", 1}})

		// if no cards in collection for this set, skip it
		if len(cards) == 0 {
			continue
		}

		fmt.Printf("Updating Set (%d/%d): %s (%s)...\n", i+1, len(sets.Data), set.Name, set.Code)

		for y, card := range cards {
			fmt.Printf("Updating Card %s (%d/%d): %s ...\n", card.SetName, y+1, len(cards), card.Name)

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

		// update set value sum

		// calculate value summary
		matchStage := bson.D{
			{"$match", bson.D{
				{"set", set.Code},
			}},
		}
		groupStage := bson.D{
			{"$group", bson.D{
				{"_id", "$set"},
				{"value", bson.D{{"$sum", bson.D{{"$multiply", bson.A{"$prices.eur", "$serra_count"}}}}}},
			}}}
		setvalue, _ := coll.storage_aggregate(mongo.Pipeline{matchStage, groupStage})

		// do the update

		set_update := bson.M{
			"$set": bson.M{"serra_updated": primitive.NewDateTimeFromTime(time.Now())},
			"$push": bson.M{"serra_prices": bson.M{"date": primitive.NewDateTimeFromTime(time.Now()),
				"value": setvalue[0]["value"]}},
		}
		fmt.Printf("Updating Set value: %s (%s) to %.02f EUR\n", set.Name, set.Code, setvalue[0]["value"])
		setscoll.storage_update(bson.M{"code": bson.M{"$eq": set.Code}}, set_update)
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
			{"count", bson.D{{"$sum", bson.D{{"$multiply", bson.A{1.0, "$serra_count"}}}}}},
		}}}

	sortStage := bson.D{
		{"$sort", bson.D{
			{"count", -1},
		}}}
	sets, _ := coll.storage_aggregate(mongo.Pipeline{groupStage, sortStage})
	for _, set := range sets {
		x, _ := set["_id"].(primitive.A)
		s := []interface{}(x)
		fmt.Printf("* %s %.0f\n", convert_mana_symbols(s), set["count"])
	}

	statsGroup := bson.D{
		{"$group", bson.D{
			{"_id", nil},
			{"value", bson.D{{"$sum", bson.D{{"$multiply", bson.A{"$prices.eur", "$serra_count"}}}}}},
			{"count", bson.D{{"$sum", bson.D{{"$multiply", bson.A{1.0, "$serra_count"}}}}}},
			{"unique", bson.D{{"$sum", 1}}},
		}},
	}
	stats, _ := coll.storage_aggregate(mongo.Pipeline{statsGroup})

	fmt.Printf("\n%sOverall %s\n", Green, Reset)
	fmt.Printf("Total Cards: %s%.0f%s\n", Yellow, stats[0]["count"], Reset)
	fmt.Printf("Unique Cards: %s%d%s\n", Purple, stats[0]["unique"], Reset)
	fmt.Printf("Total Value: %s%.2f%s\n", Pink, stats[0]["value"], Reset)

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
