package main

import (
	"context"
	"fmt"

	"github.com/alextanhongpin/dataloader"
)

type Account struct {
	ID     string
	Data   map[string]string
	Status *Status
}

func (acc Account) Clone() Account {
	clone := acc
	clone.Data = make(map[string]string)
	for k, v := range acc.Data {
		clone.Data[k] = v
	}
	return clone
}

type Status struct {
	Name string
}

// Using *Account here would allow overriding the Status pointer.
func fetchAccounts(ctx context.Context, keys []string) (map[string]Account, error) {
	fmt.Println("keys", keys)

	m := make(map[string]Account)
	for _, k := range keys {
		m[k] = (Account{
			ID: fmt.Sprint(k),
			Data: map[string]string{
				"id": fmt.Sprint(k),
			},
			Status: &Status{
				Name: "pending",
			},
		})
	}

	return m, nil
}

func main() {
	ctx := context.Background()
	dl, flush := dataloader.New(ctx, fetchAccounts)
	defer flush()

	fmt.Println("fetch 1")
	account, err := dl.Load("account-1")
	if err != nil {
		panic(err)
	}
	fmt.Println("success:", account, *account.Status)

	account.ID = "override-id"
	account.Data["hello"] = "world"
	// Better practice: to avoid the map being mutated, perform a deep clone.
	//account = account.Clone()
	account.Status = &Status{Name: "success"}

	fmt.Println()
	fmt.Println("fetch 2")
	account, err = dl.Load("account-1")
	if err != nil {
		panic(err)
	}
	delete(account.Data, "id")
	fmt.Println("success:", account, *account.Status)

	fmt.Println()
	fmt.Println("fetch 3")
	account, err = dl.Load("account-1")
	if err != nil {
		panic(err)
	}
	fmt.Println("success:", account, *account.Status)

	// This library does not protect against mutation of reference object, due to
	// the way caching is implemented.
	// Deep-clone the result.Data() to protect against mutation if the same key is fetched.

	// Output:
	// success &{account-1 map[id:account-1]}
	// success &{account-1 map[hello:world]}
	// success &{account-1 map[hello:world]}

}
