package cache

import (
	"sync"
	"testing"
)

type settings struct {
	ID   int64
	Name string
}

func TestSingletonCache(t *testing.T) {
	c := NewSingleton[settings]()
	if _, ok := c.Get(); ok {
		t.Fatal("empty cache should not return a value")
	}
	if c.IsLoaded() {
		t.Fatal("empty cache should not be loaded")
	}

	c.LoadAll(&settings{ID: 1, Name: "a"})
	if !c.IsLoaded() {
		t.Fatal("cache should be loaded after LoadAll")
	}
	got, ok := c.Get()
	if !ok || got.Name != "a" {
		t.Fatalf("Get = %+v, %v; want name a, true", got, ok)
	}

	c.Invalidate()
	if c.IsLoaded() {
		t.Fatal("cache should not be loaded after Invalidate")
	}
	if _, ok := c.Get(); ok {
		t.Fatal("cache should be empty after Invalidate")
	}
}

func TestMapCache(t *testing.T) {
	c := NewMap[int64, settings]()
	c.LoadAll([]settings{{ID: 1, Name: "a"}, {ID: 2, Name: "b"}}, func(v *settings) int64 { return v.ID })

	if !c.IsLoaded() {
		t.Fatal("map cache should be loaded")
	}
	got, ok := c.Get(1)
	if !ok || got.Name != "a" {
		t.Fatalf("Get(1) = %+v, %v; want name a, true", got, ok)
	}
	if _, ok := c.Get(99); ok {
		t.Fatal("missing key should not be found")
	}

	c.Add(3, &settings{ID: 3, Name: "c"})
	if _, ok := c.Get(3); !ok {
		t.Fatal("added key should be found")
	}
	if len(c.GetAll()) != 3 {
		t.Fatalf("GetAll len = %d; want 3", len(c.GetAll()))
	}

	c.Remove(3)
	if _, ok := c.Get(3); ok {
		t.Fatal("removed key should not be found")
	}

	c.Invalidate()
	if c.IsLoaded() {
		t.Fatal("map cache should not be loaded after Invalidate")
	}
	if len(c.GetAll()) != 0 {
		t.Fatal("map cache should be empty after Invalidate")
	}
}

func TestSingletonCache_Concurrent(t *testing.T) {
	c := NewSingleton[settings]()
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, ok := c.Get(); !ok {
				c.LoadAll(&settings{ID: 1})
			} else {
				c.Invalidate()
			}
		}()
	}
	wg.Wait()
}
