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

type CardList struct {
	Data []Set `json:"data"`
}

// Card holds pure Scryfall data for a single printing. It contains no
// ownership information (count, condition, language, ...) - that data
// lives in the "inventory" collection instead, keyed by CardID.
type Card struct {
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

	// PriceHistory holds a snapshot of Prices for every update the card has
	// gone through, regardless of whether it is owned. It is never present
	// in Scryfall's own data and is maintained solely by UpsertCard(s).
	PriceHistory []PriceEntry `json:"-" bson:"price_history,omitempty"`
}

type CardsCollection struct {
	*mongo.Collection
}

func (client StorageClient) getCardsCollection() CardsCollection {
	return CardsCollection{client.Database("serra").Collection("cards")}
}

// UpsertCard inserts a card or replaces it in place if it already exists
// (keyed by its Scryfall ID), appending a snapshot of its current prices to
// its price history. Used to (re-)cache Scryfall data.
func (coll CardsCollection) UpsertCard(card *Card) error {
	l := Logger()
	now := primitive.NewDateTimeFromTime(time.Now())

	pipeline, err := cardUpsertPipeline(card, now)
	if err != nil {
		l.Fatalf("Could not build update pipeline for card data: %s", err.Error())
		return err
	}

	_, err = coll.UpdateOne(context.TODO(), bson.M{"_id": card.ID}, pipeline, options.Update().SetUpsert(true))
	if err != nil {
		l.Fatalf("Could not store card data due to connection errors to database: %s", err.Error())
	}
	return err
}

// UpsertCards upserts a whole batch of cards in one round trip, appending a
// price snapshot to each card's price history. Used to import the full
// Scryfall bulk file so that prices can be tracked for every printing,
// regardless of ownership.
func (coll CardsCollection) UpsertCards(cards []Card) error {
	l := Logger()
	now := primitive.NewDateTimeFromTime(time.Now())

	const batchSize = 500
	for start := 0; start < len(cards); start += batchSize {
		end := start + batchSize
		if end > len(cards) {
			end = len(cards)
		}

		models := make([]mongo.WriteModel, 0, end-start)
		for i := range cards[start:end] {
			card := &cards[start+i]
			pipeline, err := cardUpsertPipeline(card, now)
			if err != nil {
				l.Error("Could not build update pipeline for card, skipping:", err)
				continue
			}
			models = append(models, mongo.NewUpdateOneModel().
				SetFilter(bson.M{"_id": card.ID}).
				SetUpdate(pipeline).
				SetUpsert(true))
		}

		if len(models) == 0 {
			continue
		}

		if _, err := coll.BulkWrite(context.TODO(), models, options.BulkWrite().SetOrdered(false)); err != nil {
			l.Fatalf("Could not bulk store card data due to connection errors to database: %s", err.Error())
			return err
		}
	}
	return nil
}

// cardUpsertPipeline builds an aggregation-pipeline update that replaces a
// card document's Scryfall data wholesale while appending (rather than
// clobbering) its price_history array. A plain $set/$replace can't do this
// in one step since the incoming Card value doesn't carry the existing
// history with it.
func cardUpsertPipeline(card *Card, snapshotDate primitive.DateTime) (mongo.Pipeline, error) {
	cardDoc, err := toBsonM(card)
	if err != nil {
		return nil, err
	}
	delete(cardDoc, "price_history")

	snapshot := card.Prices
	snapshot.Date = snapshotDate
	snapshotDoc, err := toBsonM(snapshot)
	if err != nil {
		return nil, err
	}

	return mongo.Pipeline{
		bson.D{{"$replaceWith", bson.D{{"$mergeObjects", bson.A{
			"$$ROOT",
			cardDoc,
			bson.D{{"price_history", bson.D{{"$concatArrays", bson.A{
				bson.D{{"$ifNull", bson.A{"$price_history", bson.A{}}}},
				bson.A{snapshotDoc},
			}}}}}},
		}}}},
	}, nil
}

// toBsonM marshals a value to its bson.M representation, following its bson
// struct tags exactly as a driver Insert/Replace would.
func toBsonM(v any) (bson.M, error) {
	raw, err := bson.Marshal(v)
	if err != nil {
		return nil, err
	}
	var m bson.M
	if err := bson.Unmarshal(raw, &m); err != nil {
		return nil, err
	}
	return m, nil
}

// FindCards returns a list of cards by a given filter, sort and pagination options.
func (coll CardsCollection) FindCards(filter, sort bson.D, skip, limit int64) ([]Card, error) {
	l := Logger()
	opts := options.Find().SetSort(sort).SetSkip(skip).SetLimit(limit)

	cursor, err := coll.Find(context.TODO(), filter, opts)
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

// FindCardByCollectorNumber returns cached Scryfall data for a card by set
// code and collector number. This is a pure Scryfall cache lookup and makes
// no statement about whether the card is owned.
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

// FindCardByID returns cached Scryfall data for a card by its Scryfall ID.
func (coll CardsCollection) FindCardByID(id string) (*Card, error) {
	var card Card
	err := coll.FindOne(context.TODO(), bson.M{"_id": id}).Decode(&card)
	if err != nil {
		return &Card{}, err
	}
	return &card, nil
}

// FindCardsByIDs returns cached Scryfall data for a list of Scryfall IDs, as
// a map keyed by ID for convenient joining with inventory data.
func (coll CardsCollection) FindCardsByIDs(ids []string) (map[string]Card, error) {
	cards, err := coll.FindCards(bson.D{{"_id", bson.D{{"$in", ids}}}}, bson.D{}, 0, 0)
	if err != nil {
		return nil, err
	}

	result := make(map[string]Card, len(cards))
	for _, c := range cards {
		result[c.ID] = c
	}
	return result, nil
}
