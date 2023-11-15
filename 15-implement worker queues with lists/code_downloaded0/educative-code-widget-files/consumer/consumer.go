package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/go-redis/redis/v8"
)

const emailQueueList = "jobs"

func main() {
	client := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	_, err := client.Ping(context.Background()).Result()

	if err != nil {
		log.Fatal("ping failed. could not connect", err)
	}
	fmt.Println("consumer ready")
	for {
		val, err := client.BRPop(context.Background(), 2*time.Second, emailQueueList).Result()
		if err != nil {
			if errors.Is(err, redis.Nil) {
				continue
			}
			log.Println("brpop issue", err)
		}

		job := val[1]
		var jobInfo JobInfo

		err = json.Unmarshal([]byte(job), &jobInfo)
		if err != nil {
			log.Fatal("job info unmarshal issue issue", err)
		}

		fmt.Println("received job", jobInfo.JobId)
		fmt.Println("sending email", jobInfo.Email.Message, "to", jobInfo.Email.To)
	}

}

type Email struct {
	To      string `json:"to"`
	Message string `json:"message"`
}

type JobInfo struct {
	Email Email  `json:"email"`
	JobId string `json:"id"`
}
