
local key = KEYS[1]
-- 窗口期大小
local window = tonumber(ARGV[1])
-- 请求阈值
local threshold = tonumber(ARGV[2])

--当前时间
local now = tonumber(ARGV[3])

local small = now - window

--先把所有过期的数据都删了再统计一下
redis.call("ZREMRANGEBYSCORE",key,'-inf',small)


--统计一下 现在总共处理了多少次
local count = redis.call("ZCOUNT",key,'-inf','+inf')
if  count >=threshold then
    --限流
    return "true"
else
    redis.call("ZADD",key,now,now)
    redis.call('PEXPIRE', key, window)
    return "false"
end

