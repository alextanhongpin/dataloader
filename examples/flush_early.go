package main

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/alextanhongpin/dataloader"
)

type User struct {
	Name string
}

func fetchUsers(ctx context.Context, keys []string) (map[string]User, error) {
	fmt.Println("keys", keys)

	m := make(map[string]User)
	for _, k := range keys {
		m[k] = User{k}
	}

	return m, nil
}

func main() {
	ctx := context.Background()
	dl, flush := dataloader.New(ctx, fetchUsers)
	go func() {
		time.Sleep(500 * time.Millisecond)
		flush()
	}()

	n := 1_000
	addDelay := true

	var wg sync.WaitGroup
	wg.Add(n)

	for i := 0; i < n; i++ {
		go func(i int) {
			defer wg.Done()

			if addDelay {
				sleep := time.Duration(rand.Intn(2_000)) * time.Millisecond
				time.Sleep(sleep)
			}

			key := fmt.Sprint(rand.Intn(n / 2))

			res, err := dl.Load(key)
			if err != nil {
				fmt.Println("failed:", err, key)
			} else {
				fmt.Println("success:", res, key)
			}
		}(i)
	}

	wg.Wait()
}
