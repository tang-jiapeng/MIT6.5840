package kvsrv

import (
	"sync"

	"6.5840/labrpc"
)
import "crypto/rand"
import "math/big"

type Clerk struct {
	server *labrpc.ClientEnd
	// You will have to modify this struct.
	ClientId int64 // 客户端唯一标识
	Seq      int64 // 操作序列号
	mu       sync.Mutex
}

func nrand() int64 {
	max := big.NewInt(int64(1) << 62)
	bigx, _ := rand.Int(rand.Reader, max)
	x := bigx.Int64()
	return x
}

func MakeClerk(server *labrpc.ClientEnd) *Clerk {
	ck := new(Clerk)
	ck.server = server
	// You'll have to add code here.
	ck.ClientId = nrand() // 分配唯一客户端 ID
	ck.Seq = 0            // 初始化序列号
	return ck
}

// fetch the current value for a key.
// returns "" if the key does not exist.
// keeps trying forever in the face of all other errors.
//
// you can send an RPC with code like this:
// ok := ck.server.Call("KVServer.Get", &args, &reply)
//
// the types of args and reply (including whether they are pointers)
// must match the declared types of the RPC handler function's
// arguments. and reply must be passed as a pointer.
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

// shared by Put and Append.
//
// you can send an RPC with code like this:
// ok := ck.server.Call("KVServer."+op, &args, &reply)
//
// the types of args and reply (including whether they are pointers)
// must match the declared types of the RPC handler function's
// arguments. and reply must be passed as a pointer.
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

func (ck *Clerk) Put(key string, value string) {
	ck.PutAppend(key, value, "Put")
}

// Append value to key's value and return that value
func (ck *Clerk) Append(key string, value string) string {
	return ck.PutAppend(key, value, "Append")
}
