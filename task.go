package gosf

import (
	"sync"
)

type Task struct {
}

var tasks []func()

// Add 添加任务
func (t *Task) Add(task func()) {
	tasks = append(tasks, task)
}

// Run 运行
func (t *Task) Run() {

	var taskWg sync.WaitGroup

	taskWg.Add(len(tasks))

	for i, n := 0, len(tasks); i < n; i++ {
		go func(f func()) {
			defer taskWg.Done()
			f()
		}(tasks[i])
	}

	taskWg.Wait()
}
