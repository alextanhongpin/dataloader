package main

import (
	"context"
	"fmt"
	"sync"

	"github.com/alextanhongpin/dataloader"
)

type User struct {
	ID string
}

func fetchUsers(ctx context.Context, userIDs []string) (map[string]User, error) {
	fmt.Println("fetching:", userIDs)
	m := make(map[string]User)
	for _, id := range userIDs {
		m[id] = User{
			ID: id,
		}
	}

	return m, nil
}

func main() {
	ctx := context.Background()
	dl, flush := dataloader.New(ctx, fetchUsers)
	defer flush()

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()

		u1, err := dl.Load("1")
		if err != nil {
			panic(err)
		}
		u2, err := dl.Load("2")
		if err != nil {
			panic(err)
		}
		fmt.Println(u1, u2)
	}()

	go func() {
		defer wg.Done()

		u2, err := dl.Load("2")
		if err != nil {
			panic(err)
		}
		u1, err := dl.Load("1")
		if err != nil {
			panic(err)
		}
		fmt.Println(u1, u2)
	}()

	wg.Wait()
}
