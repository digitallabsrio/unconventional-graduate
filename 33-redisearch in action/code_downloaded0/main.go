package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/RediSearch/redisearch-go/redisearch"
	"github.com/gomodule/redigo/redis"
	"github.com/gorilla/mux"
)

var pool *redis.Pool
var client *redisearch.Client

const (
	indexName                 = "users-index"
	indexDefinitionHashPrefix = "user:"

	queryParamQuery       = "q"
	queryParamFields      = "fields"
	queryParamOffsetLimit = "offset_limit"

	responseHeaderSearchHits = "Search-Hits"
	responseHeaderPageSize   = "Page-Size"
)

func init() {
	pool = &redis.Pool{Dial: func() (redis.Conn, error) {
		return redis.Dial("tcp", "localhost:6379")
	}}

	client = redisearch.NewClientFromPool(pool, indexName)
	dropAndCreateIndex()
}

func main() {
	r := mux.NewRouter()

	r.HandleFunc("/search", search).Methods(http.MethodGet)
	r.HandleFunc("/load", load).Methods(http.MethodGet)

	fmt.Println("started HTTP server....")
	log.Fatal(http.ListenAndServe(":8080", r))
}

func dropAndCreateIndex() {
	err := client.DropIndex(true)
	if err != nil {
		fmt.Println("drop index failed ", err)
	}

	schema := redisearch.NewSchema(redisearch.DefaultOptions).
		AddField(redisearch.NewTextFieldOptions("name", redisearch.TextFieldOptions{})).
		AddField(redisearch.NewTextFieldOptions("email", redisearch.TextFieldOptions{})).
		AddField(redisearch.NewTextFieldOptions("city", redisearch.TextFieldOptions{})).
		AddField(redisearch.NewNumericFieldOptions("zipcode", redisearch.NumericFieldOptions{Sortable: true}))

	indexDefinition := redisearch.NewIndexDefinition().AddPrefix(indexDefinitionHashPrefix)

	err = client.CreateIndexWithIndexDefinition(schema, indexDefinition)
	if err != nil {
		log.Fatal("index creation failed ", err)
	}

	fmt.Println("redisearch index created")
}

func load(rw http.ResponseWriter, req *http.Request) {
	cities := []string{"New York", "Tel Aviv", "Dublin", "New Delhi", "New Jersey"}
	zipcodes := []int{123456, 789101, 234567, 891012, 345678}

	conn := pool.Get()

	go func() {
		for i := 1; i <= 100000; i++ {
			data := make(map[string]interface{})
			name := "user:" + strconv.Itoa(i)
			data["name"] = name
			data["email"] = name + "@foo.com"
			data["city"] = cities[rand.Intn(len(cities))]
			data["zipcode"] = zipcodes[rand.Intn(len(zipcodes))]

			val := redis.Args{data["name"]}.AddFlat(data)

			_, err := conn.Do("HSET", val...)
			if err != nil {
				fmt.Println("failed to add user", err)
				return
			}

			fmt.Println("added user", data["name"])
			time.Sleep(1 * time.Second)
		}
	}()
}

func search(rw http.ResponseWriter, req *http.Request) {

	searchParams, err := url.ParseQuery(req.URL.RawQuery)
	if err != nil {
		fmt.Println("invalid search criteria")
		http.Error(rw, err.Error(), http.StatusBadRequest)
		return
	}

	searchQuery := searchParams.Get(queryParamQuery)

	query := redisearch.NewQuery(searchQuery)

	fields := searchParams.Get(queryParamFields)
	if fields != "" {
		fmt.Println("fields to be returned", fields)
		toBeReturned := strings.Split(fields, ",")
		query = query.SetReturnFields(toBeReturned...)
	}

	offsetAndLimit := searchParams.Get(queryParamOffsetLimit)
	if offsetAndLimit != "" {
		fmt.Println("offset_limit", offsetAndLimit)
		offsetAndLimitVals := strings.Split(offsetAndLimit, ",")

		offset, err := strconv.Atoi(offsetAndLimitVals[0])
		if err != nil {
			http.Error(rw, "invalid offset", http.StatusBadRequest)
		}
		limit, err := strconv.Atoi(offsetAndLimitVals[1])
		if err != nil {
			http.Error(rw, "invalid limit", http.StatusBadRequest)
		}
		query = query.Limit(offset, limit)
	}

	docs, total, err := client.Search(query)

	if err != nil {
		status := http.StatusInternalServerError

		if strings.Contains(err.Error(), "Syntax error") {
			status = http.StatusBadRequest
		}
		fmt.Println("search failed")
		http.Error(rw, err.Error(), status)
		return
	}

	fmt.Printf("found %v docs matching query %s\n", total, searchQuery)
	fmt.Printf("showing %v docs in results as per offset and limit %v\n", len(docs), query.Paging)

	response := []map[string]interface{}{}
	for _, doc := range docs {
		fmt.Println("doc id", doc.Id)
		response = append(response, doc.Properties)
	}

	rw.Header().Add(responseHeaderSearchHits, strconv.Itoa(total))
	rw.Header().Add(responseHeaderPageSize, strconv.Itoa(len(docs)))

	err = json.NewEncoder(rw).Encode(response)
	if err != nil {
		fmt.Println("failed to encode response")
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}
}
