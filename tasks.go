/*
Package tasks is an easy to use in-process scheduler for recurring tasks in Go. Tasks is focused on high frequency
tasks that run quick, and often. The goal of Tasks is to support concurrent running tasks at scale without scheduler
induced jitter.

Tasks is focused on accuracy of task execution. To do this each task is called within it's own goroutine.
This ensures that long execution of a single invocation does not throw the schedule as a whole off track.

As usage of this scheduler scales, it is expected to have a larger number of sleeping goroutines. As it is
designed to leverage Go's ability to optimize goroutine CPU scheduling.

For simplicity this task scheduler uses the time.Duration type to specify intervals. This allows for a simple
interface and flexible control over when tasks are executed.

Below is an example of starting the scheduler and registering a new task that runs every 30 seconds.

	// Start the Scheduler
	scheduler := tasks.New()
	defer scheduler.Stop()

	// Add a task
	id, err := scheduler.Add(&tasks.Task{
		Interval: time.Duration(30 * time.Second),
		TaskFunc: func() error {
			// Put your logic here
		},
	})
	if err != nil {
		// Do Stuff
	}

Sometimes schedules need to started at a later time. This package provides the ability to start a task only after
a certain time. The below example shows this in practice.

	// Add a recurring task for every 30 days, starting 30 days from now
	id, err := scheduler.Add(&tasks.Task{
		Interval: time.Duration(30 * (24 * time.Hour)),
		StartAfter: time.Now().Add(30 * (24 * time.Hour)),
		TaskFunc: func() error {
			// Put your logic here
		},
	})
	if err != nil {
		// Do Stuff
	}

It is also common for applications to run a task only once. The below example shows scheduling a task to run only once after
waiting for 60 seconds.

	// Add a one time only task for 60 seconds from now
	id, err := scheduler.Add(&tasks.Task{
		Interval: time.Duration(60 * time.Second)
		RunOnce:  true,
		TaskFunc: func() error {
			// Put your logic here
		},
	})
	if err != nil {
		// Do Stuff
	}

One powerful feature of Tasks is that it allows users to specify custom error handling. This is done by allowing users to
define a function that is called when a task returns an error. The below example shows scheduling a task that logs when an
error occurs.

	// Add a task with custom error handling
	id, err := scheduler.Add(&tasks.Task{
		Interval: time.Duration(30 * time.Second),
		TaskFunc: func() error {
			// Put your logic here
		}(),
		ErrFunc: func(e error) {
			log.Printf("An error occurred when executing task %s - %s", id, e)
		},
	})
	if err != nil {
		// Do Stuff
	}
*/
package tasks

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/rs/xid"
)

// Task contains the scheduled task details and control mechanisms. This struct is used during the creation of tasks.
// It allows users to control how and when tasks are executed.
type Task struct {
	sync.RWMutex

	// TaskContext allows for user-defined context that is passed to task functions.
	TaskContext TaskContext

	// Interval is the frequency that the task executes. Defining this at 30 seconds, will result in a task that
	// runs every 30 seconds.
	//
	// The below are common examples to get started with.
	//
	//  // Every 30 seconds
	//  time.Duration(30 * time.Second)
	//  // Every 5 minutes
	//  time.Duration(5 * time.Minute)
	//  // Every 12 hours
	//  time.Duration(12 * time.Hour)
	//  // Every 30 days
	//  time.Duration(30 * (24 * time.Hour))
	//
	Interval time.Duration

	// RunOnce is used to set this task as a single execution task. By default, tasks will continue executing at
	// the interval specified until deleted. With RunOnce enabled the first execution of the task will result in
	// the task self deleting.
	RunOnce bool

	// RunSingleInstance is used to set a task as a single instance task. By default, tasks will continue executing at
	// the interval specified until deleted. With RunSingleInstance enabled a subsequent task execution will be skipped
	// if the previous task execution is still running.
	//
	// This is useful for tasks that may take longer than the interval to execute. This will prevent multiple instances
	// of the same task from running concurrently.
	RunSingleInstance bool

	// StartAfter is used to specify a start time for the scheduler. When set, tasks will wait for the specified
	// time to start the schedule timer.
	StartAfter time.Time

	// TaskFunc is the user defined function to execute as part of this task.
	//
	// Either TaskFunc or FuncWithTaskContext must be defined. If both are defined, FuncWithTaskContext will be used.
	TaskFunc func() error

	// ErrFunc allows users to define a function that is called when tasks return an error. If ErrFunc is nil,
	// errors from tasks will be ignored.
	//
	// Either ErrFunc or ErrFuncWithTaskContext must be defined. If both are defined, ErrFuncWithTaskContext will be used.
	ErrFunc func(error)

	// FuncWithTaskContext is a user defined function to execute as part of this task. This function is used in
	// place of TaskFunc with the difference in that it will pass the user defined context from the Task configurations.
	//
	// Either TaskFunc or FuncWithTaskContext must be defined. If both are defined, FuncWithTaskContext will be used.
	FuncWithTaskContext func(TaskContext) error

	// ErrFuncWithTaskContext allows users to define a function that is called when tasks return an error.
	// If ErrFunc is nil, errors from tasks will be ignored. This function is used in place of ErrFunc with
	// the difference in that it will pass the user defined context from the Task configurations.
	//
	// Either ErrFunc or ErrFuncWithTaskContext must be defined. If both are defined, ErrFuncWithTaskContext will be used.
	ErrFuncWithTaskContext func(TaskContext, error)

	// id is the Unique ID created for each task. This ID is generated by the Add() function.
	id string

	// running is used for RunSingleInstance tasks to track whether a previous invocation is still running.
	running sync.Mutex

	// timer is the internal task timer. This is stored here to provide control via main scheduler functions.
	timer *time.Timer

	// ctx is the internal context used to control task cancelation.
	ctx context.Context

	// cancel is used to cancel tasks gracefully. This will not interrupt a task function that has already been
	// triggered.
	cancel context.CancelFunc
}

type TaskContext struct {
	// Context is a user-defined context.
	Context context.Context

	// id is the Unique ID created for each task. This ID is generated by the Add() function.
	id string
}

// safeOps safely change task's data
func (t *Task) safeOps(f func()) {
	t.Lock()
	defer t.Unlock()

	f()
}

// Scheduler stores the internal task list and provides an interface for task management.
type Scheduler struct {
	sync.RWMutex

	// tasks is the internal task list used to store tasks that are currently scheduled.
	tasks map[string]*Task
}

var (
	// ErrIDInUse is returned when a Task ID is specified but already used.
	ErrIDInUse = fmt.Errorf("ID already used")
)

// New will create a new scheduler instance that allows users to create and manage tasks.
func New() *Scheduler {
	s := &Scheduler{}
	s.tasks = make(map[string]*Task)
	return s
}

// Add will add a task to the task list and schedule it. Once added, tasks will wait the defined time interval and then
// execute. This means a task with a 15 second interval will be triggered 15 seconds after Add is complete. Not before
// or after (excluding typical machine time jitter).
//
//	// Add a task
//	id, err := scheduler.Add(&tasks.Task{
//		Interval: time.Duration(30 * time.Second),
//		TaskFunc: func() error {
//			// Put your logic here
//		}(),
//		ErrFunc: func(err error) {
//			// Put custom error handling here
//		}(),
//	})
//	if err != nil {
//		// Do stuff
//	}
func (schd *Scheduler) Add(t *Task) (string, error) {
	id := xid.New()
	err := schd.AddWithID(id.String(), t)
	if err == ErrIDInUse {
		return schd.Add(t)
	}
	return id.String(), err
}

// AddWithID will add a task with an ID to the task list and schedule it. It will return an error if the ID is in-use.
// Once added, tasks will wait the defined time interval and then execute. This means a task with a 15 second interval
// will be triggered 15 seconds after Add is complete. Not before or after (excluding typical machine time jitter).
//
//	// Add a task
//	id := xid.New()
//	err := scheduler.AddWithID(id, &tasks.Task{
//		Interval: time.Duration(30 * time.Second),
//		TaskFunc: func() error {
//			// Put your logic here
//		}(),
//		ErrFunc: func(err error) {
//			// Put custom error handling here
//		}(),
//	})
//	if err != nil {
//		// Do stuff
//	}
func (schd *Scheduler) AddWithID(id string, t *Task) error {
	// Check if TaskFunc is nil before doing anything
	if t.TaskFunc == nil && t.FuncWithTaskContext == nil {
		return fmt.Errorf("task function cannot be nil")
	}

	// Ensure Interval is never 0, this would cause Timer to panic
	if t.Interval <= time.Duration(0) {
		return fmt.Errorf("task interval must be defined")
	}

	// Create Context used to cancel downstream Goroutines
	t.ctx, t.cancel = context.WithCancel(context.Background())

	// Add id to TaskContext
	t.TaskContext.id = id

	// Check id is not in use, then add to task list and start background task
	schd.Lock()
	defer schd.Unlock()
	if _, ok := schd.tasks[id]; ok {
		return ErrIDInUse
	}
	t.id = id

	// To make up for bad design decisions we need to copy the task for execution
	task := t.Clone()

	// Add task to schedule
	schd.tasks[t.id] = task
	schd.scheduleTask(task)

	return nil
}

// Del will unschedule the specified task and remove it from the task list. Deletion will prevent future invocations of
// a task, but not interrupt a trigged task.
func (schd *Scheduler) Del(name string) {
	// Grab task from task list
	t, err := schd.Lookup(name)
	if err != nil {
		return
	}

	// Stop the task
	defer t.cancel()

	t.Lock()
	defer t.Unlock()

	if t.timer != nil {
		defer t.timer.Stop()
	}

	// Remove from task list
	schd.Lock()
	defer schd.Unlock()
	delete(schd.tasks, name)
}

// Lookup will find the specified task from the internal task list using the task ID provided.
//
// The returned task should be treated as read-only, and not modified outside of this package. Doing so, may cause
// panics.
func (schd *Scheduler) Lookup(name string) (*Task, error) {
	schd.RLock()
	defer schd.RUnlock()
	t, ok := schd.tasks[name]
	if ok {
		return t.Clone(), nil
	}
	return t, fmt.Errorf("could not find task within the task list")
}

// Tasks is used to return a copy of the internal tasks map.
//
// The returned task should be treated as read-only, and not modified outside of this package. Doing so, may cause
// panics.
func (schd *Scheduler) Tasks() map[string]*Task {
	schd.RLock()
	defer schd.RUnlock()
	m := make(map[string]*Task)
	for k, v := range schd.tasks {
		m[k] = v.Clone()
	}
	return m
}

// Stop is used to unschedule and delete all tasks owned by the scheduler instance.
func (schd *Scheduler) Stop() {
	tt := schd.Tasks()
	for n := range tt {
		schd.Del(n)
	}
}

// scheduleTask creates the underlying scheduled task. If StartAfter is set, this routine will wait until the
// time specified.
func (schd *Scheduler) scheduleTask(t *Task) {
	_ = time.AfterFunc(time.Until(t.StartAfter), func() {
		var err error

		// Verify if task has been cancelled before scheduling
		t.safeOps(func() {
			err = t.ctx.Err()
		})
		if err != nil {
			// Task has been cancelled, do not schedule
			return
		}

		// Schedule task
		t.safeOps(func() {
			t.timer = time.AfterFunc(t.Interval, func() { schd.execTask(t) })
		})
	})
}

// execTask is the underlying scheduler, it is used to trigger and execute tasks.
func (schd *Scheduler) execTask(t *Task) {
	go func() {
		if t.RunSingleInstance {
			if !t.running.TryLock() {
				// Skip execution if task is already running
				return
			}
			defer t.running.Unlock()
		}

		// Execute task
		var err error
		if t.FuncWithTaskContext != nil {
			err = t.FuncWithTaskContext(t.TaskContext)
		} else {
			err = t.TaskFunc()
		}
		if err != nil && (t.ErrFunc != nil || t.ErrFuncWithTaskContext != nil) {
			if t.ErrFuncWithTaskContext != nil {
				go t.ErrFuncWithTaskContext(t.TaskContext, err)
			} else {
				go t.ErrFunc(err)
			}
		}

		// If RunOnce is set, delete the task after execution
		if t.RunOnce {
			defer schd.Del(t.id)
		}
	}()

	// Reschedule task for next execution
	if !t.RunOnce {
		t.safeOps(func() {
			t.timer.Reset(t.Interval)
		})
	}
}

// ID will return the task ID. This is the same as the ID generated by the scheduler when adding a task.
// If the task was added with AddWithID, this will be the same as the ID provided.
func (ctx TaskContext) ID() string {
	return ctx.id
}

// Clone will create a copy of the existing task. This is useful for creating a new task with the same properties as
// an existing task. It is also used internally when creating a new task.
func (t *Task) Clone() *Task {
	task := &Task{}
	t.safeOps(func() {
		task.TaskFunc = t.TaskFunc
		task.FuncWithTaskContext = t.FuncWithTaskContext
		task.ErrFunc = t.ErrFunc
		task.ErrFuncWithTaskContext = t.ErrFuncWithTaskContext
		task.Interval = t.Interval
		task.StartAfter = t.StartAfter
		task.RunOnce = t.RunOnce
		task.RunSingleInstance = t.RunSingleInstance
		task.id = t.id
		task.ctx = t.ctx
		task.cancel = t.cancel
		task.timer = t.timer
		task.TaskContext = t.TaskContext
	})
	return task
}
