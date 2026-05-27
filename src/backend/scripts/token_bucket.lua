-- Token Bucket Rate Limiting Lua Script
-- KEYS[1] = rate limit key (e.g., "ratelimit:{user_id}")
-- ARGV[1] = capacity (max tokens)
-- ARGV[2] = refill_rate (tokens per second)
-- ARGV[3] = ttl (key expiration in seconds)
-- ARGV[4] = current timestamp (seconds, float)

local key = KEYS[1]
local capacity = tonumber(ARGV[1])
local refill_rate = tonumber(ARGV[2])
local ttl = tonumber(ARGV[3])
local now = tonumber(ARGV[4])

-- Get current bucket state
local bucket = redis.call('HMGET', key, 'tokens', 'last_refill')
local tokens = tonumber(bucket[1])
local last_refill = tonumber(bucket[2])

-- Initialize bucket if it doesn't exist
if tokens == nil then
    tokens = capacity
    last_refill = now
end

-- Calculate token refill
local elapsed = now - last_refill
local new_tokens = elapsed * refill_rate
tokens = math.min(capacity, tokens + new_tokens)

-- Try to consume a token
if tokens >= 1 then
    tokens = tokens - 1
    redis.call('HSET', key, 'tokens', tokens, 'last_refill', now)
    redis.call('EXPIRE', key, ttl)
    return {1, math.floor(tokens)} -- allowed, remaining tokens
else
    redis.call('HSET', key, 'last_refill', now)
    redis.call('EXPIRE', key, ttl)
    -- Calculate retry-after: time until next token is available
    local retry_after = math.ceil((1 - tokens) / refill_rate)
    return {0, retry_after} -- denied, retry after seconds
end
