# 6.5840 实验3：Raft

## Introduction
这是构建一个容错键/值存储系统的一系列实验中的第一个。在本实验中，你将实现 Raft，一个复制状态机协议。在下一个实验中，你将在 Raft 之上构建一个键/值服务。然后，你将通过多个复制状态机对服务进行“分片”以获得更高的性能。

复制服务通过在多个副本服务器上存储其状态（即数据）的完整副本来实现容错。复制允许服务在某些服务器发生故障（崩溃或网络断开或不稳定）时继续运行。挑战在于，故障可能导致副本持有的数据副本不一致。

Raft 将客户端请求组织成一个序列，称为日志，并确保所有副本服务器看到相同的日志。每个副本按照日志顺序执行客户端请求，并将其应用于本地服务状态副本。由于所有存活的副本看到相同的日志内容，它们以相同的顺序执行相同的请求，因此保持相同的服务状态。如果一个服务器失败但随后恢复，Raft 会负责将其日志更新到最新状态。只要大多数服务器存活并能够相互通信，Raft 就能继续运行。如果没有这样的多数，Raft 将暂停进展，但在多数服务器能够再次通信时会从中断处继续。

在本实验中，你将实现 Raft 作为一个 Go 对象类型，带有相关方法，旨在作为较大服务的一个模块。一组 Raft 实例通过 RPC 相互通信以维护复制日志。你的 Raft 接口将支持一个无限序列的编号命令，也称为日志条目。这些条目使用索引号进行编号。给定索引的日志条目最终会被提交。此时，你的 Raft 应将日志条目发送到更大的服务以供其执行。

你应遵循扩展 Raft 论文中的设计，特别关注图2。你将实现论文中的大部分内容，包括保存持久状态并在节点失败后重启时读取它。你不需要实现集群成员变更（第6节）。

本实验分为四个部分。你必须在相应的截止日期提交每个部分。

## Getting Started
如果你已经完成了实验1，你已经有了实验源代码的副本。如果没有，你可以在实验1的说明中找到通过 git 获取源代码的指导。

我们为你提供了骨架代码 `src/raft/raft.go`。我们还提供了一组测试，你应该用这些测试来推动你的实现工作，我们也将使用这些测试来评分你提交的实验。测试代码在 `src/raft/test_test.go` 中。

在评分你的提交时，我们将不使用 `-race` 标志运行测试。然而，你应该在开发过程中使用 `-race` 标志运行测试，以确保你的代码没有竞争条件。

要开始，请执行以下命令。不要忘记 `git pull` 以获取最新的软件。

```bash
$ cd ~/6.5840
$ git pull
...
$ cd src/raft
$ go test
Test (3A): initial election ...
--- FAIL: TestInitialElection3A (5.04s)
        config.go:326: expected one leader, got none
Test (3A): election after network failure ...
--- FAIL: TestReElection3A (5.03s)
        config.go:326: expected one leader, got none
...
$
```

## The code
在 `raft/raft.go` 中添加代码来实现 Raft。在该文件中，你会找到骨架代码，以及如何发送和接收 RPC 的示例。

你的实现必须支持以下接口，测试器和（最终）你的键/值服务器将使用该接口。你可以在 `raft.go` 的注释中找到更多细节。

```go
// 创建一个新的 Raft 服务器实例：
rf := Make(peers, me, persister, applyCh)

// 开始对新日志条目达成一致：
rf.Start(command interface{}) (index, term, isleader)

// 查询 Raft 的当前任期，以及它是否认为自己是领导者
rf.GetState() (term, isLeader)

// 每次有新日志条目提交到日志时，每个 Raft 节点
// 应通过 applyCh 向服务（或测试器）发送一个 ApplyMsg。
type ApplyMsg
```

服务通过调用 `Make(peers, me, …)` 来创建一个 Raft 节点。`peers` 参数是一个包含 Raft 节点（包括本节点）的网络标识符数组，供 RPC 使用。`me` 参数是本节点在 `peers` 数组中的索引。`Start(command)` 请求 Raft 开始处理以将命令追加到复制日志。`Start()` 应立即返回，不等待日志追加完成。服务期望你的实现通过 `Make()` 的 `applyCh` 通道参数为每个新提交的日志条目发送一个 `ApplyMsg`。

`raft.go` 包含发送 RPC（`sendRequestVote()`）和处理传入 RPC（`RequestVote()`）的示例代码。你的 Raft 节点应使用 `labrpc` Go 包（源代码在 `src/labrpc` 中）交换 RPC。测试器可以指示 `labrpc` 延迟、重新排序或丢弃 RPC，以模拟各种网络故障。虽然你可以临时修改 `labrpc`，但确保你的 Raft 在原始 `labrpc` 上正常工作，因为我们将使用它来测试和评分你的实验。你的 Raft 实例只能通过 RPC 交互；例如，不允许使用共享 Go 变量或文件进行通信。

后续实验建立在本实验之上，因此留出足够时间编写稳健的代码非常重要。

## Part 3A: leader election (moderate)

**Task**
>实现 Raft 的领导者选举和心跳（不带日志条目的 AppendEntries RPC）。第3A部分的目标是选出一个单一领导者，在无故障情况下保持该领导者地位，如果旧领导者失败或与旧领导者的数据包丢失，则选出新领导者。运行 `go test -run 3A` 测试你的 3A 代码。

### Hits

- 你无法直接运行 Raft 实现；你应通过测试器运行，即 `go test -run 3A`。

- 遵循论文的图2。此时你需要关注发送和接收 `RequestVote` RPC、与选举相关的服务器规则，以及与领导者选举相关的状态。

- 在 `raft.go` 的 Raft 结构体中添加图2中与领导者选举相关的状态。你还需要定义一个结构体来保存每个日志条目的信息。

- 填写 `RequestVoteArgs` 和 `RequestVoteReply` 结构体。修改 `Make()` 以创建一个后台 goroutine，定期通过发送 `RequestVote` RPC 启动领导者选举，当一段时间未收到其他节点的消息时触发。实现 `RequestVote()` RPC 处理程序，使服务器相互投票。

- 为实现心跳，定义一个 `AppendEntries` RPC 结构体（尽管你可能尚未需要所有参数），并让领导者定期发送。编写一个 `AppendEntries` RPC 处理方法。

- 测试器要求领导者每秒发送心跳不超过十次。

- 测试器要求你的 Raft 在旧领导者失败后五秒内（如果多数节点仍可通信）选出新领导者。

- 论文的第5.2节提到选举超时在 150 到 300 毫秒范围内。这样的范围只有在领导者发送心跳远超每 150 毫秒一次（例如每 10 毫秒一次）时才有意义。由于测试器限制每秒最多十次心跳，你需要使用比论文建议的 150 到 300 毫秒更大的选举超时，但不能太大，否则可能无法在五秒内选出领导者。

- 你可能会发现 Go 的 `rand` 包很有用。

- 你需要编写定期或延迟后执行操作的代码。最简单的方法是创建一个带有循环的 goroutine，调用 `time.Sleep()`；参见 `Make()` 创建的 `ticker()` goroutine。不要使用 Go 的 `time.Timer` 或 `time.Ticker`，它们难以正确使用。

- 如果你的代码无法通过测试，请再次阅读论文的图2；领导者选举的完整逻辑分布在图的多个部分。

- 不要忘记实现 `GetState()`。

- 测试器会在永久关闭一个实例时调用你的 Raft 的 `rf.Kill()`。你可以使用 `rf.killed()` 检查是否调用了 `Kill()`。你可能希望在所有循环中进行此检查，以避免死掉的 Raft 实例打印混乱的消息。

- Go RPC 仅发送字段名以大写字母开头的结构体。子结构体的字段名也必须大写（例如，日志记录数组中的字段）。`labgob` 包会对此发出警告；不要忽略这些警告。

- 本实验最具挑战性的部分可能是调试。花些时间使你的实现易于调试。参考指导页面获取调试提示。

在提交第3A部分之前，确保通过 3A 测试，你应该看到类似以下输出：

```bash
$ go test -run 3A
Test (3A): initial election ...
  ... Passed --   3.5  3   58   16840    0
Test (3A): election after network failure ...
  ... Passed --   5.4  3  118   25269    0
Test (3A): multiple elections ...
  ... Passed --   7.3  7  624  138014    0
PASS
ok  	6.5840/raft	16.265s
$
```

每次“Passed”行包含五个数字：测试耗时（秒）、Raft 节点数、测试期间发送的 RPC 数量、RPC 消息的总字节数，以及 Raft 报告提交的日志条目数。你的数字可能与此处显示的不同。你可以忽略这些数字，但它们可能有助于你检查实现发送的 RPC 数量。对于实验3、4和5，如果所有测试（`go test`）超过600秒，或任何单个测试超过120秒，分数脚本将使你的解决方案失败。

在评分你的提交时，我们将不使用 `-race` 标志运行测试。然而，你应确保你的代码在使用 `-race` 标志时始终通过测试。

## Part 3B: log (hard)

**Task**
> 实现领导者和跟随者的代码以追加新日志条目，使 `go test -run 3B` 测试通过。

### Hits

- 运行 `git pull` 获取最新的实验软件。

- 你的首要目标是通过 `TestBasicAgree3B()`。首先实现 `Start()`，然后编写通过 `AppendEntries` RPC 发送和接收新日志条目的代码，遵循图2。在每个节点上通过 `applyCh` 发送每个新提交的条目。

- 你需要实现选举限制（论文第5.4.1节）。

- 你的代码可能有重复检查某些事件的循环。不要让这些循环无暂停地连续执行，因为这会使你的实现变慢以至于无法通过测试。使用 Go 的条件变量，或在每次循环迭代中插入 `time.Sleep(10 * time.Millisecond)`。

- 为了后续实验的顺利进行，编写（或重写）清晰整洁的代码。重新访问指导页面，获取开发和调试代码的提示。

- 如果测试失败，请查看 `test_test.go` 和 `config.go` 以了解测试内容。`config.go` 还展示了测试器如何使用 Raft API。

后续实验的测试可能会因为你的代码运行太慢而失败。你可以使用 `time` 命令检查解决方案的实时和 CPU 时间消耗。以下是典型输出：

```bash
$ time go test -run 3B
Test (3B): basic agreement ...
  ... Passed --   0.9  3   16    4572    3
Test (3B): RPC byte count ...
  ... Passed --   1.7  3   48  114536   11
Test (3B): agreement after follower reconnects ...
  ... Passed --   3.6  3   78   22131    7
Test (3B): no agreement if too many followers disconnect ...
  ... Passed --   3.8  5  172   40935    3
Test (3B): concurrent Start()s ...
  ... Passed --   1.1  3   24    7379    6
Test (3B): rejoin of partitioned leader ...
  ... Passed --   5.1  3  152   37021    4
Test (3B): leader backs up quickly over incorrect follower logs ...
  ... Passed --  17.2  5 2080 1587388  102
Test (3B): RPC counts aren't too high ...
  ... Passed --   2.2  3   60   20119   12
PASS
ok  	6.5840/raft	35.557s

real	0m35.899s
user	0m2.556s
sys	0m1.458s
$
```

“ok 6.5840/raft 35.557s”表示 Go 测量 3B 测试的实时（壁钟时间）为 35.557 秒。“user 0m2.556s”表示代码消耗了 2.556 秒的 CPU 时间，即实际执行指令的时间（而不是等待或休眠）。如果你的解决方案在 3B 测试中使用的实时远超一分钟，或 CPU 时间远超 5 秒，可能会在后续遇到问题。检查休眠或等待 RPC 超时的时间、未休眠或等待条件或通道消息的循环，或发送的大量 RPC。

## Part 3C: persistence (hard)
如果基于 Raft 的服务器重启，它应从中断处恢复服务。这要求 Raft 保持持久状态以在重启后存活。论文的图2提到哪些状态应持久化。

实际实现会将 Raft 的持久状态在每次更改时写入磁盘，并在重启后从磁盘读取状态。你的实现不会使用磁盘；相反，它将从 `Persister` 对象（参见 `persister.go`）保存和恢复持久状态。调用 `Raft.Make()` 的人会提供一个 `Persister`，它最初持有 Raft 最近持久化的状态（如果有）。Raft 应从该 `Persister` 初始化其状态，并每次状态更改时使用它保存持久状态。使用 `Persister` 的 `ReadRaftState()` 和 `Save()` 方法。

**Task**
> 在 `raft.go` 中完成 `persist()` 和 `readPersist()` 函数，添加保存和恢复持久状态的代码。你需要将状态编码（或“序列化”）为字节数组以传递给 `Persister`。使用 `labgob` 编码器；参见 `persist()` 和 `readPersist()` 的注释。`labgob` 类似于 Go 的 `gob` 编码器，但如果尝试编码小写字段名的结构体，会打印错误消息。目前，将 `nil` 作为第二个参数传递给 `persister.Save()`。在你的实现更改持久状态的点插入 `persist()` 调用。如果其他部分实现正确，你应通过所有 3C 测试。

你可能需要优化以一次回退 `nextIndex` 多个条目。查看扩展 Raft 论文从第7页底部到第8页顶部（标有灰线）。论文对细节描述模糊，你需要填补空白。一种可能的方法是让拒绝消息包括：
```bash
    XTerm:  冲突条目（如果有）的任期
    XIndex: 具有该任期的第一个条目的索引（如果有）
    XLen:   日志长度
```
然后领导者的逻辑可以是：
```bash
  情况1：领导者没有 XTerm：
    nextIndex = XIndex
  情况2：领导者有 XTerm：
    nextIndex = 领导者该任期的最后一个条目
  情况3：跟随者日志太短：
    nextIndex = XLen
```

其他hits：
- 运行 `git pull` 获取最新的实验软件。
- 3C 测试比 3A 或 3B 更严格，失败可能由 3A 或 3B 代码中的问题引起。

你的代码应通过所有 3C 测试（如下所示），以及 3A 和 3B 测试。

```bash
$ go test -run 3C
Test (3C): basic persistence ...
  ... Passed --   5.0  3   86   22849    6
Test (3C): more persistence ...
  ... Passed --  17.6  5  952  218854   16
Test (3C): partitioned leader and one follower crash, leader restarts ...
  ... Passed --   2.0  3   34    8937    4
Test (3C): Figure 8 ...
  ... Passed --  31.2  5  580  130675   32
Test (3C): unreliable agreement ...
  ... Passed --   1.7  5 1044  366392  246
Test (3C): Figure 8 (unreliable) ...
  ... Passed --  33.6  5 10700 33695245  308
Test (3C): churn ...
  ... Passed --  16.1  5 8864 44771259 1544
Test (3C): unreliable churn ...
  ... Passed --  16.5  5 4220 6414632  906
PASS
ok  	6.5840/raft	123.564s
$
```

在提交之前多次运行测试并检查每次运行都打印 PASS 是明智的。

```bash
$ for i in {0..10}; do go test; done
```

## Part 3D: log compaction (hard)
目前，如果服务器重启，它需要重放完整的 Raft 日志以恢复状态。然而，对于长期运行的服务来说，永远记住完整的 Raft 日志是不实际的。相反，你将修改 Raft 以与定期持久存储其状态快照的服务协作，此时 Raft 会丢弃快照之前的日志条目。结果是更少的持久数据和更快的重启。然而，现在跟随者可能落后太多，以至于领导者已丢弃其需要的日志条目；领导者必须发送快照以及快照时间点开始的日志。扩展 Raft 论文的第7节概述了该方案；你需要设计细节。

你的 Raft 必须提供以下函数，供服务调用以传递其状态的序列化快照：

```go
Snapshot(index int, snapshot []byte)
```

在实验 3D 中，测试器会定期调用 `Snapshot()`。在实验4中，你将编写一个调用 `Snapshot()` 的键/值服务器；快照将包含完整的键/值对表。服务层在每个节点（不仅是领导者）上调用 `Snapshot()`。

`index` 参数表示快照中反映的最高日志条目。Raft 应丢弃该点之前的日志条目。你需要修改 Raft 代码以仅存储日志的尾部。

你需要实现论文中讨论的 `InstallSnapshot` RPC，允许 Raft 领导者通知落后的 Raft 节点用快照替换其状态。你可能需要考虑 `InstallSnapshot` 如何与图2中的状态和规则交互。

当跟随者的 Raft 代码接收到 `InstallSnapshot` RPC 时，它可以通过 `applyCh` 将快照发送到服务，使用 `ApplyMsg`。`ApplyMsg` 结构体定义已包含测试器期望的字段。注意这些快照只能推进服务状态，不能使其倒退。

如果服务器崩溃，它必须从持久数据重启。你的 Raft 应持久保存 Raft 状态和对应的快照。使用 `persister.Save()` 的第二个参数保存快照。如果没有快照，传递 `nil` 作为第二个参数。

当服务器重启时，应用层读取持久化的快照并恢复保存的状态。

**Task**
> 实现 `Snapshot()` 和 `InstallSnapshot` RPC，以及支持这些功能所需的 Raft 更改（例如，使用修剪日志操作）。当通过 3D 测试（以及所有之前的实验3测试）时，你的解决方案完成。

### Hits
- 运行 `git pull` 确保获取最新软件。

- 一个好的起点是修改代码，使其能够仅存储从某个索引 X 开始的日志部分。最初可将 X 设置为零并运行 3B/3C 测试。然后使 `Snapshot(index)` 丢弃索引之前的日志，并将 X 设置为索引。如果一切顺利，你应通过第一个 3D 测试。

- 接下来：如果领导者没有足够的日志条目来更新跟随者，让领导者发送 `InstallSnapshot` RPC。

- 在单个 `InstallSnapshot` RPC 中发送整个快照。不要实现图13的偏移机制来分割快照。

- Raft 必须以允许 Go 垃圾回收器释放和重用内存的方式丢弃旧日志条目；这要求没有可达的引用（指针）指向丢弃的日志条目。

- 实验3（3A+3B+3C+3D）完整测试的合理耗时（不带 `-race`）为实时6分钟和 CPU 时间1分钟。使用 `-race` 时，大约为实时10分钟和 CPU 时间2分钟。

你的代码应通过所有 3D 测试（如下所示），以及 3A、3B 和 3C 测试。

```bash
$ go test -run 3D
Test (3D): snapshots basic ...
  ... Passed --  11.6  3  176   61716  192
Test (3D): install snapshots (disconnect) ...
  ... Passed --  64.2  3  878  320610  336
Test (3D): install snapshots (disconnect+unreliable) ...
  ... Passed --  81.1  3 1059  375850  341
Test (3D): install snapshots (crash) ...
  ... Passed --  53.5  3  601  256638  339
Test (3D): install snapshots (unreliable+crash) ...
  ... Passed --  63.5  3  687  288294  336
Test (3D): crash and restart all servers ...
  ... Passed --  19.5  3  268   81352   58
PASS
ok      6.5840/raft      293.456s
```