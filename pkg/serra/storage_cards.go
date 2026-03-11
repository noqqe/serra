package serra

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Card struct {
	// Added by Serra
	SerraCount       int64              `bson:"serra_count"`
	SerraCountFoil   int64              `bson:"serra_count_foil"`
	SerraCountEtched int64              `bson:"serra_count_etched"`
	SerraPriceList   []PriceEntry       `bson:"serra_prices"`
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
	Keywords   []any  `json:"keywords"`
	Lang       string `json:"lang"`
	Layout     string `json:"layout"`
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
	ManaCost        string     `json:"mana_cost"`
	MultiverseIds   []any      `json:"multiverse_ids"`
	Name            string     `json:"name"`
	Nonfoil         bool       `json:"nonfoil"`
	Object          string     `json:"object"`
	OracleID        string     `json:"oracle_id"`
	OracleText      string     `json:"oracle_text"`
	Oversized       bool       `json:"oversized"`
	Prices          PriceEntry `json:"prices"`
	PrintedName     string     `json:"printed_name"`
	PrintedText     string     `json:"printed_text"`
	PrintedTypeLine string     `json:"printed_type_line"`
	PrintsSearchURI string     `json:"prints_search_uri"`
	Promo           bool       `json:"promo"`
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

type CardsCollection struct {
	*mongo.Collection
}

func (client StorageClient) getCardsCollection() CardsCollection {
	return CardsCollection{client.Database("serra").Collection("cards")}
}

func (coll CardsCollection) storageAdd(card *Card) error {

	card.SerraUpdated = primitive.NewDateTimeFromTime(time.Now())

	_, err := coll.InsertOne(context.TODO(), card)
	if err != nil {
		return err
	}
	return nil

}

// FindCards returns a list of cards by a given filter, sort and pagination options.
func (coll CardsCollection) FindCards(filter, sort bson.D, skip, limit int64) ([]Card, error) {
	opts := options.Find().SetSort(sort).SetSkip(skip).SetLimit(limit)
	cursor, err := coll.Find(context.TODO(), filter, opts)
	l := Logger()

	if err != nil {
		l.Fatalf("Could not query data due to connection errors to database: %s", err.Error())
	}

	var results []Card
	if err = cursor.All(context.TODO(), &results); err != nil {
		l.Fatal(err)
		return []Card{}, err
	}
	return results, nil

}

// FindCardByCollectorNumber returns a card by set code and collector number.
// If no card is found, an error is returned.
func (coll CardsCollection) FindCardByCollectorNumber(setCode string, collectorNumber string) (*Card, error) {
	sort := bson.D{{"_id", 1}}
	searchFilter := bson.D{{"set", setCode}, {"collectornumber", collectorNumber}}
	cards, err := coll.FindCards(searchFilter, sort, 0, 0)
	if err != nil {
		return &Card{}, err
	}

	if len(cards) < 1 {
		return &Card{}, errors.New("Card not found")
	}

	return &cards[0], nil
}

// RemoveCards removes cards from the collection by a given filter. If no card
// is found, an error is returned.
func (coll CardsCollection) RemoveCards(filter bson.M) error {
	l := Logger()

	_, err := coll.DeleteOne(context.TODO(), filter)
	if err != nil {
		l.Fatalf("Could remove card data due to connection errors to database: %s", err.Error())
	}

	return nil
}

// AggregateCards aggregates cards in the collection by a given pipeline.
func (coll CardsCollection) AggregateCards(pipeline mongo.Pipeline) ([]primitive.M, error) {
	l := Logger()
	opts := options.Aggregate()

	cursor, err := coll.Aggregate(
		context.TODO(),
		pipeline,
		opts)
	if err != nil {
		l.Fatalf("Could not aggregate data due to connection errors to database: %s", err.Error())
		return []primitive.M{}, err
	}

	// Get a list of all returned documents and print them out.
	// See the mongo.Cursor documentation for more examples of using cursors.
	var results []bson.M
	if err = cursor.All(context.TODO(), &results); err != nil {
		l.Fatal(err)
		return []primitive.M{}, err
	}

	return results, nil
}

// UpdateCards updates cards in the collection by a given filter and update statement.
func (coll CardsCollection) UpdateCards(filter, update bson.M) error {
	l := Logger()
	// Call the driver's UpdateOne() method and pass filter and update to it
	_, err := coll.UpdateOne(
		context.Background(),
		filter,
		update,
	)
	if err != nil {
		l.Fatalf("Could not update data due to connection errors to database: %s", err.Error())
	}

	return nil
}

// ModifyCardCount modifies the amount of a card in the collection by a given
// amount in foil or nonfoil
func (coll CardsCollection) ModifyCardCount(c *Card, amount int64, foil bool) error {

	l := Logger()
	storedCard, err := coll.FindCardByCollectorNumber(c.Set, c.CollectorNumber)
	if err != nil {
		return err
	}

	// update card amount
	var update bson.M
	if foil {
		update = bson.M{
			"$set": bson.M{"serra_count_foil": storedCard.SerraCountFoil + amount},
		}
	} else {
		update = bson.M{
			"$set": bson.M{"serra_count": storedCard.SerraCount + amount},
		}
	}

	coll.UpdateCards(bson.M{"_id": bson.M{"$eq": c.ID}}, update)

	var total int64
	if foil {
		total = storedCard.SerraCountFoil + amount
		if amount < 0 {
			l.Warnf("Reduced card amount of \"%s\" (%.2f%s, foil) from %d to %d", storedCard.Name, storedCard.getFoilValue(), getCurrency(), storedCard.SerraCountFoil, total)
		} else {
			l.Warnf("Increased card amount of \"%s\" (%.2f%s, foil) from %d to %d", storedCard.Name, storedCard.getFoilValue(), getCurrency(), storedCard.SerraCountFoil, total)
		}
	} else {
		total = storedCard.SerraCount + amount
		if amount < 0 {
			l.Warnf("Reduced card amount of \"%s\" (%.2f%s) from %d to %d", storedCard.Name, storedCard.getValue(), getCurrency(), storedCard.SerraCount, total)
		} else {
			l.Warnf("Increased card amount of \"%s\" (%.2f%s) from %d to %d", storedCard.Name, storedCard.getValue(), getCurrency(), storedCard.SerraCount, total)
		}
	}

	return nil
}
