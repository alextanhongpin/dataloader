package main

import (
	"context"
	"fmt"
	"time"

	"github.com/alextanhongpin/dataloader"
)

type Book struct {
	_     struct{}
	ID    int64
	Title string
}

func fetchBooks(ctx context.Context, keys []int64) (map[int64]Book, error) {
	defer func() {
		fmt.Println("fetched completed")
	}()

	fmt.Println("keys", keys)
	//if len(keys) == 5 {
	//return nil, errors.New("book error")
	//}
	select {
	case <-time.After(5 * time.Second):
		m := make(map[int64]Book)
		for i, k := range keys {
			m[k] = Book{
				ID:    k,
				Title: fmt.Sprintf("book-%d", i),
			}
		}

		return m, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func main() {
	defer func(start time.Time) {
		fmt.Println(time.Since(start))
	}(time.Now())

	ctx := context.Background()
	dl, flush := dataloader.New(ctx, fetchBooks)
	go func() {
		time.Sleep(500 * time.Millisecond)
		flush()
	}()
	defer flush()

	books, err := dl.LoadMany([]int64{1, 2, 3, 4, 5, 4, 3, 2, 1})
	if err != nil {
		fmt.Println(err)
	}

	fmt.Println("fetched", len(books), "books")
	for id, book := range books {
		fmt.Println("book id", id, book)
	}

	books, err = dl.LoadMany([]int64{10, 11, 12, 13, 14, 15})
	if err != nil {
		fmt.Println(err)
	}

	fmt.Println("fetched", len(books), "books")
	for id, book := range books {
		fmt.Println("book id", id, book)
	}
}
