package main

import (
	"fmt"

	"github.com/alextanhongpin/dataloader"
)

type Book struct {
	_     struct{}
	ID    int64
	Title string
}

func fetchBooks(keys []int64) (map[int64]dataloader.Result[Book], error) {
	fmt.Println("keys", keys)

	m := make(map[int64]dataloader.Result[Book])
	for i, k := range keys {
		m[k] = dataloader.Resolve(Book{
			ID:    k,
			Title: fmt.Sprintf("book-%d", i),
		})
	}

	return m, nil
}

func main() {
	dl, flush := dataloader.New(fetchBooks)
	defer flush()

	books := dl.LoadMany([]int64{1, 2, 3, 4, 5, 4, 3, 2, 1})
	fmt.Println("fetched", len(books), "books")
	for id, book := range books {
		fmt.Println("book id", id)
		if book.Ok() {
			fmt.Println(book.Data())
		} else {
			fmt.Println(book.Error())
		}
	}
}
