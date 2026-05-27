-- Promote next users from waiting queue to active set
-- Called periodically by the backend to let next batch in
--
-- KEYS[1] = queue key (e.g., "waitingroom:{workshop_id}")
-- KEYS[2] = active tokens key (e.g., "waitingroom:active:{workshop_id}")
-- ARGV[1] = max concurrent users allowed
-- ARGV[2] = token TTL (seconds)
--
-- Returns: number of users promoted

local queue_key = KEYS[1]
local active_key = KEYS[2]
local max_active = tonumber(ARGV[1])
local token_ttl = tonumber(ARGV[2])

-- How many slots are available?
local active_count = redis.call('SCARD', active_key)
local available_slots = max_active - active_count

if available_slots <= 0 then
    return 0
end

-- Get the next N users from queue (lowest scores = earliest arrivals = FIFO)
local next_users = redis.call('ZRANGE', queue_key, 0, available_slots - 1)

local promoted = 0
for _, user_id in ipairs(next_users) do
    redis.call('ZREM', queue_key, user_id)
    redis.call('SADD', active_key, user_id)
    promoted = promoted + 1
end

if promoted > 0 then
    redis.call('EXPIRE', active_key, token_ttl)
end

return promoted
