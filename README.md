# dataloader

Dataloader implemented with golang. Also see the alternative implementation without sync.Cond [here](https://github.com/alextanhongpin/dataloader2).


## Installation

```
go get github.com/alextanhongpin/dataloader
```


## Usage

In the example below, we run two goroutines to fetch user with the id `1` and `2` respectively. Without dataloader, the application would have performed 4 calls to the database.

With dataloader, only 1 call will be made to fetch the users.

```go
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
```
