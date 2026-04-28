-- KEYS[1] stream key
-- KEYS[2] dead
-- KEYS[3] state
-- KEYS[4] metrics
-- KEYS[5] meta key
-- KEYS[6] result key
--
-- ARGV[1] group
-- ARGV[2] stream entry id
-- ARGV[3] job id
-- ARGV[4] now ms
-- ARGV[5] error summary
-- ARGV[6] result json
-- ARGV[7] result ttl seconds
-- ARGV[8] metrics ttl seconds

redis.call("XACK", KEYS[1], ARGV[1], ARGV[2])
redis.call("XDEL", KEYS[1], ARGV[2])
redis.call("BITFIELD", KEYS[3], "SET", "u3", "#" .. ARGV[3], 6)
redis.call("ZADD", KEYS[2], ARGV[4], ARGV[3])
redis.call("HSET", KEYS[5], "fa", ARGV[4], "e", ARGV[5])
if ARGV[6] ~= "" then
  redis.call("SET", KEYS[6], ARGV[6], "EX", ARGV[7])
end
redis.call("HINCRBY", KEYS[4], "dead", 1)
redis.call("EXPIRE", KEYS[4], ARGV[8])

return 1
