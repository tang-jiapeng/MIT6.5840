package kvsrv

import (
	"log"
	"sync"
)

const Debug = false

func DPrintf(format string, a ...interface{}) (n int, err error) {
	if Debug {
		log.Printf(format, a...)
	}
	return
}

// 操作记录，用于存储客户端的最新操作
type OpRecord struct {
	Seq      int64  // 客户端序列号
	OldValue string // Append 操作的旧值
}

type KVServer struct {
	mu sync.Mutex

	// Your definitions here.
	kvStore     map[string]string
	clientCache map[int64]OpRecord // 客户端操作记录，用于重复检测
}

func (kv *KVServer) Get(args *GetArgs, reply *GetReply) {
	// Your code here.
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

func (kv *KVServer) Put(args *PutAppendArgs, reply *PutAppendReply) {
	// Your code here.
	kv.mu.Lock()
	defer kv.mu.Unlock()

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
	// Your code here.
	kv.mu.Lock()
	defer kv.mu.Unlock()

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

func StartKVServer() *KVServer {
	kv := new(KVServer)

	// You may need initialization code here.
	kv.kvStore = make(map[string]string)
	kv.clientCache = make(map[int64]OpRecord)

	return kv
}
