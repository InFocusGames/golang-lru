package simplelru

import (
	"container/list"
	"errors"
)

var (
	ErrTooLargeWeight = errors.New("The new added entry's weight exceeds the lru size")
	ErrInvalidLRUSize = errors.New("Must provide a positive size")
)

// EvictCallback is used to get a callback when a cache entry is evicted
type EvictCallback func(key interface{}, value interface{})

// LRU implements a non-thread safe fixed size LRU cache
type LRU struct {
	size      int
	used      int
	evictList *list.List
	items     map[interface{}]*list.Element
	onEvict   EvictCallback
}

// entry is used to hold a value in the evictList
type entry struct {
	key    interface{}
	value  interface{}
	weight int
}

// option is used to specify options for adding the new entry.
type Option struct {
	Weight int // The weight of the added entry
}

// NewLRU constructs an LRU of the given size
func NewLRU(size int, onEvict EvictCallback) (*LRU, error) {
	if size <= 0 {
		return nil, ErrInvalidLRUSize
	}
	c := &LRU{
		size:      size,
		evictList: list.New(),
		items:     make(map[interface{}]*list.Element),
		onEvict:   onEvict,
	}
	return c, nil
}

// Purge is used to completely clear the cache
func (c *LRU) Purge() {
	for k, v := range c.items {
		if c.onEvict != nil {
			c.onEvict(k, v.Value.(*entry).value)
		}
		delete(c.items, k)
	}
	c.evictList.Init()
	c.used = 0
}

// Add adds a value to the cache.  Returns true if an eviction occurred.
func (c *LRU) Add(key, value interface{}, opt *Option) (error, bool) {
	// Check for existing item
	var weight int = 1
	if opt != nil && opt.Weight != 0 {
		weight = opt.Weight
	}

	if weight > c.size {
		return ErrTooLargeWeight, false
	}
	if ent, ok := c.items[key]; ok {
		c.evictList.MoveToFront(ent)
		original := ent.Value.(*entry)
		c.used = c.used - original.weight + weight
		ent.Value.(*entry).value, ent.Value.(*entry).weight = value, weight
		evict := c.used > c.size
		if evict {
			c.removeOldest()
		}
		return nil, evict
	}

	// Add new item
	ent := &entry{key, value, weight}
	entry := c.evictList.PushFront(ent)
	c.items[key] = entry
	c.used += ent.weight

	evict := c.used > c.size
	// Verify size not exceeded
	if evict {
		c.removeOldest()
	}
	return nil, evict
}

// Get looks up a key's value from the cache.
func (c *LRU) Get(key interface{}) (value interface{}, ok bool) {
	if ent, ok := c.items[key]; ok {
		c.evictList.MoveToFront(ent)
		return ent.Value.(*entry).value, true
	}
	return
}

// Check if a key is in the cache, without updating the recent-ness
// or deleting it for being stale.
func (c *LRU) Contains(key interface{}) (ok bool) {
	_, ok = c.items[key]
	return ok
}

// Returns the key value (or undefined if not found) without updating
// the "recently used"-ness of the key.
func (c *LRU) Peek(key interface{}) (value interface{}, ok bool) {
	if ent, ok := c.items[key]; ok {
		return ent.Value.(*entry).value, true
	}
	return nil, ok
}

// Remove removes the provided key from the cache, returning if the
// key was contained.
func (c *LRU) Remove(key interface{}) bool {
	if ent, ok := c.items[key]; ok {
		c.removeElement(ent)
		return true
	}
	return false
}

// RemoveOldest removes the oldest item from the cache.
func (c *LRU) RemoveOldest() (interface{}, interface{}, bool) {
	ent := c.evictList.Back()
	if ent != nil {
		c.removeElement(ent)
		kv := ent.Value.(*entry)
		return kv.key, kv.value, true
	}
	return nil, nil, false
}

// GetOldest returns the oldest entry
func (c *LRU) GetOldest() (interface{}, interface{}, bool) {
	ent := c.evictList.Back()
	if ent != nil {
		kv := ent.Value.(*entry)
		return kv.key, kv.value, true
	}
	return nil, nil, false
}

// Keys returns a slice of the keys in the cache, from oldest to newest.
func (c *LRU) Keys() []interface{} {
	keys := make([]interface{}, len(c.items))
	i := 0
	for ent := c.evictList.Back(); ent != nil; ent = ent.Prev() {
		keys[i] = ent.Value.(*entry).key
		i++
	}
	return keys
}

// Len returns the number of items in the cache.
func (c *LRU) Len() int {
	return c.evictList.Len()
}

// removeOldest removes the oldest item from the cache.
func (c *LRU) removeOldest() {
	for c.used > c.size {
		ent := c.evictList.Back()
		if ent != nil {
			c.removeElement(ent)
		}
	}
}

// removeElement is used to remove a given list element from the cache
func (c *LRU) removeElement(e *list.Element) {
	c.evictList.Remove(e)
	kv := e.Value.(*entry)
	delete(c.items, kv.key)
	if c.onEvict != nil {
		c.onEvict(kv.key, kv.value)
	}
	c.used -= kv.weight
}
