package kvsrv

// Put or Append
type PutAppendArgs struct {
	Key   string
	Value string
	// You'll have to add definitions here.
	// Field names must start with capital letters,
	// otherwise RPC will break.
	Op       string
	ClientId int64
	Seq      int64
}

type PutAppendReply struct {
	Value       string
	WrongLeader bool
}

type GetArgs struct {
	Key string
	// You'll have to add definitions here.
	ClientId int64
	Seq      int64
}

type GetReply struct {
	Value       string
	WrongLeader bool
}
