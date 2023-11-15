package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/gorilla/mux"
)

var client *redis.Client
var sub *redis.PubSub

const timeout = 10

func init() {
	client = redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	err := client.Ping(context.Background()).Err()
	if err != nil {
		log.Fatalf("failed to connect to redis. error message - %v", err)
	}
}

func main() {
	//Key miss events (events generated when a key that doesn't exist is accessed)
	sub = client.Subscribe(context.Background(), "__keyevent@0__:keymiss")

	go loadCache()

	r := mux.NewRouter()
	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "application is ready")
	}).Methods(http.MethodGet)

	r.HandleFunc("/{key}", get).Methods(http.MethodGet)

	log.Fatal(http.ListenAndServe(":8080", r))

}

func get(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	key := vars["key"]

	fmt.Println("searching for key", key)

	value, err := client.Get(req.Context(), key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			fmt.Println("key", key, "not found")
			http.Error(w, "key "+key+" not found", http.StatusNotFound)
			return
		}
		fmt.Println("error fetching key", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	fmt.Println("value for", key, "=", value)
	_, err = fmt.Fprintln(w, value)

	if err != nil {
		fmt.Println("failed to send value", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func loadCache() {
	fmt.Println("entered keyspace notification event loop")
	for event := range sub.Channel() {
		key := event.Payload

		value := getValueForKeyFromDB(key)
		err := client.Set(context.Background(), key, value, timeout*time.Second).Err()
		if err != nil {
			fmt.Println("set failed")
		}

		fmt.Println("[keyspace notification handler] set value for", key, "=", value, ".it will expire after few seconds")
	}
}

func getValueForKeyFromDB(key string) string {
	return key + "__" + strconv.Itoa(rand.Intn(1000)+1)
}