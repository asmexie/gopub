package common

type TunnelTaskFunc func()

type TunnelTask struct {
	task     TunnelTaskFunc
	execOver chan bool
}

func TunnelExec(tasks chan *TunnelTask, task TunnelTaskFunc) {
	tunnelTask := &TunnelTask{
		task:     task,
		execOver: make(chan bool, 1),
	}
	tasks <- tunnelTask
	<-tunnelTask.execOver
}

func RunTunnelTasks(tasks chan *TunnelTask) {
	for {
		func() {
			tunnelTask := <-tasks
			defer func() {
				tunnelTask.execOver <- true
				if x := recover(); x != nil {
					LogError(x)
				}
			}()
			tunnelTask.task()
		}()
	}
}
