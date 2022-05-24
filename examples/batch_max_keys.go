package main

import (
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/alextanhongpin/dataloader"
)

func fetchNumber(keys []int) (map[int]dataloader.Result[string], error) {
	fmt.Println("resolving", len(keys), "keys")
	fmt.Println("keys:", keys)

	result := make(map[int]dataloader.Result[string])
	for _, key := range keys {
		result[key] = dataloader.Resolve(fmt.Sprint(key))
	}

	return result, nil
}

func main() {
	dld, flush := dataloader.New(
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
			result := dld.Load(num)
			fmt.Println("fetched")
			if result.Ok() {
				fmt.Println(result.Data())
			} else {
				fmt.Println(result.Error())
			}
		}()
	}

	wg.Wait()

	fmt.Println("completed")
}
