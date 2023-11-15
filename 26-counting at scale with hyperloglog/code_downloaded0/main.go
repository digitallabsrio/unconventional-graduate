package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/go-redis/redis/v8"
)

var client *redis.Client

const numProducts = 100
const productViewsSortedSet = "top-products"

func init() {

	client = redis.NewClient(&redis.Options{Addr: "localhost:6379"})

	err := client.Ping(context.Background()).Err()
	if err != nil {
		log.Fatalf("failed to connect to redis. error message - %v", err)
	}

	fmt.Println("successfully connected to redis")
}

func main() {

	wait := make(chan os.Signal, 1)
	signal.Notify(wait, syscall.SIGINT)

	go generateViews()
	go storeProductViews()
	go displayTopViewedProducts(5)

	<-wait
}

func generateViews() {
	ctx := context.Background()
	pipe := client.Pipeline()

	for {
		for i := 1; i <= numProducts; i++ {
			name := "product:" + strconv.Itoa(i)
			fromIP := "ip" + strconv.Itoa(rand.Intn(100000)+1)
			for c := 1; c == rand.Intn(100)+1; c++ {
				pipe.PFAdd(ctx, name, fromIP)
			}
			_, err := pipe.Exec(ctx)
			if err != nil {
				log.Fatal("PFAdd pipe err", err)
			}
		}
	}
}

func storeProductViews() {
	ctx := context.Background()
	pipe := client.Pipeline()

	for {
		for i := 1; i <= numProducts; i++ {
			name := "product:" + strconv.Itoa(i)
			views := client.PFCount(ctx, name).Val()
			pipe.ZAdd(ctx, productViewsSortedSet, &redis.Z{Member: name, Score: float64(views)})
		}
		_, err := pipe.Exec(ctx)
		if err != nil {
			log.Fatal("ZAdd pipe err", err)
		}
	}
}

func displayTopViewedProducts(topN int) {
	ctx := context.Background()

	for {
		productViews := client.ZRevRangeWithScores(ctx, productViewsSortedSet, 0, int64(topN-1)).Val()
		fmt.Println("****** LEADERBAORD *******")
		for _, pv := range productViews {
			fmt.Println(pv.Member, "has", pv.Score, "views")
		}
		time.Sleep(3 * time.Second)
	}
}
