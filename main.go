package main

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	ginadapter "github.com/awslabs/aws-lambda-go-api-proxy/gin"
	"github.com/gin-gonic/gin"
)

const endpoint = "https://boardgamegeek.com/xmlapi2/collection?subtype=boardgame&own=1&stats=1&username="
const maxRequestAttempts = 5

// Didn't write this myself, simple array map function
func Map[T, U any](ts []T, f func(T) U) []U {
	us := make([]U, len(ts))
	for i := range ts {
		us[i] = f(ts[i])
	}
	return us
}

// Simple struct for any value that represents a range
type ValueRange struct {
	Min int
	Max int
}

func (v ValueRange) IsInBetween(value int) bool {
	return value >= v.Min && value <= v.Max
}

// Map of available play time values
var playTimes = map[string]ValueRange{
	"short":  ValueRange{0, 30},
	"medium": ValueRange{30, 90},
	"long":   ValueRange{90, 1000},
}

// BGG XML Entities
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

// Retrieve raw response from BGG
func getCollection(username string) (string, error) {
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

	// BGG has a weird queue system where they encourage you to keep making requests until it works
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

// Convert XML data into Go structs
func parseCollection(collection string) (Items, error) {
	var items Items
	err := xml.Unmarshal([]byte(collection), &items)
	if err != nil {
		fmt.Println(err)
		return items, err
	}
	return items, nil
}

// Filter items by user's parameters
func filterCollection(collection Items, playerCount int, playTime string) Items {
	var filteredCollection Items
	for _, item := range collection.ItemList {
		if item.Stats.MinPlayers <= playerCount &&
			item.Stats.MaxPlayers >= playerCount &&
			playTimes[playTime].IsInBetween(item.Stats.PlayingTime) {
			filteredCollection.ItemList = append(filteredCollection.ItemList, item)
		}
	}
	return filteredCollection
}

// @TODO retrieving and parsing collection should be done in a repo
// Retrieves collection, parses it and then filters based on user params
func pickGames(username string, playerCount int, playTime string) []Item {
	collectionString, _ := getCollection(username)
	collection, _ := parseCollection(collectionString)
	filteredCollection := filterCollection(collection, playerCount, playTime)
	return filteredCollection.ItemList
}

// Handles request and performs basic validation
func handleRequest(c *gin.Context) {
	username := c.Query("username")
	playerCountString := c.Query("playerCount")
	playTime := c.Query("playTime")
	if username == "" || playerCountString == "" || playTime == "" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Missing parameters"})
		return
	}
	playerCount, err := strconv.Atoi(playerCountString)

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid player count"})
		return
	}
	games := pickGames(username, playerCount, playTime)
	c.JSON(http.StatusOK, Map(games, func(item Item) string {
		return item.Name.Value
	}))
}

// AWS gin lambda adapter
var ginLambda *ginadapter.GinLambda

// AWS Lambda Proxy Handler
// This handler acts like a bridge between AWS Lambda and our Local GIn server
// It maps each GIN route to a Lambda function as handler
//
// This is useful to make our function execution possible.
func GinRequestHandler(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	return ginLambda.ProxyWithContext(ctx, request)
}

func main() {

	//Set the router as the default one provided by Gin
	router := gin.Default()

	//Define our routes
	router.GET("/", handleRequest)

	// Check whether port is provided, assume is lambda deployment if not
	httpPort := os.Getenv("HTTP_PORT")
	if httpPort == "" {
		log.Println("Starting Lambda Handler")
		ginLambda = ginadapter.New(router)
		lambda.Start(GinRequestHandler)
	} else {
		log.Printf("Starting HTTP server on port %s\n", httpPort)
		formattedPort := fmt.Sprintf("localhost:%s", httpPort)
		router.Run(formattedPort)
	}
}
