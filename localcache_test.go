package go_localcache

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func Test_Set(t *testing.T) {
	cache, err := New(time.Second*3, WithName("foo"))
	assert.Nil(t, err)

	ctx := context.Background()

	cache.Set(ctx, "foo", "bar")
	cache.Set(ctx, "foo2", "bar2")
	cache.Set(ctx, "foo3", "bar3")

	value, ok := cache.Get(ctx, "foo")
	assert.True(t, ok)
	assert.Equal(t, "bar", value)

	value, ok = cache.Get(ctx, "foo2")
	assert.True(t, ok)
	assert.Equal(t, "bar2", value)
}

func Test_Del(t *testing.T) {
	cache, err := New(time.Second * 3)
	assert.Nil(t, err)

	ctx := context.Background()

	cache.Set(ctx, "foo", "bar")
	cache.Set(ctx, "foo2", "bar2")

	cache.Del(ctx, "foo")

	value, ok := cache.Get(ctx, "foo")
	assert.False(t, ok)
	assert.Nil(t, value)

	value, ok = cache.Get(ctx, "foo2")
	assert.True(t, ok)
	assert.Equal(t, "bar2", value)
}

func Test_Take(t *testing.T) {
	cache, err := New(time.Second * 3)
	assert.Nil(t, err)

	var (
		counter int32
		wg      sync.WaitGroup
		ctx     = context.Background()
	)

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			value, errN := cache.Take(ctx, "foo", func(ctx context.Context) (interface{}, error) {
				atomic.AddInt32(&counter, 1)
				time.Sleep(time.Millisecond * 100)
				return "bar", nil
			})
			assert.Equal(t, "bar", value)
			assert.Nil(t, errN)
		}()
	}

	wg.Wait()

	assert.Equal(t, 1, cache.size())
	assert.Equal(t, int32(1), atomic.LoadInt32(&counter))
}

func Test_TakeExists(t *testing.T) {
	cache, err := New(time.Second * 3)
	assert.Nil(t, err)

	var (
		counter int32
		wg      sync.WaitGroup
		ctx     = context.Background()
	)

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			cache.Set(ctx, "foo", "bar")
			value, errN := cache.Take(ctx, "foo", func(ctx context.Context) (interface{}, error) {
				atomic.AddInt32(&counter, 1)
				time.Sleep(time.Millisecond * 100)
				return "bar", nil
			})
			assert.Equal(t, "bar", value)
			assert.Nil(t, errN)
		}()
	}

	wg.Wait()

	assert.Equal(t, 1, cache.size())
	assert.Equal(t, int32(0), atomic.LoadInt32(&counter))
}

func Test_TakeError(t *testing.T) {
	cache, err := New(time.Second * 3)
	assert.Nil(t, err)

	var (
		counter int32
		wg      sync.WaitGroup
		ctx     = context.Background()
		errNoob = errors.New("noob")
	)

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			value, errN := cache.Take(ctx, "foo", func(ctx context.Context) (interface{}, error) {
				atomic.AddInt32(&counter, 1)
				time.Sleep(time.Millisecond * 100)
				return nil, errNoob
			})
			assert.Nil(t, value)
			assert.ErrorIs(t, errN, errNoob)
		}()
	}

	wg.Wait()

	assert.Equal(t, 0, cache.size())
	assert.Equal(t, int32(1), atomic.LoadInt32(&counter))
}

func Test_WithLruEvicts(t *testing.T) {
	cache, err := New(time.Second*3, WithLimit(3))
	assert.Nil(t, err)

	var (
		ctx = context.Background()
	)

	cache.Set(ctx, "foo1", "bar1")
	cache.Set(ctx, "foo2", "bar2")
	cache.Set(ctx, "foo3", "bar3")
	cache.Set(ctx, "foo4", "bar4")

	get, ok := cache.Get(ctx, "foo1")
	assert.False(t, ok)
	assert.Nil(t, get)

	get, ok = cache.Get(ctx, "foo2")
	assert.True(t, ok)
	assert.Equal(t, "bar2", get)

	get, ok = cache.Get(ctx, "foo3")
	assert.True(t, ok)
	assert.Equal(t, "bar3", get)

	get, ok = cache.Get(ctx, "foo4")
	assert.True(t, ok)
	assert.Equal(t, "bar4", get)

}
