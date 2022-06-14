package main

import (
	"context"
	"fmt"
	"time"

	"github.com/alextanhongpin/dataloader"
)

type User struct {
	Name string
}

func fetchUsers(ctx context.Context, keys []string) (map[string]User, error) {
	m := make(map[string]User)
	for _, k := range keys {
		m[k] = User{k}
	}

	return m, nil
}

type Book struct {
	Title  string
	Author *dataloader.Result[User]
}

func main() {
	defer func(start time.Time) {
		fmt.Println(time.Since(start))
	}(time.Now())

	ctx := context.Background()
	dl, flush := dataloader.New(ctx, fetchUsers)
	defer flush()

	book := &Book{
		Title:  "hello world",
		Author: dl.LoadThunk("john-doe"),
	}

	user, err := book.Author.Unwrap()
	if err != nil {
		panic(err)
	}
	fmt.Println(user)

}
