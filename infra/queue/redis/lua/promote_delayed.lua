-- KEYS[1] delayed
-- KEYS[2] stream high
-- KEYS[3] stream normal
-- KEYS[4] stream low
-- KEYS[5] state
-- KEYS[6] metrics
-- KEYS[7] meta prefix
--
-- ARGV[1] now ms
-- ARGV[2] limit
-- ARGV[3] maxlen approx
-- ARGV[4] metrics ttl seconds

local ids = redis.call("ZRANGEBYSCORE", KEYS[1], "-inf", ARGV[1], "LIMIT", 0, ARGV[2])

for _, id in ipairs(ids) do
  local removed = redis.call("ZREM", KEYS[1], id)
  if removed == 1 then
    local current = redis.call("BITFIELD", KEYS[5], "GET", "u3", "#" .. id)
    if current[1] ~= 7 then
      local metaKey = KEYS[7] .. id
      local priority = redis.call("HGET", metaKey, "p")
      local streamKey = KEYS[3]
      if priority == "high" then
        streamKey = KEYS[2]
      elseif priority == "low" then
        streamKey = KEYS[4]
      end
      redis.call("BITFIELD", KEYS[5], "SET", "u3", "#" .. id, 1)
      redis.call("HDEL", metaKey, "ra")
      redis.call("XADD", streamKey, "MAXLEN", "~", ARGV[3], "*", "i", id)
    end
  end
end

if #ids > 0 then
  redis.call("HINCRBY", KEYS[6], "promoted", #ids)
  redis.call("EXPIRE", KEYS[6], ARGV[4])
end

return ids
