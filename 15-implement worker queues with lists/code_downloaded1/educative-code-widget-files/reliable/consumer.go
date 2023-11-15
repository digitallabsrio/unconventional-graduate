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
const emailTempQueueList = "jobs-temp"

func main() {
	client := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	_, err := client.Ping(context.Background()).Result()

	if err != nil {
		log.Fatal("ping failed. could not connect", err)
	}
	fmt.Println("reliable consumer ready")
	for {

		val, err := client.BLMove(context.Background(), emailQueueList, emailTempQueueList, "RIGHT", "LEFT", 2*time.Second).Result()

		if err != nil {
			if errors.Is(err, redis.Nil) {
				continue
			}
			log.Println("blmove issue", err)
		}

		job := val
		var jobInfo JobInfo

		err = json.Unmarshal([]byte(job), &jobInfo)
		if err != nil {
			log.Fatal("job info unmarshal issue issue", err)
		}

		fmt.Println("received job", jobInfo.JobId)
		fmt.Println("sending email", jobInfo.Email.Message, "to", jobInfo.Email.To)

		go func() {
			err = client.LRem(context.Background(), "jobs-temp", 0, job).Err()
			if err != nil {
				log.Fatal("lrem failed for", job, "error", err)
			}
			log.Println("removed job from temporary list", job)
		}()
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
