package dataloader

import (
	"sync"
	"time"
)

type Result[T any] struct {
	Data  T
	Err   error
	dirty bool
}

func (r *Result[T]) IsSet() bool {
	return r.dirty
}

type Dataloader[K comparable, T any] struct {
	data     map[K]Result[T]
	cond     sync.Cond
	done     chan bool
	debounce chan struct{}

	cache         map[K]bool
	batchDuration time.Duration
	batchFn       func(keys []K) (map[K]Result[T], error)
	start         time.Time
}

func New[K comparable, T any](batchFn func(keys []K) (map[K]Result[T], error), batchDuration time.Duration) (*Dataloader[K, T], func()) {
	if batchDuration <= 0 {
		batchDuration = 16 * time.Millisecond
	}
	d := &Dataloader[K, T]{
		data:          make(map[K]Result[T]),
		cond:          sync.Cond{L: &sync.Mutex{}},
		done:          make(chan bool),
		cache:         make(map[K]bool),
		debounce:      make(chan struct{}),
		batchDuration: batchDuration,
		batchFn:       batchFn,
		start:         time.Now(),
	}
	go d.loop()

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
	for key := range l.data {
		if l.cache[key] {
			continue
		}
		keys = append(keys, key)
		l.cache[key] = true
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
	if err != nil {
		// If there's an error, set all results to the error.
		// Otherwise, the sync.Cond will wait forever.
		for _, key := range keys {
			l.data[key] = Result[T]{Err: err, dirty: true}
		}
	} else {
		for k, v := range res {
			v.dirty = true
			l.data[k] = v
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
	var mu sync.Mutex
	var wg sync.WaitGroup
	wg.Add(len(keys))

	res := make(map[K]*Result[T])

	for _, key := range keys {
		key := key
		go func(key K) {
			defer wg.Done()

			out := l.Load(key)

			mu.Lock()
			res[key] = out
			mu.Unlock()
		}(key)
	}

	wg.Wait()
	return res
}

func (l *Dataloader[K, T]) Load(key K) *Result[T] {
	l.cond.L.Lock()
	res, ok := l.data[key]
	l.cond.L.Unlock()

	if ok && res.IsSet() {
		return &res
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

	return &res
}

func (l *Dataloader[K, T]) Prime(key K, data T) {
	l.cond.L.Lock()
	l.data[key] = Result[T]{Data: data, dirty: true}
	l.cond.L.Unlock()
	l.cond.Broadcast()
}

func (l *Dataloader[K, T]) isFetching(key K) bool {
	res, ok := l.data[key]
	return !ok || !res.IsSet()
}
