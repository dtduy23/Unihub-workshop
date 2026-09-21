-- Promote next users from waiting queue to active set
-- Called periodically by backend or during lazy promotion
--
-- KEYS[1] = queue key (Sorted Set: user_id -> priority score)
-- KEYS[2] = active tokens key (Sorted Set: user_id -> expiration timestamp)
-- KEYS[3] = heartbeat key (Sorted Set: user_id -> heartbeat expiration timestamp)
--
-- ARGV[1] = max concurrent users allowed
-- ARGV[2] = token TTL (seconds)
-- ARGV[3] = current timestamp (seconds, float)
--
-- Returns: number of users promoted

local queue_key = KEYS[1]
local active_key = KEYS[2]
local heartbeat_key = KEYS[3]

local max_active = tonumber(ARGV[1])
local token_ttl = tonumber(ARGV[2])
local now = tonumber(ARGV[3])

-- 1. Purge expired active tokens
redis.call('ZREMRANGEBYSCORE', active_key, 0, now)

-- 2. Purge dead waiting users (heartbeat expired > 60s without ping)
local dead_users = redis.call('ZRANGEBYSCORE', heartbeat_key, 0, now)
if #dead_users > 0 then
    for _, uid in ipairs(dead_users) do
        redis.call('ZREM', queue_key, uid)
        redis.call('ZREM', heartbeat_key, uid)
    end
end

-- 3. Check available slots in active set
local active_count = redis.call('ZCARD', active_key)
local available_slots = max_active - active_count

if available_slots <= 0 then
    return 0
end

-- 4. Get the next N users from queue (lowest scores = top of the queue)
local next_users = redis.call('ZRANGE', queue_key, 0, available_slots - 1)

local promoted = 0
for _, user_id in ipairs(next_users) do
    redis.call('ZREM', queue_key, user_id)
    redis.call('ZREM', heartbeat_key, user_id)
    redis.call('ZADD', active_key, now + token_ttl, user_id)
    promoted = promoted + 1
end

if promoted > 0 then
    redis.call('EXPIRE', active_key, 3600)
end

return promoted
