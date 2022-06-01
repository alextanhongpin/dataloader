package main

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/alextanhongpin/dataloader"
)

func fetchNumber(ctx context.Context, keys []int) (map[int]string, error) {
	fmt.Println("resolving", len(keys), "keys")
	fmt.Println("keys:", keys)

	result := make(map[int]string)
	for _, key := range keys {
		result[key] = fmt.Sprint(key)
	}

	return result, nil
}

func main() {
	ctx := context.Background()
	dld, flush := dataloader.New(
		ctx,
		fetchNumber,
		dataloader.WithBatchMaxKeys[int, string](20),
	)
	defer flush()

	n := 100
	addDelay := false

	var wg sync.WaitGroup
	wg.Add(n)

	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()

			if addDelay {
				duration := time.Duration(rand.Intn(1000)) * time.Millisecond
				time.Sleep(duration)
			}

			num := rand.Intn(n)
			res, err := dld.Load(num)
			if err != nil {
				fmt.Println("failed:", err)
			} else {
				fmt.Println("success:", res)
			}
		}()
	}

	wg.Wait()

	fmt.Println("completed")
}
