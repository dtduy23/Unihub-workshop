-- Redis Lua script to decrement seats with high performance
-- KEYS[1]: workshop_seats_key (e.g., workshop:seats:ID)
-- ARGV[1]: amount to decrement (usually 1)

local current = redis.call("GET", KEYS[1])
if not current then
    return -1 -- Workshop not cached yet
end

current = tonumber(current)
local decrement = tonumber(ARGV[1])

if current >= decrement then
    local new_val = redis.call("DECRBY", KEYS[1], decrement)
    return new_val
else
    return -2 -- Out of seats
end
