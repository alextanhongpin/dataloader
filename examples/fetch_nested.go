package main

import (
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/alextanhongpin/dataloader"
)

func main() {
	loader, flush := NewLoader()
	defer flush()

	start := time.Now()

	n := 10
	tasks := make([]dataloader.Task[OrderAggregate], n)
	for i := 0; i < n; i++ {
		i := i
		tasks[i] = func() *dataloader.Result[OrderAggregate] {
			orderAggregate := NewOrderAggregateResolver(loader, fmt.Sprintf("order-%d", i)).LoadAll().Wait()
			return dataloader.Resolve(*orderAggregate)
		}
	}
	result := dataloader.PromiseAll(tasks...)

	fmt.Println("order aggregate", time.Since(start))

	// Output:
	// address keys: [order-5 order-6 order-3 order-4 order-7 order-9 order-2 order-1 order-0 order-8]
	// order keys: [order-7 order-2 order-0 order-6 order-4 order-9 order-1 order-5 order-3 order-8]
	// country keys: [country-order-0 country-order-6 country-order-1 country-order-8 country-order-3 country-order-4 country-order-2 country-order-5 country-order-7 country-order-9]
	// shipment keys: [8 5 3 7 6 1 2 4 9 0]
	// order aggregate 983.073205ms

	for _, res := range result {
		if res.Ok() {
			data := res.Data()
			fmt.Printf("success: %#v %#v %#v\n", *data.Order, *data.Address, *data.Shipment)
		} else {
			fmt.Printf("failed: %s", res.Error())
		}
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

		if order, err := r.loader.Order.Load(r.orderID).Unwrap(); err == nil {
			r.orderAggregate.Order = &order

			if shipment, err := r.loader.Shipment.Load(order.ShipmentID).Unwrap(); err == nil {
				r.orderAggregate.Shipment = &shipment
			}
		}
	}()
}

func (r *OrderAggregateResolver) LoadShipment() {
	r.wg.Add(1)

	go func() {
		defer r.wg.Done()

		if address, err := r.loader.Address.Load(r.orderID).Unwrap(); err == nil {
			r.orderAggregate.Address = &address

			if country, err := r.loader.Country.Load(address.CountryID).Unwrap(); err == nil {
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

func fetchOrders(keys []string) (map[string]*dataloader.Result[Order], error) {
	fmt.Println("order keys:", keys)
	randSleep()

	result := make(map[string]*dataloader.Result[Order])
	for i, key := range keys {
		result[key] = dataloader.Resolve(Order{
			ID:         key,
			ShipmentID: int64(i),
		})
	}

	return result, nil
}

func fetchAddresses(keys []string) (map[string]*dataloader.Result[Address], error) {
	fmt.Println("address keys:", keys)
	randSleep()

	result := make(map[string]*dataloader.Result[Address])
	for _, key := range keys {
		result[key] = dataloader.Resolve(Address{
			ID:        "address-" + key,
			CountryID: "country-" + key,
		})
	}

	return result, nil
}

func fetchShipments(keys []int64) (map[int64]*dataloader.Result[Shipment], error) {
	fmt.Println("shipment keys:", keys)
	randSleep()

	result := make(map[int64]*dataloader.Result[Shipment])
	for _, key := range keys {
		result[key] = dataloader.Resolve(Shipment{
			ID: key,
		})
	}

	return result, nil
}

func fetchCountries(keys []string) (map[string]*dataloader.Result[Country], error) {
	fmt.Println("country keys:", keys)
	randSleep()

	result := make(map[string]*dataloader.Result[Country])
	for _, key := range keys {
		result[key] = dataloader.Resolve(Country{ID: key})
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
	order, flush1 := dataloader.New(fetchOrders)
	address, flush2 := dataloader.New(fetchAddresses)
	shipment, flush3 := dataloader.New(fetchShipments)
	country, flush4 := dataloader.New(fetchCountries)

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
