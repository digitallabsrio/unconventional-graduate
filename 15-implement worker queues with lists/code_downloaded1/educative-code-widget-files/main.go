package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"strconv"

	"github.com/go-redis/redis/v8"
	"github.com/gorilla/mux"
)

var client *redis.Client

func init() {

	client = redis.NewClient(&redis.Options{Addr: "localhost:6379"})

	err := client.Ping(context.Background()).Err()
	if err != nil {
		log.Fatalf("failed to connect to redis. error message - %v", err)
	}

	log.Println("successfully connected to redis")
}

func main() {

	r := mux.NewRouter()

	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "application is ready")
	}).Methods(http.MethodGet)

	r.HandleFunc("/", send).Methods(http.MethodPost)

	log.Println("started HTTP server....")
	log.Fatal(http.ListenAndServe(":8080", r))
}

func send(w http.ResponseWriter, req *http.Request) {
	var email Email
	err := json.NewDecoder(req.Body).Decode(&email)
	if err != nil {
		log.Fatal(err)
	}

	log.Println("recieved email request", email)

	jobID := strconv.Itoa(rand.Intn(1000) + 1)

	jobInfo := JobInfo{Email: email, JobId: jobID}
	job, err := json.Marshal(jobInfo)
	if err != nil {
		log.Fatal(err)
	}

	err = client.LPush(context.Background(), "jobs", job).Err()
	if err != nil {
		log.Fatal("lpush issue", err)
	}

	w.Header().Add("jobid", jobID)

}

type Email struct {
	To      string `json:"to"`
	Message string `json:"message"`
}

type JobInfo struct {
	Email Email  `json:"email"`
	JobId string `json:"id"`
}
