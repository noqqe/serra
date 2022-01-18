package serra

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/schollz/progressbar/v3"
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

func Cards(rarity, set, sort string) {

	var total float64
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

	var sortStage bson.D
	switch sort {
	case "value":
		sortStage = bson.D{{"prices.eur", 1}}
	case "collectornumber":
		sortStage = bson.D{{"collectornumber", 1}}
	case "name":
		sortStage = bson.D{{"name", 1}}
	default:
		sortStage = bson.D{{"name", 1}}
	}

	if len(set) > 0 {
		filter = append(filter, bson.E{"set", set})
	}

	cards, _ := coll.storage_find(filter, sortStage)

	for _, card := range cards {
		LogMessage(fmt.Sprintf("* %dx %s%s%s (%s/%s) %s%.2f EUR%s", card.SerraCount, Purple, card.Name, Reset, card.Set, card.CollectorNumber, Yellow, card.Prices.Eur, Reset), "normal")
		total = total + card.Prices.Eur*float64(card.SerraCount)
	}
	fmt.Printf("\nTotal Value: %s%.2f EUR%s\n", Yellow, total, Reset)

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

func Missing(setname string) error {

	client := storage_connect()
	coll := &Collection{client.Database("serra").Collection("cards")}
	defer storage_disconnect(client)

	// fetch all cards in set
	cards, err := coll.storage_find(bson.D{{"set", setname}}, bson.D{{"collectornumber", 1}})
	if (err != nil) || len(cards) == 0 {
		LogMessage(fmt.Sprintf("Error: Set %s not found or no card in your collection.", setname), "red")
		return err
	}

	// fetch set informations
	setcoll := &Collection{client.Database("serra").Collection("sets")}
	sets, _ := setcoll.storage_find_set(bson.D{{"code", setname}}, bson.D{{"_id", 1}})
	set := sets[0]

	LogMessage(fmt.Sprintf("Missing cards in %s", sets[0].Name), "green")

	// generate set with all setnumbers
	var complete_set []string
	var i int64
	for i = 1; i <= set.CardCount; i++ {
		complete_set = append(complete_set, strconv.FormatInt(i, 10))
	}

	// iterate over all cards in collection
	var in_collection []string
	for _, c := range cards {
		in_collection = append(in_collection, c.CollectorNumber)
	}

	misses := missing(in_collection, complete_set)
	for _, m := range misses {
		ncard, err := fetch_card(fmt.Sprintf("%s/%s", setname, m))
		if err != nil {
			continue
		}
		fmt.Printf("%s (%s)\n", ncard.Name, ncard.SetName)
	}
	return nil
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

	// this is maybe the ugliest way someone could choose to verify, if a rarity type is missing
	// [
	// { _id: { rarity: 'common' }, count: 20 },
	// { _id: { rarity: 'uncommon' }, count: 2 }
	// ]
	// if a result like this is there, 1 rarity type "rare" is not in the array. and needs to be
	// initialized with 0, otherwise we get a panic
	var rares, uncommons, commons float64
	for _, r := range rar {
		switch r["_id"] {
		case "rare":
			rares = r["count"].(float64)
		case "uncommon":
			uncommons = r["count"].(float64)
		case "common":
			commons = r["count"].(float64)
		}
	}

	LogMessage(fmt.Sprintf("%s", sets[0].Name), "green")
	LogMessage(fmt.Sprintf("Set Cards: %d/%d", len(cards), sets[0].CardCount), "normal")
	LogMessage(fmt.Sprintf("Total Cards: %.0f", stats[0]["count"]), "normal")
	LogMessage(fmt.Sprintf("Total Value: %.2f EUR", stats[0]["value"]), "normal")
	LogMessage(fmt.Sprintf("Released: %s", sets[0].ReleasedAt), "normal")
	LogMessage(fmt.Sprintf("Rares: %.0f", rares), "normal")
	LogMessage(fmt.Sprintf("Uncommons: %.0f", uncommons), "normal")
	LogMessage(fmt.Sprintf("Commons: %.0f", commons), "normal")
	fmt.Printf("\n%sPrice History:%s\n", Pink, Reset)

	var before float64
	for _, e := range sets[0].SerraPrices {
		if e.Value > before {
			fmt.Printf("* %s %s%.2f EUR%s\n", stringToTime(e.Date), Green, e.Value, Reset)
		} else if e.Value < before {
			fmt.Printf("* %s %s%.2f EUR%s\n", stringToTime(e.Date), Red, e.Value, Reset)
		} else {
			fmt.Printf("* %s %.2f EUR%s\n", stringToTime(e.Date), e.Value)
		}
		before = e.Value
	}

	fmt.Printf("\n%sMost valuable cards%s\n", Pink, Reset)
	ccards := 0
	if len(cards) < 10 {
		ccards = len(cards)

	} else {
		ccards = 10
	}

	for i := 0; i < ccards; i++ {
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
	totalcoll := &Collection{client.Database("serra").Collection("total")}

	sets, _ := fetch_sets()
	for _, set := range sets.Data {
		setscoll.storage_add_set(&set)
		cards, _ := coll.storage_find(bson.D{{"set", set.Code}}, bson.D{{"_id", 1}})

		// if no cards in collection for this set, skip it
		if len(cards) == 0 {
			continue
		}

		bar := progressbar.NewOptions(len(cards),
			progressbar.OptionSetWidth(80),
			progressbar.OptionSetDescription(fmt.Sprintf("%s%s%s (%s%s%s, %s)", Pink, set.Name, Reset, Yellow, set.Code, Reset, set.ReleasedAt[0:4])),
		)
		for _, card := range cards {
			bar.Add(1)
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
		fmt.Println()

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
		// fmt.Printf("Set %s%s%s (%s) is now worth %s%.02f EUR%s\n", Pink, set.Name, Reset, set.Code, Yellow, setvalue[0]["value"], Reset)
		setscoll.storage_update(bson.M{"code": bson.M{"$eq": set.Code}}, set_update)
	}

	// calculate total summary over all sets
	overall_value := mongo.Pipeline{
		bson.D{{"$match",
			bson.D{{"serra_prices", bson.D{{"$exists", true}}}}}},
		bson.D{{"$project",
			bson.D{{"name", true}, {"totalValue", bson.D{{"$arrayElemAt", bson.A{"$serra_prices", -1}}}}}}},
		bson.D{{"$group", bson.D{{"_id", nil}, {"total", bson.D{{"$sum", "$totalValue.value"}}}}}},
	}
	ostats, _ := setscoll.storage_aggregate(overall_value)
	fmt.Printf("\n%sUpdating total value of collection to: %s%.02f EUR%s\n", Green, Yellow, ostats[0]["total"].(float64), Reset)
	totalcoll.storage_add_total(ostats[0]["total"].(float64))

	return nil
}

func Gains(limit float64, sort int) error {

	client := storage_connect()
	coll := &Collection{client.Database("serra").Collection("cards")}
	defer storage_disconnect(client)

	// db.cards.aggregate({$project: {set: 1, collectornumber:1, name: 1, "old": {$arrayElemAt: ["$serra_prices.value", -2]}, "current": {$arrayElemAt: ["$serra_prices.value", -1]} }}, {$match: {old: {$gt: 2}}} ,{$project: {name: 1,set:1,collectornumber:1,current:1, "rate": {$subtract: [{$divide: ["$current", {$divide: ["$old", 100]}]}, 100]} }}, {$sort: { rate: -1}})
	raise_pipeline := mongo.Pipeline{
		bson.D{{"$project",
			bson.D{
				{"name", true},
				{"set", true},
				{"collectornumber", true},
				{"old",
					bson.D{{"$arrayElemAt",
						bson.A{"$serra_prices.value", 0},
					}},
				},
				{"current",
					bson.D{{"$arrayElemAt",
						bson.A{"$serra_prices.value", -1},
					}},
				},
			},
		}},
		bson.D{{"$match",
			bson.D{{"old", bson.D{{"$gt", limit}}}},
		}},
		bson.D{{"$project",
			bson.D{
				{"name", true},
				{"set", true},
				{"old", true},
				{"current", true},
				{"collectornumber", true},
				{"rate",
					bson.D{{"$subtract",
						bson.A{
							bson.D{{"$divide",
								bson.A{"$current",
									bson.D{{"$divide",
										bson.A{"$old", 100},
									}},
								},
							}},
							100,
						},
					}},
				},
			},
		}},
		bson.D{{"$sort",
			bson.D{{"rate", sort}}}},
		bson.D{{"$limit", 20}},
	}
	raise, _ := coll.storage_aggregate(raise_pipeline)

	// percentage coloring
	var p_color string
	if sort == 1 {
		p_color = Red
	} else {
		p_color = Green
	}

	// print each card
	for _, e := range raise {
		fmt.Printf("%s%+.0f%%%s %s %s(%s/%s)%s (%.2f->%s%.2f EUR%s) \n", p_color, e["rate"], Reset, e["name"], Yellow, e["set"], e["collectornumber"], Reset, e["old"], Green, e["current"], Reset)
	}
	return nil

}

func Stats() {

	client := storage_connect()
	coll := &Collection{client.Database("serra").Collection("cards")}
	totalcoll := &Collection{client.Database("serra").Collection("total")}
	defer storage_disconnect(client)

	// LogMessage(fmt.Sprintf("Color distribution in Collection"), "green")
	// groupStage := bson.D{
	// 	{"$group", bson.D{
	// 		{"_id", "$coloridentity"},
	// 		{"count", bson.D{{"$sum", bson.D{{"$multiply", bson.A{1.0, "$serra_count"}}}}}},
	// 	}}}

	sortStage := bson.D{
		{"$sort", bson.D{
			{"count", -1},
		}}}
	// sets, _ := coll.storage_aggregate(mongo.Pipeline{groupStage, sortStage})
	// for _, set := range sets {
	// 	x, _ := set["_id"].(primitive.A)
	// 	s := []interface{}(x)
	// 	fmt.Printf("* %s %.0f\n", convert_mana_symbols(s), set["count"])
	// }

	statsGroup := bson.D{
		{"$group", bson.D{
			{"_id", nil},
			{"value", bson.D{{"$sum", bson.D{{"$multiply", bson.A{"$prices.eur", "$serra_count"}}}}}},
			{"count", bson.D{{"$sum", bson.D{{"$multiply", bson.A{1.0, "$serra_count"}}}}}},
			{"rarity", bson.D{{"$sum", "$rarity"}}},
			{"unique", bson.D{{"$sum", 1}}},
		}},
	}
	stats, _ := coll.storage_aggregate(mongo.Pipeline{statsGroup})

	rarityStage := bson.D{
		{"$group", bson.D{
			{"_id", "$rarity"},
			{"count", bson.D{{"$sum", bson.D{{"$multiply", bson.A{1.0, "$serra_count"}}}}}},
		}}}

	sortStage = bson.D{
		{"$sort", bson.D{
			{"_id", 1},
		}}}
	rar, _ := coll.storage_aggregate(mongo.Pipeline{rarityStage, sortStage})

	// this is maybe the ugliest way someone could choose to verify, if a rarity type is missing
	// [
	// { _id: { rarity: 'common' }, count: 20 },
	// { _id: { rarity: 'uncommon' }, count: 2 }
	// ]
	// if a result like this is there, 1 rarity type "rare" is not in the array. and needs to be
	// initialized with 0, otherwise we get a panic
	var rares, uncommons, commons float64
	for _, r := range rar {
		switch r["_id"] {
		case "rare":
			rares = r["count"].(float64)
		case "uncommon":
			uncommons = r["count"].(float64)
		case "common":
			commons = r["count"].(float64)
		}
	}

	fmt.Printf("%sCards %s\n", Green, Reset)
	fmt.Printf("Total Cards: %s%.0f%s\n", Yellow, stats[0]["count"], Reset)
	fmt.Printf("Unique Cards: %s%d%s\n", Purple, stats[0]["unique"], Reset)

	fmt.Printf("\n%sRarity%s\n", Green, Reset)
	fmt.Printf("Rares: %s%.0f%s\n", Pink, rares, Reset)
	fmt.Printf("Uncommons: %s%.0f%s\n", Yellow, uncommons, Reset)
	fmt.Printf("Commons: %s%.0f%s\n", Purple, commons, Reset)

	fmt.Printf("\n%sTotal Value%s\n", Green, Reset)
	fmt.Printf("Current: %s%.2f%s\n", Pink, stats[0]["value"], Reset)
	total, _ := totalcoll.storage_find_total()

	var before float64
	fmt.Printf("History: \n")
	for _, e := range total.Value {
		if e.Value > before {
			fmt.Printf("* %s %s%.2f EUR%s\n", stringToTime(e.Date), Green, e.Value, Reset)
		} else if e.Value < before {
			fmt.Printf("* %s %s%.2f EUR%s\n", stringToTime(e.Date), Red, e.Value, Reset)
		} else {
			fmt.Printf("* %s %.2f EUR\n", stringToTime(e.Date), e.Value)
		}
		before = e.Value
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
