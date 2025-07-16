package cache

import (
	"github.com/dgraph-io/ristretto/v2"
	"time"
)

var cache *ristretto.Cache[string, any]

func Init() error {
	var err error
	cache, err = ristretto.NewCache(&ristretto.Config[string, any]{
		NumCounters: 1e7, // number of keys to track frequency of (10M).
		// maximum cost of cache (10MB).
		MaxCost:     1 << 20,
		BufferItems: 64, // number of keys per Get buffer.
	})
	return err
}
func Set() bool {
	result := cache.Set("key", "value", 1)

	// wait for value to pass through buffers
	defer cache.Wait()
	return result
}
func Get(key string) (any, bool) {
	value, found := cache.Get(key)
	if !found {
		return "", false
	}
	return value, true
}
func Delete(key string) bool {
	cache.Del(key)

	// wait for value to pass through buffers
	defer cache.Wait()
	return true
}
func SetWithTTL(key string, value any, ttl time.Duration) bool {
	result := cache.SetWithTTL(key, value, 1, ttl)

	// wait for value to pass through buffers
	defer cache.Wait()
	return result
}

// CheckSpam checks if the number of times a key has been accessed exceeds a given threshold.
func CheckSpam(key string, num int) (bool, error) {
	nKey := key + "_spam"
	value, found := cache.Get(nKey)
	if !found {
		cache.SetWithTTL(nKey, num, 1, time.Minute*5)
		return false, nil
	}
	if value.(int) >= num {
		return true, nil
	}
	cache.SetWithTTL(nKey, value.(int)+1, 1, time.Minute*5)
	return false, nil
}
