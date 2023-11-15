package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/go-redis/redis/v8"
)

const stream = "users"
const consumerGroup = "test-group"
const hashPrefix = "user:"

var client *redis.Client

func init() {
	client = redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	_, err := client.Ping(context.Background()).Result()

	if err != nil {
		log.Fatal("ping failed. could not connect", err)
	}

	client.XGroupCreateMkStream(context.Background(), stream, consumerGroup, "$")
}

func main() {
	fmt.Println("redis streams consumer ready")

	for {
		result, err := client.XReadGroup(context.Background(), &redis.XReadGroupArgs{
			Streams: []string{stream, ">"},
			Group:   consumerGroup,
			Block:   1 * time.Second,
		}).Result()

		if err != nil {
			if errors.Is(err, redis.Nil) {
				continue
			}
			log.Fatal("xreadgroup failed", err)
		}

		for _, s := range result {
			for _, message := range s.Messages {

				var hashName string
				for k, _ := range message.Values {
					hashName = hashPrefix + k
				}

				err = client.HSet(context.Background(), hashName, message.Values).Err()
				if err != nil {
					log.Fatal("failed to add user to hash", err)
				}
				fmt.Println("added user to hash", hashName)

				client.XAck(context.Background(), stream, consumerGroup, message.ID).Err()
				if err != nil {
					log.Fatal("failed to xack message ", message.ID, err)
				}

				fmt.Println("message", message.ID, "acknowledged")
			}
		}
	}

}
