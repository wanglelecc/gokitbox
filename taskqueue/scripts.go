package taskqueue

const enqueueScript = `
if redis.call('HEXISTS',KEYS[3],ARGV[1])==1 then return 0 end
redis.call('HSET',KEYS[3],ARGV[1],ARGV[2]);redis.call('HSET',KEYS[7],ARGV[1],0)
if ARGV[4]=='1' then redis.call('RPUSH',KEYS[2],ARGV[1]) else redis.call('ZADD',KEYS[8],ARGV[3],ARGV[1]) end
return 1`
const claimScript = `
local clock=redis.call('TIME');local now=clock[1]*1000+math.floor(clock[2]/1000)
local previous=redis.call('HGET',KEYS[6],ARGV[1])
if previous then return {previous,redis.call('HGET',KEYS[3],previous),redis.call('HGET',KEYS[7],previous)} end
local expired=redis.call('ZRANGEBYSCORE',KEYS[4],'-inf',now,'LIMIT',0,100)
for _,id in ipairs(expired) do
 local owner=redis.call('HGET',KEYS[5],id);if owner then redis.call('HDEL',KEYS[6],owner) end
 redis.call('HDEL',KEYS[5],id);redis.call('ZREM',KEYS[4],id);redis.call('RPUSH',KEYS[2],id)
end
local scheduled=redis.call('ZRANGEBYSCORE',KEYS[8],'-inf',now,'LIMIT',0,100)
for _,id in ipairs(scheduled) do redis.call('ZREM',KEYS[8],id);redis.call('RPUSH',KEYS[2],id) end
local id=redis.call('LPOP',KEYS[2])
if not id then
 local payload
 if ARGV[3]=='list' then payload=redis.call('LPOP',KEYS[1]) else
 local due=redis.call('ZRANGE',KEYS[1],0,0,'WITHSCORES')
 if #due>0 and tonumber(due[2])*1000<=now then payload=due[1];redis.call('ZREM',KEYS[1],payload) end
 end
 if not payload then return false end
 id=ARGV[1];redis.call('HSET',KEYS[3],id,payload);redis.call('HSET',KEYS[7],id,0)
end
local value=redis.call('HGET',KEYS[3],id)
if not value then return redis.error_reply('TASK_DATA_MISSING') end
local attempt=redis.call('HINCRBY',KEYS[7],id,1)
redis.call('HSET',KEYS[5],id,ARGV[1]);redis.call('HSET',KEYS[6],ARGV[1],id);redis.call('ZADD',KEYS[4],now+tonumber(ARGV[2]),id)
return {id,value,attempt}`
const actionScript = `
local clock=redis.call('TIME');local now=clock[1]*1000+math.floor(clock[2]/1000)
if redis.call('HGET',KEYS[5],ARGV[1])~=ARGV[2] or tonumber(redis.call('ZSCORE',KEYS[4],ARGV[1]) or '0')<=now then return 0 end
if ARGV[3]=='renew' then redis.call('ZADD',KEYS[4],now+tonumber(ARGV[4]),ARGV[1]);return 1 end
local payload=redis.call('HGET',KEYS[3],ARGV[1])
if ARGV[3]=='fail' then
 local existing=redis.call('HGET',KEYS[12],ARGV[1])
 if existing and existing~=payload then return redis.error_reply('TASK_ID_CONFLICT') end
 if not existing then redis.call('HSET',KEYS[12],ARGV[1],payload);redis.call('HSET',KEYS[16],ARGV[1],0);redis.call('RPUSH',KEYS[11],ARGV[1]) end
elseif ARGV[3]=='retry' then redis.call('RPUSH',KEYS[2],ARGV[1])
elseif ARGV[3]=='dead' then redis.call('RPUSH',KEYS[9],ARGV[1])
end
redis.call('ZREM',KEYS[4],ARGV[1]);redis.call('HDEL',KEYS[5],ARGV[1]);redis.call('HDEL',KEYS[6],ARGV[2])
if ARGV[3]=='ack' or ARGV[3]=='fail' then redis.call('HDEL',KEYS[3],ARGV[1]);redis.call('HDEL',KEYS[7],ARGV[1]) end
return 1`
