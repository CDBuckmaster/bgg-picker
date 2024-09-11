package main

import (
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const endpoint = "https://boardgamegeek.com/xmlapi2/collection?subtype=boardgame&own=1&stats=1&username="
const username = ""
const maxRequestAttempts = 5
const playerCount = 4
const weight = "light"
const playTime = "short"

type ValueRange struct {
	Min int
	Max int
}

func (v ValueRange) IsInBetween(value int) bool {
	return value >= v.Min && value <= v.Max
}

var weights = map[string]ValueRange{
	"light":  ValueRange{1, 2},
	"medium": ValueRange{2, 3},
	"heavy":  ValueRange{3, 5},
}
var playTimes = map[string]ValueRange{
	"short":  ValueRange{0, 30},
	"medium": ValueRange{30, 90},
	"long":   ValueRange{90, 1000},
}

type Items struct {
	XMLName    xml.Name `xml:"items"`
	TotalItems int      `xml:"totalitems,attr"`
	TermsOfUse string   `xml:"termsofuse,attr"`
	PubDate    string   `xml:"pubdate,attr"`
	ItemList   []Item   `xml:"item"`
}

type Item struct {
	XMLName       xml.Name `xml:"item"`
	ObjectType    string   `xml:"objecttype,attr"`
	ObjectID      int      `xml:"objectid,attr"`
	Subtype       string   `xml:"subtype,attr"`
	CollID        int      `xml:"collid,attr"`
	Name          Name     `xml:"name"`
	YearPublished int      `xml:"yearpublished"`
	Image         string   `xml:"image"`
	Thumbnail     string   `xml:"thumbnail"`
	Stats         Stats    `xml:"stats"`
	Status        Status   `xml:"status"`
	NumPlays      int      `xml:"numplays"`
}

type Name struct {
	SortIndex int    `xml:"sortindex,attr"`
	Value     string `xml:",chardata"`
}

type Stats struct {
	MinPlayers  int    `xml:"minplayers,attr"`
	MaxPlayers  int    `xml:"maxplayers,attr"`
	MinPlayTime int    `xml:"minplaytime,attr"`
	MaxPlayTime int    `xml:"maxplaytime,attr"`
	PlayingTime int    `xml:"playingtime,attr"`
	NumOwned    int    `xml:"numowned,attr"`
	Rating      Rating `xml:"rating"`
}

type Rating struct {
	Value        string `xml:"value,attr"`
	UsersRated   string `xml:"usersrated"`
	Average      string `xml:"average"`
	BayesAverage string `xml:"bayesaverage"`
	StdDev       string `xml:"stddev"`
	Median       string `xml:"median"`
	Ranks        []Rank `xml:"ranks>rank"`
}

type Value struct {
	Value float64 `xml:"value,attr"`
}

type Rank struct {
	Type         string `xml:"type,attr"`
	ID           int    `xml:"id,attr"`
	Name         string `xml:"name,attr"`
	FriendlyName string `xml:"friendlyname,attr"`
	Value        string `xml:"value,attr"`
	BayesAverage string `xml:"bayesaverage,attr"`
}

type Status struct {
	Own          int    `xml:"own,attr"`
	PrevOwned    int    `xml:"prevowned,attr"`
	ForTrade     int    `xml:"fortrade,attr"`
	Want         int    `xml:"want,attr"`
	WantToPlay   int    `xml:"wanttoplay,attr"`
	WantToBuy    int    `xml:"wanttobuy,attr"`
	Wishlist     int    `xml:"wishlist,attr"`
	Preordered   int    `xml:"preordered,attr"`
	LastModified string `xml:"lastmodified,attr"`
}

func getCollection() (string, error) {
	url := endpoint + username
	reader := strings.NewReader(``)
	request, err := http.NewRequest("GET", url, reader)
	if err != nil {
		fmt.Println(err)
		return "", err
	}

	client := &http.Client{}
	var responseCode string
	var bodyString string
	counter := 0

	for responseCode != "200 OK" && counter < maxRequestAttempts {
		counter++
		resp, err := client.Do(request)
		if err != nil {
			fmt.Println(err)
			return "", err
		}
		responseCode = resp.Status
		if responseCode != "200 OK" {
			continue
		}
		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			fmt.Println(err)
			return "", err
		}

		bodyString = string(bodyBytes)
		resp.Body.Close()
	}
	return bodyString, nil
}

func parseCollection(collection string) (Items, error) {
	var items Items
	err := xml.Unmarshal([]byte(collection), &items)
	if err != nil {
		fmt.Println(err)
		return items, err
	}
	return items, nil
}

func filterCollection(collection Items) Items {
	var filteredCollection Items
	for _, item := range collection.ItemList {
		if item.Stats.MinPlayers <= playerCount &&
			item.Stats.MaxPlayers >= playerCount &&
			playTimes[playTime].IsInBetween(item.Stats.PlayingTime) {
			//weights[weight].IsInBetween(int(item.Rating.Average.Value)) {
			filteredCollection.ItemList = append(filteredCollection.ItemList, item)
		}
	}
	return filteredCollection
}

func main() {
	collectionString, _ := getCollection()
	collection, _ := parseCollection(collectionString)
	filteredCollection := filterCollection(collection)
	for _, item := range filteredCollection.ItemList {
		fmt.Println(item.Name.Value)
	}
}
