package common

import (
	"context"
)

// IActionTask ...
type IActionTask interface {
	Execute()
}

// ActionTaskFunc ...
type ActionTaskFunc func()

// Execute ...
func (f ActionTaskFunc) Execute() {
	f()
}

// SyncAction ...
type SyncAction struct {
	actionTasks chan IActionTask
	ctx         context.Context
}

func (sa *SyncAction) serve() {
	defer func() {
		if e := recover(); e != nil {
			LogError(e)
		}
	}()
	for {
		select {
		case task := <-sa.actionTasks:
			if task != nil {
				task.Execute()
			}
		case <-sa.ctx.Done():
			return
		}

	}
}

// RunTaskFunc ....
func (sa *SyncAction) RunTaskFunc(taskFunc ActionTaskFunc) {
	sa.RunTask(taskFunc)
}

// RunTask ...
func (sa *SyncAction) RunTask(task IActionTask) {
	wt := make(chan struct{})
	sa.actionTasks <- ActionTaskFunc(func() {
		defer func() {
			close(wt)
		}()
		task.Execute()
	})
	select {
	case <-wt:
	case <-sa.ctx.Done():
	}
}

// StartNewSyncAction ...
func StartNewSyncAction(ctx context.Context, taskBufSize int) *SyncAction {
	sa := &SyncAction{
		ctx:         ctx,
		actionTasks: make(chan IActionTask, taskBufSize),
	}
	go sa.serve()
	return sa
}
