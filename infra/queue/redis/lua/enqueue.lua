-- KEYS[1]  seq
-- KEYS[2]  body prefix
-- KEYS[3]  state
-- KEYS[4]  attempts
-- KEYS[5]  stream high
-- KEYS[6]  stream normal
-- KEYS[7]  stream low
-- KEYS[8]  delayed
-- KEYS[9]  metrics
-- KEYS[10] meta prefix
-- KEYS[11] jobs index
-- KEYS[12] result prefix
--
-- ARGV[1]  body
-- ARGV[2]  job name
-- ARGV[3]  priority
-- ARGV[4]  now ms
-- ARGV[5]  run at ms
-- ARGV[6]  body ttl seconds
-- ARGV[7]  stream maxlen approx
-- ARGV[8]  queue
-- ARGV[9]  max attempts
-- ARGV[10] result ttl seconds

local id = redis.call("INCR", KEYS[1])
local bodyKey = KEYS[2] .. id
local metaKey = KEYS[10] .. id
local resultKey = KEYS[12] .. id

redis.call("SET", bodyKey, ARGV[1], "EX", ARGV[6])
redis.call("BITFIELD", KEYS[4], "SET", "u8", "#" .. id, 0)
redis.call(
  "HSET",
  metaKey,
  "j", ARGV[2],
  "q", ARGV[8],
  "p", ARGV[3],
  "a", 0,
  "ma", ARGV[9],
  "ca", ARGV[4]
)
redis.call("EXPIRE", metaKey, ARGV[6])
redis.call("DEL", resultKey)
redis.call("ZADD", KEYS[11], ARGV[4], id)

if tonumber(ARGV[5]) > tonumber(ARGV[4]) then
  redis.call("BITFIELD", KEYS[3], "SET", "u3", "#" .. id, 5)
  redis.call("HSET", metaKey, "ra", ARGV[5])
  redis.call("ZADD", KEYS[8], ARGV[5], id)
  redis.call("HINCRBY", KEYS[9], "delayed", 1)
else
  redis.call("BITFIELD", KEYS[3], "SET", "u3", "#" .. id, 1)
  local streamKey = KEYS[6]
  if ARGV[3] == "high" then
    streamKey = KEYS[5]
  elseif ARGV[3] == "low" then
    streamKey = KEYS[7]
  end
  redis.call("XADD", streamKey, "MAXLEN", "~", ARGV[7], "*", "i", id)
  redis.call("HINCRBY", KEYS[9], "queued", 1)
end

redis.call("HINCRBY", KEYS[9], "enqueued", 1)
redis.call("EXPIRE", KEYS[9], ARGV[10])

return id
