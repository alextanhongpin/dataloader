package main

import (
	"context"
	"fmt"

	"github.com/alextanhongpin/dataloader"
)

func fetchNumber42(ctx context.Context, keys []int) (map[int]string, error) {
	fmt.Println("keys", keys)

	res := make(map[int]string)
	for _, k := range keys {
		res[k] = string(fmt.Sprintf("number-%d", k))
	}

	return res, nil
}

func main() {
	ctx := context.Background()
	dl, flush := dataloader.New(ctx, fetchNumber42)
	defer flush()

	dl.Prime(42, "the meaning of life")

	{
		// Not in cache yet.
		res, err := dl.Load(41)
		if err != nil {
			fmt.Println("failed:", err)
		} else {
			fmt.Println("success:", res)
		}
	}

	{
		// Already in cache.
		res, err := dl.Load(42)
		if err != nil {
			fmt.Println("failed:", err)
		} else {
			fmt.Println("success:", res)
		}
	}
}
