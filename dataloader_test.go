package dataloader_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/alextanhongpin/dataloader"
)

func TestLoad(t *testing.T) {
	t.Parallel()

	fetchNumber := func(ctx context.Context, keys []int) (map[int]string, error) {
		res := make(map[int]string)

		for _, key := range keys {
			res[key] = fmt.Sprint(key)
		}

		return res, nil
	}

	ctx := context.Background()

	dl, flush := dataloader.New(ctx, fetchNumber)
	t.Cleanup(flush)

	t.Run("load once", func(t *testing.T) {
		t.Parallel()

		res, err := dl.Load(42)
		if err != nil {
			t.FailNow()
		}

		if exp, got := "42", res; exp != got {
			t.Fatalf("expected %s, got %s", exp, got)
		}
	})

	t.Run("load many", func(t *testing.T) {
		t.Parallel()

		keys := []int{1, 2, 3, 4, 5, 4, 3, 2, 1}
		res, err := dl.LoadMany(keys)
		if err != nil {
			t.FailNow()
		}

		if exp, got := 5, len(res); exp != got {
			t.Fatalf("expected %v, got %v", exp, got)
		}

		for _, key := range keys {
			if exp, got := fmt.Sprint(key), res[key]; exp != got {
				t.Fatalf("expected %v, got %v", exp, got)
			}
		}
	})

	t.Run("primed", func(t *testing.T) {
		t.Parallel()

		dl.Prime(100, "one-hundred")
		res, err := dl.Load(100)
		if err != nil {
			t.FailNow()
		}

		if exp, got := "one-hundred", res; exp != got {
			t.Fatalf("expected %v, got %v", exp, got)
		}

		res, err = dl.Load(200)
		if err != nil {
			t.FailNow()
		}

		if exp, got := "200", res; exp != got {
			t.Fatalf("expected %v, got %v", exp, got)
		}
	})
}

func TestFlush(t *testing.T) {
	t.Parallel()

	fetchNumber := func(ctx context.Context, keys []int) (map[int]string, error) {
		res := make(map[int]string)

		for _, key := range keys {
			res[key] = fmt.Sprint(key)
		}

		return res, nil
	}

	ctx := context.Background()

	dl, flush := dataloader.New(ctx, fetchNumber)
	flush()

	t.Run("load", func(t *testing.T) {
		t.Parallel()

		res, err := dl.Load(42)
		if !errors.Is(err, dataloader.ErrTerminated) {
			t.Fatalf("expected %v, got %v", dataloader.ErrTerminated, err)
		}

		if exp, got := "", res; exp != got {
			t.Fatalf("expected %v, got %v", exp, got)
		}
	})

	t.Run("load_many", func(t *testing.T) {
		t.Parallel()

		res, err := dl.LoadMany([]int{100, 200})
		if !errors.Is(err, dataloader.ErrTerminated) {
			t.Fatalf("expected %v, got %v", dataloader.ErrTerminated, err)
		}

		if exp, got := 0, len(res); exp != got {
			t.Fatalf("expected %v, got %v", exp, got)
		}
	})
}

func TestMissingKey(t *testing.T) {
	t.Parallel()

	fetchNumber := func(ctx context.Context, keys []int) (map[int]string, error) {
		return nil, nil
	}

	ctx := context.Background()

	dl, flush := dataloader.New(ctx, fetchNumber)
	t.Cleanup(flush)

	t.Run("no key", func(t *testing.T) {
		t.Parallel()

		_, err := dl.Load(42)
		if exp, got := true, errors.Is(err, dataloader.ErrKeyNotFound); exp != got {
			t.Fatalf("expected %v, got %v", exp, got)
		}
	})

	t.Run("zero keys", func(t *testing.T) {
		t.Parallel()

		_, err := dl.LoadMany(nil)
		if exp, got := true, errors.Is(err, nil); exp != got {
			t.Fatalf("expected %v, got %v", exp, got)
		}
	})
}
