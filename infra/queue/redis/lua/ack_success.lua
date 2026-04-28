-- KEYS[1] stream key
-- KEYS[2] body key
-- KEYS[3] state
-- KEYS[4] metrics
-- KEYS[5] meta key
-- KEYS[6] result key
--
-- ARGV[1] group
-- ARGV[2] stream entry id
-- ARGV[3] job id
-- ARGV[4] duration ms
-- ARGV[5] finished at ms
-- ARGV[6] result json
-- ARGV[7] result ttl seconds
-- ARGV[8] metrics ttl seconds

redis.call("XACK", KEYS[1], ARGV[1], ARGV[2])
redis.call("XDEL", KEYS[1], ARGV[2])
redis.call("DEL", KEYS[2])
redis.call("BITFIELD", KEYS[3], "SET", "u3", "#" .. ARGV[3], 3)
redis.call("HSET", KEYS[5], "fa", ARGV[5], "e", "")
if ARGV[6] ~= "" then
  redis.call("SET", KEYS[6], ARGV[6], "EX", ARGV[7])
end
redis.call("HINCRBY", KEYS[4], "succeeded", 1)
redis.call("HINCRBY", KEYS[4], "duration_ms_sum", ARGV[4])
redis.call("EXPIRE", KEYS[4], ARGV[8])

return 1
