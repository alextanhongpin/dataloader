package dataloader

import (
	"context"
	"fmt"
	"sync"
	"time"
)

const defaultBatchDuration = 16 * time.Millisecond

type Dataloader[K comparable, T any] struct {
	ch   chan K
	cond sync.Cond
	ctx  context.Context
	data map[K]*Result[T]
	done chan bool
	init sync.Once
	wg   sync.WaitGroup

	// How many keys gathered before the batchFn executes.
	batchMaxKeys int

	// How long elapsed before the batchFn executes.
	batchDuration time.Duration

	// How many concurrent batchFn is allowed to run.
	batchMaxWorker chan struct{}
	batchFn        BatchFunc[K, T]
}

type BatchFunc[K comparable, T any] func(ctx context.Context, keys []K) (map[K]T, error)

func New[K comparable, T any](ctx context.Context, batchFn BatchFunc[K, T], options ...Option[K, T]) (*Dataloader[K, T], func()) {
	dataloader := &Dataloader[K, T]{
		data:           make(map[K]*Result[T]),
		cond:           sync.Cond{L: &sync.Mutex{}},
		done:           make(chan bool),
		ch:             make(chan K),
		ctx:            ctx,
		batchDuration:  defaultBatchDuration,
		batchMaxKeys:   0,
		batchMaxWorker: make(chan struct{}, 1),
		batchFn:        batchFn,
	}

	for _, opt := range options {
		opt(dataloader)
	}

	var once sync.Once
	return dataloader, func() {
		dataloader.init.Do(func() {})

		once.Do(func() {
			close(dataloader.done)
			dataloader.wg.Wait()
		})
	}
}

func (l *Dataloader[K, T]) Load(key K) (t T, err error) {
	if res := l.load(key); !res.IsZero() {
		return res.Unwrap()
	}

	return l.wait(key)
}

func (l *Dataloader[K, T]) LoadMany(keys []K) (map[K]T, error) {
	result := make(map[K]T, len(keys))

	for _, key := range keys {
		if res := l.load(key); !res.IsZero() {
			t, err := res.Unwrap()
			if err != nil {
				return nil, err
			}

			result[key] = t
		}
	}

	for _, key := range keys {
		_, ok := result[key]
		if ok {
			continue
		}

		res, err := l.wait(key)
		if err != nil {
			return nil, err
		}

		result[key] = res
	}

	return result, nil
}

func (l *Dataloader[K, T]) Prime(key K, res T) {
	l.cond.L.Lock()

	l.data[key] = new(Result[T]).resolve(res)

	l.cond.L.Unlock()
	l.cond.Broadcast()
}

func (l *Dataloader[K, T]) wait(key K) (T, error) {
	l.cond.L.Lock()
	for l.pending(key) {
		l.cond.Wait()
	}

	res := l.data[key]
	l.cond.L.Unlock()

	return res.Unwrap()
}

func (l *Dataloader[K, T]) load(key K) *Result[T] {
	l.init.Do(func() {
		select {
		case <-l.done:
			return
		default:
			// Lazily create a background goroutine.
			l.loopAsync()
		}
	})

	l.cond.L.Lock()
	res, found := l.data[key]
	if !found {
		l.data[key] = new(Result[T])
	}
	l.cond.L.Unlock()

	if found {
		return res
	}

	// If it's not yet set, then set it.
	// Otherwise, the fetching might not be completed yet.
	select {
	case <-l.done:
		l.cond.L.Lock()
		res = l.data[key].reject(ErrTerminated)
		l.cond.L.Unlock()

		return res
	case l.ch <- key:
		return nil
	}
}

func (l *Dataloader[K, T]) pending(key K) bool {
	res, ok := l.data[key]

	return !ok || res.IsZero()
}

func (l *Dataloader[K, T]) batch(ctx context.Context, keys []K) {
	res, err := l.batchFn(ctx, keys)

	l.cond.L.Lock()

	for _, key := range keys {
		// If there's an error, set all results to the error.
		// Otherwise, the sync.Cond will wait forever.
		if err != nil {
			l.data[key].reject(err)

			continue
		}

		val, ok := res[key]
		if !ok {
			l.data[key].reject(fmt.Errorf("%w: %v", ErrKeyNotFound, key))
		} else {
			l.data[key].resolve(val)
		}
	}

	l.cond.L.Unlock()
	l.cond.Broadcast()
}

func (l *Dataloader[K, T]) batchAsync(ctx context.Context, keys []K) {
	if len(keys) == 0 {
		return
	}

	l.wg.Add(1)
	l.batchMaxWorker <- struct{}{}

	go func(keys []K) {
		defer func() {
			<-l.batchMaxWorker
			l.wg.Done()
		}()

		l.batch(ctx, keys)
	}(keys)
}

func (l *Dataloader[K, T]) loop() {
	ticker := time.NewTicker(l.batchDuration)
	defer ticker.Stop()

	ctx, cancel := context.WithCancel(l.ctx)
	defer cancel()

	keys := make([]K, 0, l.batchMaxKeys)

	for {
		select {
		case <-l.done:
			l.cond.L.Lock()

			for _, key := range keys {
				_, found := l.data[key]
				if found {
					l.data[key].reject(ErrTerminated)
					continue
				}
				l.data[key] = new(Result[T]).reject(ErrTerminated)
			}

			for key := range l.data {
				l.data[key].reject(ErrTerminated)
			}

			l.cond.L.Unlock()
			l.cond.Broadcast()

			return
		case <-ticker.C:
			l.batchAsync(ctx, keys)
			keys = nil
		case key := <-l.ch:
			ticker.Reset(l.batchDuration)

			keys = append(keys, key)
			if l.batchMaxKeys == 0 || len(keys) < l.batchMaxKeys {
				continue
			}

			l.batchAsync(ctx, keys)
			keys = nil
		}
	}
}

func (l *Dataloader[K, T]) loopAsync() {
	l.wg.Add(1)

	go func() {
		defer l.wg.Done()
		l.loop()
	}()
}
