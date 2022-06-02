package main

import (
	"context"
	"fmt"
	"math/rand"
	"runtime"
	"sync"
	"time"

	"github.com/alextanhongpin/dataloader"
)

func init() {
	rand.Seed(42)
}

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
	case <-time.After(time.Duration(rand.Intn(10)+1) * time.Second):
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
	dl, flush := dataloader.New(ctx, fetchBooks,
		dataloader.WithBatchMaxKeys[int64, Book](1_000),
		dataloader.WithBatchMaxWorker[int64, Book](runtime.NumCPU()),
	)
	defer flush()

	nWorker := 10
	minTask, maxTask := 10, 100
	var wg sync.WaitGroup
	wg.Add(nWorker)

	for i := 0; i < nWorker; i++ {
		go func(i int) {
			defer wg.Done()

			nTask := rand.Intn(maxTask-minTask) + minTask
			bookIDs := make([]int64, nTask)
			for i := 0; i < nTask; i++ {
				bookIDs[i] = rand.Int63n(1_000_000)
			}

			time.Sleep(time.Duration(rand.Intn(500)) * time.Millisecond)
			fmt.Println("got bookIDs", bookIDs)
			books, err := dl.LoadMany(bookIDs)
			if err != nil {
				fmt.Println(err)
			}

			fmt.Println("fetched", len(books), "books", i)
		}(i)
	}

	wg.Wait()
}
