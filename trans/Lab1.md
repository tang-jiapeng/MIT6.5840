# 6.5840 Lab 1: MapReduce

## Introduction

在本实验中，你将构建一个 MapReduce 系统。你需要实现一个工作进程，调用应用的 Map 和 Reduce 函数，并处理文件的读写；以及一个协调进程，负责向工作进程分配任务并处理失败的工作进程。你将构建一个类似于 MapReduce 论文描述的系统。（注意：本实验使用“协调者”而不是论文中的“主节点”。）

## Getting started

你需要设置 Go 环境以完成实验。
使用 git（一个版本控制系统）获取初始实验软件。要了解更多关于 git 的信息，请查看《Pro Git》书籍或 git 用户手册。
```bash
$ git clone git://g.csail.mit.edu/6.5840-golabs-2024 6.5840
$ cd 6.5840
$ ls
Makefile src
$
```

我们为你提供了一个简单的顺序 MapReduce 实现，位于 src/main/mrsequential.go。它在一个进程中逐一运行 Map 和 Reduce。我们还提供了一些 MapReduce 应用：位于 mrapps/wc.go 的单词计数，以及位于 mrapps/indexer.go 的文本索引器。你可以按以下方式顺序运行单词计数：
```bash
$ cd ~/6.5840
$ cd src/main
$ go build -buildmode=plugin ../mrapps/wc.go
$ rm mr-out*
$ go run mrsequential.go wc.so pg*.txt
$ more mr-out-0
A 509
ABOUT 2
ACT 8
...
```

mrsequential.go 将其输出保存在文件 mr-out-0 中。输入来自名为 pg-xxx.txt 的文本文件。
你可以自由借用 mrsequential.go 的代码。你还应该查看 mrapps/wc.go，以了解 MapReduce 应用代码的样子。
对于本实验及所有其他实验，我们可能会更新提供的代码。为确保你可以使用 git pull 获取这些更新并轻松合并，建议保留我们提供的代码在原始文件中。你可以根据实验说明在提供的代码中添加内容；只是不要移动它们。你可以在新文件中添加自己的新函数。

## Your Job (moderate/hard)

你的任务是实现一个分布式 MapReduce 系统，包括两个程序：协调者和工作进程。系统中将只有一个协调者进程，以及一个或多个并行执行的工作进程。在实际系统中，工作进程会在多台机器上运行，但在本实验中，你将在单台机器上运行所有工作进程。工作进程将通过 RPC 与协调者通信。每个工作进程将循环地向协调者请求任务，读取任务的输入文件，执行任务，将任务的输出写入文件，然后再次向协调者请求新任务。协调者应能注意到如果某个工作进程在合理时间内（本实验中使用十秒）未完成任务，则将同一任务分配给另一个工作进程。
我们已经为你提供了一些启动代码。协调者和工作进程的“main”例程分别位于 main/mrcoordinator.go 和 main/mrworker.go；不要修改这些文件。你应该在 mr/coordinator.go、mr/worker.go 和 mr/rpc.go 中实现你的代码。
以下是如何在单词计数 MapReduce 应用上运行你的代码。首先，确保单词计数插件是最新构建的：
```bash
$ go build -buildmode=plugin ../mrapps/wc.go
```

在主目录中，运行协调者：
```bash
$ rm mr-out*
$ go run mrcoordinator.go pg-*.txt
```

mrcoordinator.go 的 pg-*.txt 参数是输入文件；每个文件对应一个“分片”，并作为一项 Map 任务的输入。
在一个或多个其他窗口中，运行一些工作进程：
```bash
$ go run mrworker.go wc.so
```

当工作进程和协调者完成后，查看 mr-out-* 中的输出。完成实验后，输出文件的排序合并结果应与顺序输出的结果匹配，如下所示：
```bash
$ cat mr-out-* | sort | more
A 509
ABOUT 2
ACT 8
...
```

我们为你提供了一个测试脚本，位于 main/test-mr.sh。测试脚本会检查 wc 和 indexer MapReduce 应用在以 pg-xxx.txt 文件为输入时是否产生正确输出。测试还会检查你的实现是否并行运行 Map 和 Reduce 任务，以及你的实现是否能从运行任务时崩溃的工作进程中恢复。
如果你现在运行测试脚本，它会挂起，因为协调者从未结束：
```
$ cd ~/6.5840/src/main
$ bash test-mr.sh
*** Starting wc test.
```

你可以将 mr/coordinator.go 中 Done 函数的 ret := false 改为 true，以便协调者立即退出。然后：
```bash
$ bash test-mr.sh
*** Starting wc test.
sort: No such file or directory
cmp: EOF on mr-wc-all
--- wc output is not the same as mr-correct-wc.txt
--- wc test: FAIL
$
```

测试脚本期望看到每个 Reduce 任务的输出保存在名为 mr-out-X 的文件中。mr/coordinator.go 和 mr/worker.go 的空实现不会生成这些文件（或执行其他操作），因此测试会失败。
完成后，测试脚本的输出应如下所示：
```bash
$ bash test-mr.sh
*** Starting wc test.
--- wc test: PASS
*** Starting indexer test.
--- indexer test: PASS
*** Starting map parallelism test.
--- map parallelism test: PASS
*** Starting reduce parallelism test.
--- reduce parallelism test: PASS
*** Starting job count test.
--- job count test: PASS
*** Starting early exit test.
--- early exit test: PASS
*** Starting crash test.
--- crash test: PASS
*** PASSED ALL TESTS
$
```

你可能会看到一些来自 Go RPC 包的错误，例如：
```bash
2019/12/16 13:27:09 rpc.Register: method "Done" has 1 input parameters; needs exactly three
```

请忽略这些消息；将协调者注册为 RPC 服务器时会检查其所有方法是否适合 RPC（需要 3 个输入）；我们知道 Done 方法不是通过 RPC 调用的。
此外，根据你终止工作进程的策略，你可能会看到一些如下形式的错误：
```bash
2024/02/11 16:21:32 dialing:dial unix /var/tmp/5840-mr-501: connect: connection refused
```

每次测试看到少量此类消息是可以的；它们是在协调者退出后，工作进程无法联系协调者 RPC 服务器时出现的。

## A few rules:

- Map 阶段应将中间键分为 nReduce 个 Reduce 任务的桶，其中 nReduce 是 Reduce 任务的数量——main/mrcoordinator.go 传递给 MakeCoordinator() 的参数。每个 Map 任务应创建 nReduce 个中间文件，供 Reduce 任务使用。
- 工作进程实现应将第 X 个 Reduce 任务的输出保存在文件 mr-out-X 中。
- mr-out-X 文件应为每个 Reduce 函数输出包含一行。行应使用 Go 的 "%v %v" 格式生成，调用时传入键和值。请查看 main/mrsequential.go 中注释为“this is the correct format”的行。如果你的实现偏离此格式太多，测试脚本将会失败。
- 你可以修改 mr/worker.go、mr/coordinator.go 和 mr/rpc.go。你可以临时修改其他文件进行测试，但确保你的代码与原始版本兼容；我们将使用原始版本进行测试。
- 工作进程应将中间 Map 输出保存在当前目录的文件中，供后续 Reduce 任务读取。
- main/mrcoordinator.go 期望 mr/coordinator.go 实现一个 Done() 方法，当 MapReduce 作业完全完成时返回 true；此时，mrcoordinator.go 将退出。
- 当作业完全完成时，工作进程应退出。一个简单的实现方法是使用 call() 的返回值：如果工作进程无法联系协调者，它可以假设协调者因作业完成而退出，因此工作进程也可以终止。根据你的设计，你可能会发现让协调者向工作进程发送“请退出”伪任务也很有帮助。

## Hints

- 指导页面提供了一些开发和调试的提示。
- 入门的一种方法是修改 mr/worker.go 的 Worker()，使其向协调者发送 RPC 请求任务。然后修改协调者以返回尚未开始的 Map 任务的文件名。然后修改工作进程以读取该文件并调用应用 Map 函数，如 mrsequential.go 中所示。
- 应用的 Map 和 Reduce 函数在运行时使用 Go 的 plugin 包加载，来自以 .so 结尾的文件。
- 如果你更改了 mr/ 目录中的任何内容，你可能需要重新构建使用的 MapReduce 插件，例如：go build -buildmode=plugin ../mrapps/wc.go

- 本实验依赖于工作进程共享文件系统。当所有工作进程在同一台机器上运行时，这很简单；但如果工作进程在不同机器上运行，则需要像 GFS 这样的全局文件系统。
- 中间文件的合理命名约定是 mr-X-Y，其中 X 是 Map 任务编号，Y 是 Reduce 任务编号。
- 工作进程的 Map 任务代码需要一种方式将中间键/值对存储在文件中，以便在 Reduce 任务中正确读取。一种可能是使用 Go 的 encoding/json 包。将键/值对以 JSON 格式写入打开的文件：
  ```go
  enc := json.NewEncoder(file)
  for _, kv := ... {
    err := enc.Encode(&kv)
  ```
  读取这样的文件：
  ```go
  dec := json.NewDecoder(file)
  for {
    var kv KeyValue
    if err := dec.Decode(&kv); err != nil {
      break
    }
    kva = append(kva, kv)
  }
  ```

- 工作进程的 Map 部分可以使用 worker.go 中的 ihash(key) 函数为给定键选择 Reduce 任务。
- 你可以从 mrsequential.go 中借用一些代码，用于读取 Map 输入文件，在 Map 和 Reduce 之间排序中间键/值对，以及将 Reduce 输出存储在文件中。
- 协调者作为 RPC 服务器将是并发的；不要忘记锁定共享数据。
- 使用 Go 的竞态检测器，带上 go run -race。test-mr.sh 开头有一个注释，告诉你如何使用 -race 运行它。我们在评分实验时不会使用竞态检测器。但如果你的代码存在竞态问题，即使不使用竞态检测器，它也很可能在我们测试时失败。
- 工作进程有时需要等待，例如，Reduce 任务在最后一个 Map 任务完成前不能开始。一种可能是工作进程定期向协调者请求工作，在每次请求之间使用 time.Sleep() 休眠。另一种可能是协调者中相关的 RPC 处理程序包含一个等待循环，使用 time.Sleep() 或 sync.Cond。Go 为每个 RPC 在单独的线程中运行处理程序，因此一个处理程序的等待不会阻止协调者处理其他 RPC。
- 协调者无法可靠区分崩溃的工作进程、存活但因某种原因停滞的工作进程，以及执行但速度太慢而无用的工作进程。你能做的最好方法是让协调者等待一段时间，然后放弃并将任务重新分配给另一个工作进程。对于本实验，让协调者等待十秒；之后，协调者应假设工作进程已死亡（当然，实际上可能没有）。
- 如果你选择实现备份任务（第 3.6 节），请注意，我们会测试你的代码在工作进程正常执行任务而不崩溃时不会调度多余的任务。备份任务应在较长时间（例如 10 秒）后才调度。
- 要测试崩溃恢复，你可以使用 mrapps/crash.go 应用插件。它会在 Map 和 Reduce 函数中随机退出。
- 为确保在崩溃时没有人观察到部分写入的文件，MapReduce 论文提到了使用临时文件并在完全写入后原子重命名的技巧。你可以使用 ioutil.TempFile（或 Go 1.17 及更高版本的 os.CreateTemp）创建临时文件，并使用 os.Rename 原子重命名。
- test-mr.sh 在子目录 mr-tmp 中运行所有进程，因此如果出现问题，你想查看中间或输出文件，请在那里查找。你可以临时修改 test-mr.sh，使其在失败的测试后退出，以便脚本不会继续测试（并覆盖输出文件）。
- test-mr-many.sh 会连续多次运行 test-mr.sh，你可能需要这样做以发现低概率的错误。它接受一个参数，指定运行测试的次数。你不应并行运行多个 test-mr.sh 实例，因为协调者会重用相同的套接字，导致冲突。
- Go RPC 仅发送字段名以大写字母开头的结构体。子结构体的字段名也必须大写。
- 调用 RPC call() 函数时，回复结构体应包含所有默认值。RPC 调用应如下所示：
  ```go
  reply := SomeType{}
  call(..., &reply)
  ```
  在调用前不要设置 reply 的任何字段。如果你传递的 reply 结构体包含非默认字段，RPC 系统可能会悄悄返回错误值。


