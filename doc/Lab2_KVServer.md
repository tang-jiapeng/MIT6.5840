#  Lab2设计与实现

Lab2的要求是实现能处理消息丢失的KV server，在基础的kv server实现下添加重试逻辑和重复检测机制即可

## Client实现
在 `client.go` 中，Clerk 的 Get 和 PutAppend 方法通过无限循环实现重试，确保在未收到回复（RPC 调用失败或超时）时继续尝试，直到成功，在Clerk结构中定义`ClientId`和`Seq`字段组合唯一标识每个操作，来确保服务器能识别重复请求

```go
type Clerk struct {  
    server *labrpc.ClientEnd  
    ClientId int64 // 客户端唯一标识  
    Seq      int64 // 操作序列号  
    mu       sync.Mutex  
}

func (ck *Clerk) Get(key string) string {  
    ck.mu.Lock()  
    ck.Seq++  
    seq := ck.Seq  
    clientId := ck.ClientId  
    ck.mu.Unlock()  
  
    args := GetArgs{  
       Key:      key,  
       ClientId: clientId,  
       Seq:      seq,  
    }  
    reply := GetReply{}  
    for {  
       ok := ck.server.Call("KVServer.Get", &args, &reply)  
       if ok && !reply.WrongLeader {  
          return reply.Value  
       }  
       // 重试  
    }  
}

func (ck *Clerk) PutAppend(key string, value string, op string) string {  
    ck.mu.Lock()  
    ck.Seq++  
    seq := ck.Seq  
    clientId := ck.ClientId  
    ck.mu.Unlock()  
  
    args := PutAppendArgs{  
       Key:      key,  
       Value:    value,  
       Op:       op,  
       ClientId: clientId,  
       Seq:      seq,  
    }  
    reply := PutAppendReply{}  
  
    for {  
       ok := ck.server.Call("KVServer."+op, &args, &reply)  
       if ok && !reply.WrongLeader {  
          return reply.Value  
       }  
       // 重试  
    }  
}
```
## Server实现
在 `server.go` 中，KVServer 通过维护 clientCache（映射 ClientId 到 OpRecord）过滤重复的 Put 和 Append 请求，确保每个操作只执行一次。Get 是幂等操作，无需重复检测。

为实现重复检测机制，在KVServer中定义一个clientCache字段，clientCache 是一个 `map[int64]OpRecord`，以 ClientId 为键，存储每个客户端的最新操作记录。
**OpRecord结构**包含：Seq：客户端发送的序列号；OldValue：Append 操作的旧值（Put 的旧值为空）
```go
// 操作记录，用于存储客户端的最新操作
type OpRecord struct {
	Seq      int64  // 客户端序列号
	OldValue string // Append 操作的旧值
}

type KVServer struct {
	mu sync.Mutex

	kvStore     map[string]string
	clientCache map[int64]OpRecord // 客户端操作记录，用于重复检测
}
```

重复检测的逻辑如下：
- 在 Put 和 Append 中，检查 clientCache 是否包含当前 ClientId 的记录，且记录的 Seq 是否大于或等于请求的 args.Seq
- 如果是重复请求（record.Seq >= args.Seq）
	- 对于 Put，直接返回（reply.OldValue 为空）
	- 对于 Append，返回缓存的 OldValue（确保返回与首次执行相同的旧值）
	- 不修改 kvStore，避免重复执行
- 如果不是重复请求，执行操作并更新 clientCache

```go
func (kv *KVServer) Put(args *PutAppendArgs, reply *PutAppendReply) {
	kv.mu.Lock()
	defer kv.mu.Unlock() 
	// 检查是否为重复请求
	if record, exists := kv.clientCache[args.ClientId]; exists && record.Seq >= args.Seq {
		reply.WrongLeader = false
		reply.Value = "" // Put 不返回旧值
		DPrintf("Put: duplicate request, key=%s, value=%s, clientId=%d, seq=%d", args.Key, args.Value, args.ClientId, args.Seq)
		return
	}
	// 执行 Put 操作
	kv.kvStore[args.Key] = args.Value
	reply.Value = ""
	reply.WrongLeader = false
	// 更新客户端缓存
	kv.clientCache[args.ClientId] = OpRecord{
		Seq:      args.Seq,
		OldValue: "",
	}
	DPrintf("Put: key=%s, value=%s, clientId=%d, seq=%d", args.Key, args.Value, args.ClientId, args.Seq)
}

func (kv *KVServer) Append(args *PutAppendArgs, reply *PutAppendReply) {
	kv.mu.Lock()
	defer kv.mu.Unlock() 
	// 检查是否为重复请求
	if record, exists := kv.clientCache[args.ClientId]; exists && record.Seq >= args.Seq {
		reply.Value = record.OldValue
		reply.WrongLeader = false
		DPrintf("Append: duplicate request, key=%s, value=%s, clientId=%d, seq=%d, oldValue=%s", args.Key, args.Value, args.ClientId, args.Seq, reply.Value)
		return
	}
	// 获取当前值（不存在则为空字符串）
	oldValue, exists := kv.kvStore[args.Key]
	if !exists {
		oldValue = ""
	}
	// 执行 Append 操作
	kv.kvStore[args.Key] = oldValue + args.Value
	reply.Value = oldValue
	reply.WrongLeader = false
	// 更新客户端缓存
	kv.clientCache[args.ClientId] = OpRecord{
		Seq:      args.Seq,
		OldValue: oldValue,
	}
	DPrintf("Append: key=%s, value=%s, clientId=%d, seq=%d, oldValue=%s", args.Key, args.Value, args.ClientId, args.Seq, oldValue)
}

```

对于Get的处理，由于Get是只读操作，重复执行不会改变状态，因此无需重复检测，直接从 kvStore 读取值并返回
```go
func (kv *KVServer) Get(args *GetArgs, reply *GetReply) {  
    kv.mu.Lock()  
    defer kv.mu.Unlock()  
  
    value, exists := kv.kvStore[args.Key]  
    if !exists {  
       value = ""  
    }  
    reply.Value = value  
    reply.WrongLeader = false  
  
    DPrintf("Get: key=%s, value=%s, clientId=%d, seq=%d", args.Key, reply.Value, args.ClientId, args.Seq) 
}
```