package main

import (
	"context"
	"encoding/json"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"html/template"
	"log"
	"net/http"
	"slices"
	"strconv"
	"time"
)

type Alert struct {
	Name      string    `bson:"name" json:"name"`
	Priority  string    `bson:"priority" json:"priority"`
	CreatedAt time.Time `bson:"createdAt" json:"createdAt"`
	Cleared   bool      `bson:"cleared" json:"cleared"`
}

func main() {
	client, err := mongo.Connect(context.TODO(), options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		log.Fatal(err)
	}
	defer client.Disconnect(context.TODO())
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		dashboardHandler(w, r, client)
	})
	http.HandleFunc("/loadMoreAlerts", func(w http.ResponseWriter, r *http.Request) {
		loadMoreAlertsHandler(w, r, client)
	})
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	log.Println("Server is running on http://localhost:8080")
	http.ListenAndServe(":8080", nil)
}

/****
This version of getRecentAlerts uses the _alerts_ collection to sort
and limit documents, then retrieves all documents, decoding them into
array.
****

func getRecentAlerts(client *mongo.Client) ([]Alert, time.Duration) {
	ctx := context.TODO()

	startTime := time.Now()
	collection := client.Database("alertdb").Collection("alerts")

	sort := bson.D{{"createdAt", 1}}

	opts := options.Find()
	opts.SetLimit(25)
	opts.SetSort(sort)

	cursor, err := collection.Find(ctx, bson.D{{"cleared", false}}, opts)

	if err != nil {
		log.Println("Error fetching recent alerts:", err)
		return nil, 0
	}
	defer cursor.Close(ctx)

	var alerts []Alert
	if err = cursor.All(ctx, &alerts); err != nil {
		log.Println("Error decoding recent alerts:", err)
		return nil, 0
	}

	return alerts, time.Since(startTime)
}
*/

/*
***
The final getRecentAlerts function contains a single bucket of the 25 most
recent alerts. These alerts are stored in a document in the _dashboard_
collection with all 25 alerts stored in the `values` field.
*/
func getRecentAlerts(client *mongo.Client) ([]Alert, time.Duration) {
	startTime := time.Now()
	collection := client.Database("alertdb").Collection("dashboard")

	var result struct {
		ID     string  `bson:"_id,omitempty"`
		Values []Alert `bson:"values"`
	}

	filter := bson.D{{"_id", "top25"}}
	err := collection.FindOne(context.TODO(), filter).Decode(&result)

	if err != nil {
		if err == mongo.ErrNoDocuments {
			return []Alert{}, 0
		} else {
			log.Println("Error fetching recent alerts:", err)
			return nil, 0
		}
	}

	return result.Values, time.Since(startTime)
}

/****
This version of getPriorityAlerts uses the _alerts_ collection to sort
and limit documents, then retrieves all documents, decoding them into
array.
****

func getPriorityAlerts(client *mongo.Client) ([]Alert, time.Duration, int, time.Duration) {
	startTime := time.Now()
	collection := client.Database("alertdb").Collection("alerts")

	filter := bson.D{
		{"priority", bson.D{{"$in", bson.A{"Critical", "High"}}}},
		{"cleared", false},
	}

	// Measure the time taken to count the total number of critical and high priority alerts
	countStartTime := time.Now()
	count, err := collection.CountDocuments(context.TODO(), filter)

	countQueryTime := time.Since(countStartTime)
	if err != nil {
		log.Println("Error counting priority alerts:", err)
		return nil, 0, 0, 0
	}

	// Retrieve critical and high priority alerts
	cursor, err := collection.Find(context.TODO(), filter, options.Find().SetSort(bson.D{{"createdAt", -1}}))
	if err != nil {
		log.Println("Error fetching priority alerts:", err)
		return nil, 0, 0, 0
	}

	defer cursor.Close(context.TODO())

	var alerts []Alert
	if err = cursor.All(context.TODO(), &alerts); err != nil {
		log.Println("Error decoding priority alerts:", err)
		return nil, 0, 0, 0
	}

	return alerts, time.Since(startTime), int(count), countQueryTime
}
*/

/****
This version of getPriorityAlerts uses a single bucket that contains all
"Critical" and "High" priorities. The bucket is populated via the aggregation
in the file "most-recent-aggreation-all.js" and exists in a document with the
_id equal to "priority".
****

func getPriorityAlerts(client *mongo.Client) ([]Alert, time.Duration, int, time.Duration) {
	startTime := time.Now()
	collection := client.Database("alertdb").Collection("dashboard")

	filter := bson.D{
		{"_id", "priority"},
	}

	var result struct {
		ID     string  `bson:"_id,omitempty"`
		Count  int     `bson:"count"`
		Values []Alert `bson:"values"`
	}

	// Retrieve the "Critical" and "High" priority dashboard documents.
	err := collection.FindOne(context.TODO(), filter).Decode(&result)
	if err != nil {
		log.Println("Error fetching priority alerts:", err)
		return nil, 0, 0, 0
	}

	return result.Values, time.Since(startTime), result.Count, 0
}
*/

/*
***
The final version of getPriorityAlerts uses buckets of a pre-determined size
(set by the number of objects stored in the _values_ array) and returns an
array of alerts, and the number of items in the array directly from a document.

Each document is represented by _id containing the string "bucket_" with
a number representing a monotonically increasing integer for subsequent buckets.
For example, the first bucket of alerts has _id of "bucket_0" and is loaded
during the initial page load. Then, asynchronous calls get each subsequent bucket
until no additional buckets are available.
***
*/
func getPriorityAlerts(client *mongo.Client, bucketIndex int) ([]Alert, time.Duration, int) {
	startTime := time.Now()

	collection := client.Database("alertdb").Collection("dashboard")
	bucketID := fmt.Sprintf("priority_bucket_%d", bucketIndex)

	var result struct {
		ID     string  `bson:"_id,omitempty"`
		Count  int     `bson:"count"`
		Values []Alert `bson:"values"`
	}

	err := collection.FindOne(context.TODO(), bson.D{{"_id", bucketID}}).Decode(&result)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return []Alert{}, time.Since(startTime), 0
		}
		log.Println("Error fetching priority alerts:", err)
		return nil, time.Since(startTime), 0
	}

	return result.Values, time.Since(startTime), result.Count
}

func getPriorityAlertsCount(client *mongo.Client) (int, time.Duration) {
	startTime := time.Now()

	collection := client.Database("alertdb").Collection("dashboard")

	query := bson.D{{"_id", primitive.Regex{"^priority_bucket_", ""}}}
	count, err := collection.CountDocuments(context.TODO(), query)

	if err != nil {
		log.Println("Error fetching priority alerts count:", err)
		return 0, time.Since(startTime)
	}

	return int(count), time.Since(startTime)
}

func dashboardHandler(w http.ResponseWriter, _ *http.Request, client *mongo.Client) {
	startTime := time.Now()

	// Retrieve the top 25 most recent alerts
	recentAlerts, recentQueryTime := getRecentAlerts(client)

	// Retrieve the first bucket of priority alerts
	priorityAlertsCount, priorityCountTime := getPriorityAlertsCount(client)
	priorityAlerts, priorityQueryTime, priorityCount := getPriorityAlerts(client, priorityAlertsCount-1)

	// Switch the display order from ascending to descending.
	slices.Reverse(priorityAlerts)

	pageLoadTime := time.Since(startTime)

	// Prepare the data to be passed to the template
	data := struct {
		RecentAlerts          []Alert
		PriorityAlerts        []Alert
		PageLoadTime          string
		RecentQueryTime       string
		PriorityQueryTime     string
		PriorityCount         int
		PriorityBucketCount   int
		TotalPriorityLoadTime string
	}{
		RecentAlerts:          recentAlerts,
		PriorityAlerts:        priorityAlerts,
		PageLoadTime:          formatDuration(pageLoadTime),
		RecentQueryTime:       formatDuration(recentQueryTime),
		PriorityQueryTime:     formatDuration(priorityQueryTime + priorityCountTime),
		PriorityCount:         priorityCount,
		PriorityBucketCount:   priorityAlertsCount,
		TotalPriorityLoadTime: formatDuration(priorityQueryTime), // Initial load time
	}
	// Parse and execute the template
	tmpl := template.Must(template.ParseFiles("templates/index.html"))
	if err := tmpl.Execute(w, data); err != nil {
		log.Println("Error executing template:", err)
	}
}

func loadMoreAlertsHandler(w http.ResponseWriter, r *http.Request, client *mongo.Client) {
	bucketIndex, _ := strconv.Atoi(r.URL.Query().Get("bucketIndex"))
	alerts, queryTime, count := getPriorityAlerts(client, bucketIndex)

	w.Header().Set("Content-Type", "application/json")

	json.NewEncoder(w).Encode(struct {
		Alerts    []Alert `json:"alerts"`
		Count     int     `json:"count"`
		QueryTime string  `json:"queryTime"`
	}{
		Alerts:    alerts,
		Count:     count,
		QueryTime: formatDuration(queryTime),
	})
}

func formatDuration(d time.Duration) string {
	ms := float64(d.Microseconds()) / 1000.0
	return fmt.Sprintf("%.1f ms", ms)
}
