package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-redis/redis/v8"
	"github.com/gorilla/mux"
)

var client *redis.Client

const consumerGroup = "test-group"

func init() {

	client = redis.NewClient(&redis.Options{Addr: "localhost:6379"})

	err := client.Ping(context.Background()).Err()
	if err != nil {
		log.Fatalf("failed to connect to redis. error message - %v", err)
	}
	fmt.Println("successfully connected to redis")
}

func main() {

	r := mux.NewRouter()

	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "application is ready")
	}).Methods(http.MethodGet)

	r.HandleFunc("/", send).Methods(http.MethodPost)
	r.HandleFunc("/monitor", monitor).Methods(http.MethodGet)

	fmt.Println("started HTTP server")
	log.Fatal(http.ListenAndServe(":8080", r))
}

const stream = "users"

func send(w http.ResponseWriter, req *http.Request) {

	info, err := ioutil.ReadAll(req.Body)
	if err != nil {
		log.Fatal("failed to read request payload", err)
	}
	defer req.Body.Close()

	name := strings.Split(string(info), ",")[0]
	email := strings.Split(string(info), ",")[1]

	err = client.XAdd(context.Background(), &redis.XAddArgs{Stream: stream, Values: []interface{}{name, email}}).Err()

	if err != nil {
		log.Fatal("xadd issue", err)
	}

	fmt.Println("added user info to stream", name)

	w.Header().Add("user", name)
}

func monitor(w http.ResponseWriter, req *http.Request) {
	fmt.Println("fetching number of pending messages")

	pel := client.XPending(context.Background(), stream, consumerGroup).Val()
	fmt.Println("number of pending messages", pel.Count)

	w.Header().Add("X-Pending-Messages", strconv.Itoa(int(pel.Count)))
}
