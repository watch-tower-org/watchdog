package cache

import "sync"

type SingletonCache[T any] struct {
	mu     sync.RWMutex
	item   *T
	loaded bool
}

func NewSingleton[T any]() *SingletonCache[T] {
	return &SingletonCache[T]{}
}

func (c *SingletonCache[T]) Get() (*T, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.item, c.loaded && c.item != nil
}

func (c *SingletonCache[T]) IsLoaded() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.loaded
}

func (c *SingletonCache[T]) LoadAll(item *T) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.item = item
	c.loaded = true
}

func (c *SingletonCache[T]) Invalidate() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.item = nil
	c.loaded = false
}
