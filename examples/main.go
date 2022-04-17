package main

import (
	"errors"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/alextanhongpin/dataloader"
)

func main() {
	dl, close := dataloader.New(func(keys []string) (map[string]dataloader.Result, error) {
		fmt.Println("batchFn", keys)
		if len(keys) == 1 {
			return nil, errors.New("intended error")
		}
		m := make(map[string]dataloader.Result)
		for _, k := range keys {
			m[k] = dataloader.Result{Data: k}
		}
		return m, nil
	}, 16*time.Millisecond)
	defer close()

	n := 100

	var wg sync.WaitGroup
	wg.Add(n)

	for i := 0; i < n; i++ {
		go func(i int) {
			defer wg.Done()
			sleep := time.Duration(rand.Intn(1000)) * time.Millisecond
			time.Sleep(sleep)

			key := fmt.Sprint(rand.Intn(n / 2))
			fmt.Println("fetching", key, dl.Load(key))
		}(i)
	}

	res := dl.LoadMany([]string{"100", "200", "300"})
	fmt.Println("get many", res)

	wg.Wait()
}
