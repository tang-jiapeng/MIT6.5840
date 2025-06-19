# 6.5840 实验 2：键/值服务器

## 引言

在本实验中，你将构建一个单机键/值服务器，确保即使在网络故障的情况下，每个操作都能被精确执行一次，并且操作是线性化的（linearizable）。后续实验将复制这样的服务器以处理服务器崩溃。

客户端可以通过三种不同的 RPC 向键/值服务器发送请求：`Put(key, value)`、`Append(key, arg)` 和 `Get(key)`。服务器维护一个内存中的键/值对映射，键和值均为字符串。`Put(key, value)` 为特定键安装或替换值，`Append(key, arg)` 将参数 `arg` 追加到键的值上并返回旧值，`Get(key)` 获取键的当前值。对于不存在的键，`Get` 应返回空字符串。对不存在的键执行 `Append` 应视为现有值是零长度字符串。每个客户端通过一个带有 `Put/Append/Get` 方法的 `Clerk` 与服务器交互，`Clerk` 管理与服务器的 RPC 交互。

你的服务器必须保证对 `Clerk` 的 `Get/Put/Append` 方法的应用调用是线性化的。如果客户端请求不是并发的，每个客户端的 `Get/Put/Append` 调用应观察到前序调用序列对状态的修改。对于并发调用，返回值和最终状态必须与操作按某种顺序逐一执行的结果相同。调用是并发的如果它们在时间上重叠：例如，客户端 X 调用 `Clerk.Put()`，客户端 Y 调用 `Clerk.Append()`，然后客户端 X 的调用返回。调用必须观察到所有在调用开始前已完成的操作的效果。

线性化（Linearizability）对应用程序来说很方便，因为它等同于单个服务器按顺序逐一处理请求的行为。例如，如果一个客户端收到服务器对更新请求的成功响应，后续从其他客户端发起的读取操作保证能看到该更新的效果。对于单个服务器，提供线性化相对容易。

## 开始

我们为你提供了 `src/kvsrv` 中的骨架代码和测试用例。你需要修改 `kvsrv/client.go`、`kvsrv/server.go` 和 `kvsrv/common.go`。

要开始，请执行以下命令。不要忘记执行 `git pull` 以获取最新软件。

```bash
$ cd ~/6.5840
$ git pull
...
$ cd src/kvsrv
$ go test
...
$
```

## 无网络故障的键/值服务器（简单）

你的第一个任务是实现一个在没有消息丢失的情况下工作的解决方案。

你需要在 `client.go` 中的 `Clerk` 的 `Put/Append/Get` 方法中添加发送 RPC 的代码，并在 `server.go` 中实现 `Put`、`Append()` 和 `Get()` 的 RPC 处理程序。

当你通过测试套件中的前两个测试：“one client”和“many clients”时，你已完成此任务。

使用 `go test -race` 检查你的代码是否无竞争（race-free）。

## 处理消息丢失的键/值服务器（简单）

现在你需要修改你的解决方案，使其能够在消息丢失（例如，RPC 请求和回复丢失）的情况下继续工作。如果消息丢失，客户端的 `ck.server.Call()` 将返回 `false`（更具体地说，`Call()` 会等待回复消息一个超时时间，如果在该时间内没有收到回复，则返回 `false`）。你将面临的一个问题是，`Clerk` 可能需要多次重试发送 RPC 直到成功。然而，每次 `Clerk.Put()` 或 `Clerk.Append()` 调用都应仅导致一次执行，因此你需要确保重试不会导致服务器重复执行请求。

在 `Clerk` 中添加代码以在未收到回复时重试，并在 `server.go` 中添加代码以过滤重复请求（如果操作需要）。这些说明中包含有关重复检测的指导。

你需要唯一标识客户端操作，以确保键/值服务器对每个操作只执行一次。

你需要仔细考虑服务器需要维护哪些状态来处理重复的 `Get()`、`Put()` 和 `Append()` 请求（如果需要的话）。

你的重复检测方案应尽快释放服务器内存，例如通过让每个 RPC 暗示客户端已看到其前一个 RPC 的回复。可以假设客户端一次只会调用一个 `Clerk`。

你的代码现在应该通过所有测试，如下所示：

```bash
$ go test
Test: one client ...
  ... Passed -- t  3.8 nrpc 31135 ops 31135
Test: many clients ...
  ... Passed -- t  4.7 nrpc 102853 ops 102853
Test: unreliable net, many clients ...
  ... Passed -- t  4.1 nrpc   580 ops  496
Test: concurrent append to same key, unreliable ...
  ... Passed -- t  0.6 nrpc    61 ops   52
Test: memory use get ...
  ... Passed -- t  0.4 nrpc     4 ops    0
Test: memory use put ...
  ... Passed -- t  0.2 nrpc     2 ops    0
Test: memory use append ...
  ... Passed -- t  0.4 nrpc     2 ops    0
Test: memory use many puts ...
  ... Passed -- t 11.5 nrpc 100000 ops    0
Test: memory use many gets ...
  ... Passed -- t 12.2 nrpc 100001 ops    0
PASS
ok      6.5840/kvsrv    39.000s
```

每次测试通过后显示的数字分别是：实际时间（秒），发送的 RPC 数量（包括客户端 RPC），以及执行的键/值操作数量（`Clerk` 的 `Get/Put/Append` 调用）。