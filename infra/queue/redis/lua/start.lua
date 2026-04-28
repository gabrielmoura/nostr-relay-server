-- KEYS[1] state
-- KEYS[2] attempts
-- KEYS[3] metrics
-- KEYS[4] meta key
--
-- ARGV[1] job id
-- ARGV[2] now ms
-- ARGV[3] metrics ttl seconds

redis.call("BITFIELD", KEYS[1], "SET", "u3", "#" .. ARGV[1], 2)
local attempts = redis.call("BITFIELD", KEYS[2], "INCRBY", "u8", "#" .. ARGV[1], 1)
redis.call("HSET", KEYS[4], "a", attempts[1], "la", ARGV[2], "sa", ARGV[2])
redis.call("HINCRBY", KEYS[3], "started", 1)
redis.call("EXPIRE", KEYS[3], ARGV[3])

return attempts[1]
