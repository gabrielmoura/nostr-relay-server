package cache

import (
	"strconv"
	"time"
)

func CheckSpam(key string, threshold int) (bool, error) {
	spamKey := key + "_spam"
	if !IsEnabled() {
		return false, nil
	}
	ctx, cancel := cacheContext()
	defer cancel()

	if checkSpamScript != nil {
		count, err := checkSpamScript.Run(ctx, redisClient.Raw(), []string{spamKey}, 300).Int64()
		if err != nil {
			return false, err
		}
		return count >= int64(threshold), nil
	}

	count, err := redisClient.Incr(ctx, spamKey)
	if err != nil {
		return false, err
	}
	if count == 1 {
		_ = redisClient.Expire(ctx, spamKey, 5*time.Minute)
	}
	return count >= int64(threshold), nil
}

func IncrCounter(key string) (int64, error) {
	if !IsEnabled() {
		return 0, nil
	}
	ctx, cancel := cacheContext()
	defer cancel()
	return redisClient.Incr(ctx, key)
}

func IncrCounterWithExpiry(key string, expiry time.Duration) (int64, error) {
	if !IsEnabled() {
		return 0, nil
	}
	ctx, cancel := cacheContext()
	defer cancel()
	count, err := redisClient.Incr(ctx, key)
	if err != nil {
		return count, err
	}
	if count == 1 {
		_ = redisClient.Expire(ctx, key, expiry)
	}
	return count, nil
}

func GetCounter(key string) (int64, error) {
	if !IsEnabled() {
		return 0, nil
	}
	val, err := Get(key)
	if err != nil {
		return 0, err
	}
	return strconv.ParseInt(val, 10, 64)
}
