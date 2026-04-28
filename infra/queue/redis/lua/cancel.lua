-- KEYS[1] state
-- KEYS[2] meta key
-- KEYS[3] metrics
--
-- ARGV[1] job id
-- ARGV[2] now ms
-- ARGV[3] metrics ttl seconds

redis.call("BITFIELD", KEYS[1], "SET", "u3", "#" .. ARGV[1], 7)
redis.call("HSET", KEYS[2], "fa", ARGV[2], "e", "canceled")
redis.call("HINCRBY", KEYS[3], "canceled", 1)
redis.call("EXPIRE", KEYS[3], ARGV[3])

return 1
