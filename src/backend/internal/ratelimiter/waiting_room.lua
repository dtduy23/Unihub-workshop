-- Virtual Waiting Room Queue Management
-- Uses Redis Sorted Set for FIFO ordering
--
-- KEYS[1] = queue key (e.g., "waitingroom:{workshop_id}")
-- KEYS[2] = active tokens key (e.g., "waitingroom:active:{workshop_id}")
-- ARGV[1] = user_id
-- ARGV[2] = current timestamp (seconds, float)
-- ARGV[3] = max concurrent users allowed through (batch size)
-- ARGV[4] = token TTL (seconds) - how long a user has to complete action
-- ARGV[5] = queue TTL (seconds) - when to auto-expire the queue
--
-- Returns: {status, position_or_ttl, total_in_queue}
--   status = 1: user granted access (proceed to register)
--   status = 2: user queued (wait in line)
--   status = 3: user already has active token (proceed)
--   status = 0: user already in queue (return current position)

local queue_key = KEYS[1]
local active_key = KEYS[2]
local user_id = ARGV[1]
local now = tonumber(ARGV[2])
local max_active = tonumber(ARGV[3])
local token_ttl = tonumber(ARGV[4])
local queue_ttl = tonumber(ARGV[5])

-- Check if user already has an active access token
local has_token = redis.call('SISMEMBER', active_key, user_id)
if has_token == 1 then
    local remaining = redis.call('TTL', active_key)
    return {3, remaining, 0} -- Already has access
end

-- Check if user is already in the queue
local existing_score = redis.call('ZSCORE', queue_key, user_id)
if existing_score then
    -- Return current position (1-indexed)
    local position = redis.call('ZRANK', queue_key, user_id)
    local total = redis.call('ZCARD', queue_key)
    return {0, position + 1, total}
end

-- Count currently active users
local active_count = redis.call('SCARD', active_key)

-- If there's room, grant immediate access
if active_count < max_active then
    redis.call('SADD', active_key, user_id)
    redis.call('EXPIRE', active_key, token_ttl)
    return {1, token_ttl, 0} -- Granted access
end

-- Otherwise, add to waiting queue with timestamp as score (FIFO)
redis.call('ZADD', queue_key, now, user_id)
redis.call('EXPIRE', queue_key, queue_ttl)

local position = redis.call('ZRANK', queue_key, user_id)
local total = redis.call('ZCARD', queue_key)

return {2, position + 1, total} -- Queued at position
