-- KEYS[1] stream key
-- KEYS[2] delayed
-- KEYS[3] state
-- KEYS[4] metrics
-- KEYS[5] meta key
--
-- ARGV[1] group
-- ARGV[2] stream entry id
-- ARGV[3] job id
-- ARGV[4] run at ms
-- ARGV[5] error summary
-- ARGV[6] now ms
-- ARGV[7] metrics ttl seconds

redis.call("XACK", KEYS[1], ARGV[1], ARGV[2])
redis.call("XDEL", KEYS[1], ARGV[2])
redis.call("BITFIELD", KEYS[3], "SET", "u3", "#" .. ARGV[3], 5)
redis.call("ZADD", KEYS[2], ARGV[4], ARGV[3])
redis.call("HSET", KEYS[5], "ra", ARGV[4], "la", ARGV[6], "e", ARGV[5])
redis.call("HINCRBY", KEYS[4], "retried", 1)
redis.call("EXPIRE", KEYS[4], ARGV[7])

return 1
