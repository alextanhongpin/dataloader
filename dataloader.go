package dataloader

import (
	"sync"
	"time"
)

type Result struct {
	Data interface{}
	Err  error
}

func (r *Result) IsSet() bool {
	return r.Data != nil || r.Err != nil
}

type Dataloader struct {
	data     map[string]Result
	cond     sync.Cond
	done     chan bool
	debounce chan struct{}

	cache         map[string]bool
	batchDuration time.Duration
	batchFn       func(keys []string) (map[string]Result, error)
	start         time.Time
}

func New(batchFn func(keys []string) (map[string]Result, error), batchDuration time.Duration) (*Dataloader, func()) {
	if batchDuration <= 0 {
		batchDuration = 16 * time.Millisecond
	}
	d := &Dataloader{
		data:          make(map[string]Result),
		cond:          sync.Cond{L: &sync.Mutex{}},
		done:          make(chan bool),
		cache:         make(map[string]bool),
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

func (l *Dataloader) keysToFetch() []string {
	var keys []string

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

func (l *Dataloader) batch() {
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
			l.data[key] = Result{Err: err}
		}
	} else {
		for k, v := range res {
			l.data[k] = v
		}
	}
	l.cond.L.Unlock()
	l.cond.Broadcast()
}

func (l *Dataloader) loop() {
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

func (l *Dataloader) LoadMany(keys []string) map[string]*Result {
	var mu sync.Mutex
	var wg sync.WaitGroup
	wg.Add(len(keys))

	res := make(map[string]*Result)

	for _, key := range keys {
		key := key
		go func(key string) {
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

func (l *Dataloader) Load(key string) *Result {
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

func (l *Dataloader) isFetching(key string) bool {
	res, ok := l.data[key]
	return !ok || !res.IsSet()
}
