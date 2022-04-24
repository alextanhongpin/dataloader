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

func main() {
	fmt.Println("Hello, world!")
	dl, close := dataloader.New[string, User](func(keys []string) (map[string]dataloader.Result[User], error) {
		fmt.Println("batchFn", keys)
		if len(keys) == 1 {
			return nil, errors.New("intended error")
		}
		m := make(map[string]dataloader.Result[User])
		for _, k := range keys {
			m[k] = dataloader.Result[User]{Data: User{k}}
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
			fmt.Println("fetching", key, dl.Load(key).Data.Name)
		}(i)
	}

	res := dl.LoadMany([]string{"100", "200", "300", "100", "200", "300"})
	fmt.Println("get many", res)

	wg.Wait()
}
