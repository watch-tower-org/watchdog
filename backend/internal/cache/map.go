package cache

import "sync"

type MapCache[K comparable, V any] struct {
	mu     sync.RWMutex
	items  map[K]*V
	loaded bool
}

func NewMap[K comparable, V any]() *MapCache[K, V] {
	return &MapCache[K, V]{
		items: make(map[K]*V),
	}
}

func (c *MapCache[K, V]) Get(key K) (*V, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	item, exists := c.items[key]
	return item, exists
}

func (c *MapCache[K, V]) IsLoaded() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.loaded
}

func (c *MapCache[K, V]) LoadAll(items []V, keyFn func(*V) K) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items = make(map[K]*V, len(items))
	for i := range items {
		c.items[keyFn(&items[i])] = &items[i]
	}
	c.loaded = true
}

func (c *MapCache[K, V]) Add(key K, item *V) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items[key] = item
}

func (c *MapCache[K, V]) Remove(key K) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.items, key)
}

func (c *MapCache[K, V]) Invalidate() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items = make(map[K]*V)
	c.loaded = false
}

func (c *MapCache[K, V]) GetAll() map[K]*V {
	c.mu.RLock()
	defer c.mu.RUnlock()
	result := make(map[K]*V, len(c.items))
	for k, v := range c.items {
		result[k] = v
	}
	return result
}
