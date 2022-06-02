package dataloader

import "time"

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

func WithBatchMaxWorker[K comparable, T any](worker int) Option[K, T] {
	return func(dl *Dataloader[K, T]) *Dataloader[K, T] {
		dl.batchMaxWorker = make(chan struct{}, worker)

		return dl
	}
}
