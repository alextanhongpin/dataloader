package main

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"runtime"
	"sync"
	"time"

	"github.com/alextanhongpin/dataloader"
)

type User struct {
	Name string
}

func fetchUsers(ctx context.Context, keys []string) (map[string]User, error) {
	fmt.Println("keys", keys)
	if len(keys) == 1 {
		return nil, errors.New("intended error")
	}
	time.Sleep(1 * time.Second)

	m := make(map[string]User)
	for _, k := range keys {
		m[k] = User{k}
	}

	return m, nil
}

func main() {
	defer func(start time.Time) {
		fmt.Println(time.Since(start))
	}(time.Now())

	ctx := context.Background()
	dl, flush := dataloader.New(ctx, fetchUsers,
		dataloader.WithBatchMaxKeys[string, User](1_000),
		dataloader.WithBatchMaxWorker[string, User](runtime.NumCPU()),
	)

	defer flush()

	n := 1_000
	addDelay := true

	var wg sync.WaitGroup
	wg.Add(n)

	for i := 0; i < n; i++ {
		go func(i int) {
			defer wg.Done()

			if addDelay {
				sleep := time.Duration(rand.Intn(1_000)) * time.Millisecond
				time.Sleep(sleep)
			}

			key := fmt.Sprint(rand.Intn(n / 2))

			res, err := dl.Load(key)
			if err != nil {
				fmt.Println("failed", err)
			} else {
				fmt.Println("success", res)
			}
		}(i)
	}

	wg.Wait()
}
