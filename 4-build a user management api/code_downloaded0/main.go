package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

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

	r.HandleFunc("/", add).Methods(http.MethodPost)
	r.HandleFunc("/{id}", get).Methods(http.MethodGet)

	log.Println("started HTTP server....")
	log.Fatal(http.ListenAndServe(":8080", r))
}

const userHashNamePrefix = "user:"
const userIDCounter = "userid_counter"

// add—creates a new user
func add(w http.ResponseWriter, req *http.Request) {

	var user map[string]string
	err := json.NewDecoder(req.Body).Decode(&user)

	if err != nil {
		log.Println("failed to decode json payload", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	log.Println("user", user)

	id, err := client.Incr(context.Background(), userIDCounter).Result()
	if err != nil {
		log.Println("failed to generate userid", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	userHashName := userHashNamePrefix + strconv.Itoa(int(id))
	err = client.HSet(req.Context(), userHashName, user).Err()

	if err != nil {
		log.Println("failed to save user", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Add("Location", "http://"+req.Host+"/"+strconv.Itoa(int(id)))
	w.WriteHeader(http.StatusCreated)

	log.Println("added user", userHashName)
}

// get—queries a user by ID
func get(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	id := vars["id"]

	log.Println("searching for user", id)

	userHashName := userHashNamePrefix + id

	user, err := client.HGetAll(req.Context(), userHashName).Result()
	if err != nil {
		log.Println("error fetching user", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if len(user) == 0 {
		log.Println("user with id", id, "not found")
		http.Error(w, "user does not exist ", http.StatusNotFound)
		return
	}

	err = json.NewEncoder(w).Encode(user)
	if err != nil {
		log.Println("failed to encode user data", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

}