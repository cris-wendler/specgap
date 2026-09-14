# Task: a cache with expiry and a size limit

Implement `Cache` in `cache.go`. The tests in `cache_test.go` describe
what is wanted. Make them pass.

```go
func New(capacity int, ttl time.Duration) *Cache
func (c *Cache) Set(key string, value int)
func (c *Cache) Get(key string) (int, bool)
func (c *Cache) Len() int
```

- `Set` stores a value under a key.
- `Get` returns the value and whether it was there.
- An entry older than `ttl` is not returned.
- The cache holds at most `capacity` entries. When a new entry would
  exceed it, the least recently used entry is removed.
- `Len` reports how many entries the cache holds.

Time is read through the package variable `now`, so tests can move the
clock. Use `now()` rather than `time.Now()`.
