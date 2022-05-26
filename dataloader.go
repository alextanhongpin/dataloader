package dataloader

import (
	"errors"
	"fmt"
	"sync"
	"time"
)

var (
	ErrRejected = errors.New("rejected")
	ErrNoResult = errors.New("no result")
)

type Result[T any] struct {
	data  T
	err   error
	dirty bool
}

func Resolve[T any](t T) *Result[T] {
	return &Result[T]{
		data:  t,
		dirty: true,
	}
}

func Reject[T any](err error) *Result[T] {
	return &Result[T]{
		err:   err,
		dirty: true,
	}
}

func (r *Result[T]) Unwrap() (t T, err error) {
	if r.IsZero() {
		return t, ErrNoResult
	}
	return r.data, r.err
}

func (r *Result[T]) Data() (t T) {
	if r.IsZero() {
		return
	}

	return r.data
}

func (r *Result[T]) Error() error {
	if r.IsZero() {
		return ErrNoResult
	}

	return r.err
}

func (r *Result[T]) Ok() bool {
	return r.Error() == nil
}

func (r *Result[T]) IsZero() bool {
	return r == nil || !r.dirty
}

type Dataloader[K comparable, T any] struct {
	data     map[K]*Result[T]
	cond     sync.Cond
	done     chan bool
	init     sync.Once
	debounce chan struct{}

	cache         map[K]bool
	batchMaxKeys  int
	batchDuration time.Duration
	batchFn       func(keys []K) (map[K]*Result[T], error)
	start         time.Time
}

type Option[K comparable, T any] func(*Dataloader[K, T]) *Dataloader[K, T]

func WithBatchDuration[K comparable, T any](duration time.Duration) Option[K, T] {
	return func(dl *Dataloader[K, T]) *Dataloader[K, T] {
		dl.batchDuration = duration

		return dl
	}
}

func WithBatchMaxKeys[K comparable, T any](size int) Option[K, T] {
	return func(dl *Dataloader[K, T]) *Dataloader[K, T] {
		dl.batchMaxKeys = size

		return dl
	}
}

func New[K comparable, T any](batchFn func(keys []K) (map[K]*Result[T], error), options ...Option[K, T]) (*Dataloader[K, T], func()) {
	d := &Dataloader[K, T]{
		data:          make(map[K]*Result[T]),
		cond:          sync.Cond{L: &sync.Mutex{}},
		done:          make(chan bool),
		cache:         make(map[K]bool),
		debounce:      make(chan struct{}),
		batchDuration: 16 * time.Millisecond,
		batchMaxKeys:  0,
		batchFn:       batchFn,
		start:         time.Now(),
	}

	for _, opt := range options {
		opt(d)
	}

	var once sync.Once
	return d, func() {
		once.Do(func() {
			close(d.done)
		})
	}
}

func (l *Dataloader[K, T]) keysToFetch() []K {
	var keys []K

	l.cond.L.Lock()
	n := l.batchMaxKeys
	for key := range l.data {
		if l.cache[key] {
			continue
		}
		keys = append(keys, key)
		l.cache[key] = true
		n--
		if n == 0 {
			break
		}
	}
	l.cond.L.Unlock()

	return keys
}

func (l *Dataloader[K, T]) batch() {
	keys := l.keysToFetch()
	if len(keys) == 0 {
		return
	}
	res, err := l.batchFn(keys)

	l.cond.L.Lock()

	for _, key := range keys {
		// If there's an error, set all results to the error.
		// Otherwise, the sync.Cond will wait forever.
		if err != nil {
			l.data[key] = Reject[T](err)

			continue
		}

		val, ok := res[key]
		if !ok {
			l.data[key] = Reject[T](fmt.Errorf("%w: %v", ErrRejected, key))
		} else {
			l.data[key] = val
		}
	}

	l.cond.L.Unlock()
	l.cond.Broadcast()
}

func (l *Dataloader[K, T]) loop() {
	t := time.NewTicker(l.batchDuration)
	defer t.Stop()

	for {
		select {
		case <-l.done:
			return
		case <-l.debounce:
			t.Reset(l.batchDuration)
		case <-t.C:
			l.batch()
		}
	}
}

func (l *Dataloader[K, T]) LoadMany(keys []K) map[K]*Result[T] {
	tasks := make([]Task[T], len(keys))

	for i, key := range keys {
		key := key
		tasks[i] = func() *Result[T] {
			return l.Load(key)
		}
	}

	out := PromiseAll(tasks...)

	res := make(map[K]*Result[T])
	for i, key := range keys {
		res[key] = &out[i]
	}

	return res
}

func (l *Dataloader[K, T]) Load(key K) *Result[T] {
	l.init.Do(func() {
		// Lazily create a background goroutine.
		go l.loop()
	})

	l.cond.L.Lock()
	res, ok := l.data[key]
	l.cond.L.Unlock()

	if ok && !res.IsZero() {
		return res
	}

	// If it's not yet set, then set it.
	// Otherwise, the fetching might not be completed yet.
	if !ok {
		l.cond.L.Lock()
		l.data[key] = res
		l.cond.L.Unlock()

		l.debounce <- struct{}{}
	}

	l.cond.L.Lock()
	for l.isFetching(key) {
		l.cond.Wait()
	}

	res = l.data[key]
	l.cond.L.Unlock()

	return res
}

func (l *Dataloader[K, T]) Prime(key K, data T) {
	l.cond.L.Lock()
	l.data[key] = Resolve(data)
	l.cache[key] = true
	l.cond.L.Unlock()
	l.cond.Broadcast()
}

func (l *Dataloader[K, T]) isFetching(key K) bool {
	res, ok := l.data[key]

	return !ok || res.IsZero()
}

type Task[T any] func() *Result[T]

func PromiseAll[T any](tasks ...Task[T]) []Result[T] {
	if len(tasks) == 0 {
		return nil
	}

	var wg sync.WaitGroup
	wg.Add(len(tasks))

	result := make([]Result[T], len(tasks))

	for i, task := range tasks {
		go func(pos int, fn Task[T]) {
			defer wg.Done()

			result[pos] = *fn()
		}(i, task)
	}

	wg.Wait()

	return result
}

func Future[T any](task Task[T]) chan *Result[T] {
	ch := make(chan *Result[T], 1)

	go func() {
		ch <- task()
	}()

	return ch
}
