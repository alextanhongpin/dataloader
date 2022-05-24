package main

import (
	"fmt"

	"github.com/alextanhongpin/dataloader"
)

func fetchNumber42(keys []int) (map[int]dataloader.Result[string], error) {
	fmt.Println("keys", keys)

	res := make(map[int]dataloader.Result[string])
	for _, k := range keys {
		res[k] = dataloader.Resolve[string](fmt.Sprintf("number-%d", k))
	}

	return res, nil
}

func main() {
	dl, flush := dataloader.New(fetchNumber42)
	defer flush()

	dl.Prime(42, "the meaning of life")

	{
		// Not in cache yet.
		result := dl.Load(41)
		if result.Ok() {
			fmt.Println(result.Data())
		} else {
			fmt.Println(result.Error())
		}
	}

	{
		// Already in cache.
		result := dl.Load(42)
		if result.Ok() {
			fmt.Println(result.Data())
		} else {
			fmt.Println(result.Error())
		}
	}
}
