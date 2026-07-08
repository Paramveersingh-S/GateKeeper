package ratelimit

import "github.com/redis/go-redis/v9"

// TokenBucketScript implements the token bucket algorithm atomically.
// KEYS[1] = bucket key
// ARGV[1] = bucket capacity (max tokens)
// ARGV[2] = refill rate (tokens per second)
// ARGV[3] = requested tokens
// ARGV[4] = current timestamp in seconds
var TokenBucketScript = redis.NewScript(`
local key = KEYS[1]
local capacity = tonumber(ARGV[1])
local rate = tonumber(ARGV[2])
local requested = tonumber(ARGV[3])
local now = tonumber(ARGV[4])

local bucket = redis.call("HMGET", key, "tokens", "last_update")
local tokens = tonumber(bucket[1])
local last_update = tonumber(bucket[2])

if not tokens then
    tokens = capacity
    last_update = now
else
    local elapsed = now - last_update
    tokens = math.min(capacity, tokens + (elapsed * rate))
end

if tokens >= requested then
    tokens = tokens - requested
    redis.call("HMSET", key, "tokens", tokens, "last_update", now)
    redis.call("EXPIRE", key, math.ceil(capacity / rate) + 1)
    return {1, tokens}
else
    return {0, tokens}
end
`)

// FixedWindowScript implements the fixed window counter.
// KEYS[1] = window key
// ARGV[1] = limit
// ARGV[2] = requested
// ARGV[3] = window size in seconds
var FixedWindowScript = redis.NewScript(`
local key = KEYS[1]
local limit = tonumber(ARGV[1])
local requested = tonumber(ARGV[2])
local window = tonumber(ARGV[3])

local current = tonumber(redis.call("GET", key) or "0")
if current + requested <= limit then
    redis.call("INCRBY", key, requested)
    if current == 0 then
        redis.call("EXPIRE", key, window)
    end
    return {1, limit - (current + requested)}
else
    return {0, limit - current}
end
`)

// SlidingWindowLogScript implements the exact sliding window log.
// KEYS[1] = sorted set key
// ARGV[1] = limit
// ARGV[2] = requested (usually 1, but we can add N elements with same score)
// ARGV[3] = current timestamp (ms)
// ARGV[4] = window start timestamp (ms)
// ARGV[5] = window size in seconds
var SlidingWindowLogScript = redis.NewScript(`
local key = KEYS[1]
local limit = tonumber(ARGV[1])
local requested = tonumber(ARGV[2])
local now = tonumber(ARGV[3])
local window_start = tonumber(ARGV[4])
local window_size = tonumber(ARGV[5])

-- Remove old entries
redis.call("ZREMRANGEBYSCORE", key, "-inf", window_start)

local current_count = redis.call("ZCARD", key)

if current_count + requested <= limit then
    for i=1,requested do
        redis.call("ZADD", key, now, now .. "-" .. i)
    end
    redis.call("EXPIRE", key, window_size)
    return {1, limit - (current_count + requested)}
else
    return {0, limit - current_count}
end
`)

// SlidingWindowCounterScript implements approximate sliding window with 2 windows.
// KEYS[1] = current window key
// KEYS[2] = previous window key
// ARGV[1] = limit
// ARGV[2] = requested
// ARGV[3] = weight of previous window (0.0 to 1.0)
// ARGV[4] = window size in seconds
var SlidingWindowCounterScript = redis.NewScript(`
local curr_key = KEYS[1]
local prev_key = KEYS[2]
local limit = tonumber(ARGV[1])
local requested = tonumber(ARGV[2])
local weight = tonumber(ARGV[3])
local window_size = tonumber(ARGV[4])

local curr_count = tonumber(redis.call("GET", curr_key) or "0")
local prev_count = tonumber(redis.call("GET", prev_key) or "0")

local estimated_count = curr_count + math.floor(prev_count * weight)

if estimated_count + requested <= limit then
    redis.call("INCRBY", curr_key, requested)
    if curr_count == 0 then
        redis.call("EXPIRE", curr_key, window_size * 2)
    end
    return {1, limit - (estimated_count + requested)}
else
    return {0, math.max(0, limit - estimated_count)}
end
`)

// ReserveReconcileScript handles token reservation and reconciliation.
// Operation mode: "reserve" or "reconcile"
var ReserveReconcileScript = redis.NewScript(`
local key = KEYS[1]
local mode = ARGV[1]

if mode == "reserve" then
    local capacity = tonumber(ARGV[2])
    local rate = tonumber(ARGV[3])
    local requested = tonumber(ARGV[4])
    local now = tonumber(ARGV[5])
    
    local bucket = redis.call("HMGET", key, "tokens", "last_update")
    local tokens = tonumber(bucket[1])
    local last_update = tonumber(bucket[2])
    
    if not tokens then
        tokens = capacity
        last_update = now
    else
        local elapsed = now - last_update
        tokens = math.min(capacity, tokens + (elapsed * rate))
    end
    
    if tokens >= requested then
        tokens = tokens - requested
        redis.call("HMSET", key, "tokens", tokens, "last_update", now)
        redis.call("EXPIRE", key, math.ceil(capacity / rate) + 1)
        return {1, tokens}
    else
        return {0, tokens}
    end
elseif mode == "reconcile" then
    local diff = tonumber(ARGV[2]) -- difference to add back (estimated - actual)
    if diff > 0 then
        -- Add tokens back
        local bucket = redis.call("HMGET", key, "tokens")
        local tokens = tonumber(bucket[1])
        if tokens then
            tokens = tokens + diff
            redis.call("HMSET", key, "tokens", tokens)
        end
    end
    return {1, 0}
end
`)
