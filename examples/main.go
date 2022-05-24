package main

import (
	"errors"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/alextanhongpin/dataloader"
)

type User struct {
	Name string
}

func fetchUsers(keys []string) (map[string]dataloader.Result[User], error) {
	fmt.Println("keys", keys)
	if len(keys) == 1 {
		return nil, errors.New("intended error")
	}

	m := make(map[string]dataloader.Result[User])
	for _, k := range keys {
		m[k] = dataloader.Resolve(User{k})
	}

	return m, nil
}

func main() {
	dl, flush := dataloader.New(fetchUsers)
	defer flush()

	n := 100
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

			result := dl.Load(key)
			if result.Ok() {
				fmt.Println("success", result.Data())
			} else {
				fmt.Println("failed", result.Error())
			}
		}(i)
	}

	wg.Wait()
}
