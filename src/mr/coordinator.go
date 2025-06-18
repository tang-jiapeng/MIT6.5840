package mr

import (
	"log"
	"net"
	"net/http"
	"net/rpc"
	"os"
	"sync"
	"time"
)

// 管理任务状态
// 处理worker的RPC请求
// 检测任务是否超时

type Task struct {
	FileName  string
	Status    string
	StartTime time.Time
	TaskID    int
}

type Coordinator struct {
	// Your definitions here.
	mu          sync.Mutex
	mapTasks    []Task
	reduceTasks []Task
	nReduce     int
	mapFinished bool
	allFinished bool
	files       []string
	nextTaskID  int
}

// Your code here -- RPC handlers for the worker to call.

// Worker调用此方法从Coordinator获取任务
func (c *Coordinator) GetTask(args *GetTaskArgs, reply *GetTaskReply) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// 检查超时任务
	c.checkTimeout()

	// 如果所有任务完成，通知worker退出
	if c.allFinished {
		reply.TaskType = ExitTask
		return nil
	}

	// 如果map任务还未执行完，分配map任务
	if !c.mapFinished {
		for i, task := range c.mapTasks {
			if task.Status == Idle {
				reply.TaskType = MapTask
				reply.TaskID = task.TaskID
				reply.FileName = task.FileName
				reply.NReduce = c.nReduce

				// 更新任务状态
				c.mapTasks[i].Status = InProgress
				c.mapTasks[i].StartTime = time.Now()
				return nil
			}
		}

		// 如果找不到一个空闲的map任务，但还没有执行完
		reply.TaskType = WaitTask
		return nil
	}

	// 如果所有map任务执行完，则分配reduce任务
	for i, task := range c.reduceTasks {
		if task.Status == Idle {
			reply.TaskType = ReduceTask
			reply.TaskID = task.TaskID
			reply.ReduceTaskNum = i
			reply.MapTaskNum = len(c.mapTasks)

			// 更新任务状态
			c.reduceTasks[i].Status = InProgress
			c.reduceTasks[i].StartTime = time.Now()
			return nil
		}
	}
	// 没有任何空闲的reduce任务
	reply.TaskType = WaitTask
	return nil
}

func (c *Coordinator) checkTimeout() {
	// 超时时间设置为10s
	timeout := 10 * time.Second

	now := time.Now()

	// 检查map任务是否超时
	if !c.mapFinished {
		allCompleted := true
		for i, task := range c.mapTasks {
			if task.Status == InProgress && now.Sub(task.StartTime) > timeout {
				// 任务已超时
				c.mapTasks[i].Status = Idle
			}
			if task.Status != Completed {
				allCompleted = false
			}
		}
		c.mapFinished = allCompleted
	}

	// 检查reduce任务是否超时
	allCompleted := true
	for i, task := range c.reduceTasks {
		if task.Status == InProgress && now.Sub(task.StartTime) > timeout {
			c.reduceTasks[i].Status = Idle
		}
		if task.Status != Completed {
			allCompleted = false
		}
	}
	c.allFinished = allCompleted
}

// Worker完成任务后调用此方法通知Coordinator
func (c *Coordinator) ReportTask(args *ReportTaskArgs, reply *ReportTaskReply) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if args.TaskType == MapTask {
		for i, task := range c.mapTasks {
			if task.TaskID == args.TaskID && task.Status == InProgress {
				c.mapTasks[i].Status = Completed

				// 检查所有map任务是否已经全部完成
				allCompleted := true
				for _, task := range c.mapTasks {
					if task.Status != Completed {
						allCompleted = false
						break
					}
				}
				c.mapFinished = allCompleted
				reply.OK = true
				return nil
			}
		}
	} else if args.TaskType == ReduceTask {
		for i, task := range c.reduceTasks {
			if task.TaskID == args.TaskID && task.Status == InProgress {
				c.reduceTasks[i].Status = Completed

				// 检查所有reduce任务是否已经全部完成
				allCompleted := true
				for _, task := range c.reduceTasks {
					if task.Status != Completed {
						allCompleted = false
						break
					}
				}
				c.mapFinished = allCompleted
				reply.OK = true
				return nil
			}
		}
	}
	reply.OK = false
	return nil
}

// an example RPC handler.
//
// the RPC argument and reply types are defined in rpc.go.
func (c *Coordinator) Example(args *ExampleArgs, reply *ExampleReply) error {
	reply.Y = args.X + 1
	return nil
}

// start a thread that listens for RPCs from worker.go
func (c *Coordinator) server() {
	rpc.Register(c)
	rpc.HandleHTTP()
	//l, e := net.Listen("tcp", ":1234")
	sockname := coordinatorSock()
	os.Remove(sockname)
	l, e := net.Listen("unix", sockname)
	if e != nil {
		log.Fatal("listen error:", e)
	}
	go http.Serve(l, nil)
}

// main/mrcoordinator.go calls Done() periodically to find out
// if the entire job has finished.
func (c *Coordinator) Done() bool {
	ret := false

	// Your code here.

	if c.allFinished {
		ret = true
	}

	return ret
}

// create a Coordinator.
// main/mrcoordinator.go calls this function.
// nReduce is the number of reduce tasks to use.
func MakeCoordinator(files []string, nReduce int) *Coordinator {
	c := Coordinator{
		files:       files,
		nReduce:     nReduce,
		mapTasks:    make([]Task, len(files)),
		reduceTasks: make([]Task, nReduce),
		mapFinished: false,
		allFinished: false,
		nextTaskID:  0,
	}

	// 初始化map任务
	for i, file := range c.files {
		c.mapTasks[i] = Task{
			TaskID:   i,
			FileName: file,
			Status:   Idle,
		}
	}

	// 初始化reduce任务
	for i := 0; i < nReduce; i++ {
		c.reduceTasks[i] = Task{
			TaskID: i,
			Status: Idle,
		}
	}

	c.server()
	return &c
}
