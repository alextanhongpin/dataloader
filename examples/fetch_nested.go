package main

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/alextanhongpin/dataloader"
)

func main() {
	defer func(start time.Time) {
		fmt.Println(time.Since(start))
	}(time.Now())

	loader, flush := NewLoader()
	defer flush()

	fetchOrderAggregate := func(ctx context.Context, ids []string) (map[string]*OrderAggregate, error) {
		result := make([]*OrderAggregate, len(ids))

		var wg sync.WaitGroup
		wg.Add(len(ids))
		for i, id := range ids {
			go func(i int, id string) {
				defer wg.Done()

				result[i] = NewOrderAggregateResolver(loader, id).LoadAll().Wait()
			}(i, id)
		}

		wg.Wait()

		output := make(map[string]*OrderAggregate, len(result))
		for i, id := range ids {
			output[id] = result[i]
		}

		return output, nil
	}
	dl, flush2 := dataloader.New(context.Background(), fetchOrderAggregate)
	defer flush2()

	n := 10
	ids := make([]string, n)
	for i := 0; i < n; i++ {
		ids[i] = fmt.Sprintf("order-%d", i)
	}
	result, err := dl.LoadMany(ids)
	if err != nil {
		panic(err)
	}

	// Output:
	// address keys: [order-5 order-6 order-3 order-4 order-7 order-9 order-2 order-1 order-0 order-8]
	// order keys: [order-7 order-2 order-0 order-6 order-4 order-9 order-1 order-5 order-3 order-8]
	// country keys: [country-order-0 country-order-6 country-order-1 country-order-8 country-order-3 country-order-4 country-order-2 country-order-5 country-order-7 country-order-9]
	// shipment keys: [8 5 3 7 6 1 2 4 9 0]
	// order aggregate 983.073205ms

	for _, res := range result {
		fmt.Printf("success: %#v %#v %#v\n", *res.Order, *res.Address, *res.Shipment)
	}
}

type OrderAggregateResolver struct {
	loader         *Loader
	orderID        string
	orderAggregate *OrderAggregate
	wg             sync.WaitGroup
}

func NewOrderAggregateResolver(loader *Loader, orderID string) *OrderAggregateResolver {
	return &OrderAggregateResolver{
		loader:         loader,
		orderID:        orderID,
		orderAggregate: new(OrderAggregate),
	}
}

func (r *OrderAggregateResolver) LoadOrder() {
	r.wg.Add(1)

	go func() {
		defer r.wg.Done()

		if order, err := r.loader.Order.Load(r.orderID); err == nil {
			r.orderAggregate.Order = &order

			if shipment, err := r.loader.Shipment.Load(order.ShipmentID); err == nil {
				r.orderAggregate.Shipment = &shipment
			}
		}
	}()
}

func (r *OrderAggregateResolver) LoadShipment() {
	r.wg.Add(1)

	go func() {
		defer r.wg.Done()

		if address, err := r.loader.Address.Load(r.orderID); err == nil {
			r.orderAggregate.Address = &address

			if country, err := r.loader.Country.Load(address.CountryID); err == nil {
				r.orderAggregate.Address.Country = &country
			}
		}
	}()
}

func (r *OrderAggregateResolver) LoadAll() *OrderAggregateResolver {
	r.LoadOrder()
	r.LoadShipment()
	return r
}

func (r *OrderAggregateResolver) Wait() *OrderAggregate {
	r.wg.Wait()

	return r.orderAggregate
}

type Country struct {
	_  struct{}
	ID string
}

type Address struct {
	_         struct{}
	ID        string
	CountryID string
	Country   *Country
}

type Shipment struct {
	_  struct{}
	ID int64
}

type Order struct {
	_          struct{}
	ID         string
	ShipmentID int64
}

type OrderAggregate struct {
	_        struct{}
	Order    *Order
	Address  *Address
	Shipment *Shipment
}

func randSleep() {
	duration := time.Duration(rand.Intn(1_000)) * time.Millisecond
	//fmt.Println("sleep for", duration)
	time.Sleep(duration)
}

func fetchOrders(ctx context.Context, keys []string) (map[string]Order, error) {
	fmt.Println("order keys:", keys)
	randSleep()

	result := make(map[string]Order)
	for i, key := range keys {
		result[key] = (Order{
			ID:         key,
			ShipmentID: int64(i),
		})
	}

	return result, nil
}

func fetchAddresses(ctx context.Context, keys []string) (map[string]Address, error) {
	fmt.Println("address keys:", keys)
	randSleep()

	result := make(map[string]Address)
	for _, key := range keys {
		result[key] = Address{
			ID:        "address-" + key,
			CountryID: "country-" + key,
		}
	}

	return result, nil
}

func fetchShipments(ctx context.Context, keys []int64) (map[int64]Shipment, error) {
	fmt.Println("shipment keys:", keys)
	randSleep()

	result := make(map[int64]Shipment)
	for _, key := range keys {
		result[key] = (Shipment{
			ID: key,
		})
	}

	return result, nil
}

func fetchCountries(ctx context.Context, keys []string) (map[string]Country, error) {
	fmt.Println("country keys:", keys)
	randSleep()

	result := make(map[string]Country)
	for _, key := range keys {
		result[key] = Country{ID: key}
	}

	return result, nil
}

type Loader struct {
	Order    *dataloader.Dataloader[string, Order]
	Address  *dataloader.Dataloader[string, Address]
	Shipment *dataloader.Dataloader[int64, Shipment]
	Country  *dataloader.Dataloader[string, Country]
}

func NewLoader() (*Loader, func()) {
	ctx := context.Background()
	order, flush1 := dataloader.New(ctx, fetchOrders)
	address, flush2 := dataloader.New(ctx, fetchAddresses)
	shipment, flush3 := dataloader.New(ctx, fetchShipments)
	country, flush4 := dataloader.New(ctx, fetchCountries)

	return &Loader{
			Order:    order,
			Address:  address,
			Shipment: shipment,
			Country:  country,
		}, func() {
			flush4()
			flush3()
			flush2()
			flush1()
		}
}
