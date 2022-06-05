package dataloader

import (
	"errors"
	"sync"
)

var (
	ErrKeyNotFound = errors.New("key not found")
	ErrNoResult    = errors.New("no result")
	ErrTerminated  = errors.New("terminated")
)

type Result[T any] struct {
	res   T
	err   error
	dirty bool
	once  sync.Once
}

func (r *Result[T]) resolve(t T) *Result[T] {
	r.once.Do(func() {
		r.res = t
		r.dirty = true
	})

	return r
}

func (r *Result[T]) reject(err error) *Result[T] {
	r.once.Do(func() {
		r.err = err
		r.dirty = true
	})

	return r
}

func (r *Result[T]) Result() (t T) {
	if r.IsZero() {
		return
	}

	return r.res
}

func (r *Result[T]) Error() error {
	if r.IsZero() {
		return ErrNoResult
	}

	return r.err
}

func (r *Result[T]) Unwrap() (t T, err error) {
	return r.Result(), r.Error()
}

func (r *Result[T]) IsZero() bool {
	return r == nil || !r.dirty
}
