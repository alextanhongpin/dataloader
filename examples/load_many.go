package main

import (
	"context"
	"errors"
	"fmt"

	"github.com/alextanhongpin/dataloader"
)

type Book struct {
	_     struct{}
	ID    int64
	Title string
}

func fetchBooks(ctx context.Context, keys []int64) (map[int64]Book, error) {
	fmt.Println("keys", keys)
	if len(keys) == 5 {
		return nil, errors.New("book error")
	}

	m := make(map[int64]Book)
	for i, k := range keys {
		m[k] = Book{
			ID:    k,
			Title: fmt.Sprintf("book-%d", i),
		}
	}

	return m, nil
}

func main() {
	ctx := context.Background()
	dl, flush := dataloader.New(ctx, fetchBooks)
	defer flush()

	books, err := dl.LoadMany([]int64{1, 2, 3, 4, 5, 4, 3, 2, 1})
	if err != nil {
		panic(err)
	}

	fmt.Println("fetched", len(books), "books")
	for id, book := range books {
		fmt.Println("book id", id, book)
	}
}
