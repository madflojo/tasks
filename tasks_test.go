package tasks

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/rs/xid"
)

const (
	testInterval     = 25 * time.Millisecond
	testStartDelay   = 75 * time.Millisecond
	testTimeout      = 2 * time.Second
	testNoRunTimeout = 150 * time.Millisecond
)

type InterfaceTestCase struct {
	name   string
	task   *Task
	id     string
	addErr bool
}

type ExecutionTestCase struct {
	id        string
	ctx       context.Context
	task      *Task
	callsFunc bool
}

type Counter struct {
	sync.RWMutex
	val int
}

func NewCounter() *Counter {
	return &Counter{}
}

func (c *Counter) Inc() {
	c.Lock()
	defer c.Unlock()
	c.val++
}

func (c *Counter) Dec() {
	c.Lock()
	defer c.Unlock()
	c.val--
}

func (c *Counter) Val() int {
	c.RLock()
	defer c.RUnlock()
	return c.val
}

func TestTasksInterface(t *testing.T) {
	var tt []InterfaceTestCase

	tt = append(tt, InterfaceTestCase{
		name: "Basic Valid Task",
		task: &Task{
			Interval: testInterval,
			TaskFunc: func() error { return nil },
		},
	})

	tt = append(tt, InterfaceTestCase{
		name: "Basic Valid Task with ID",
		task: &Task{
			Interval: testInterval,
			TaskFunc: func() error { return nil },
		},
		id: xid.New().String(),
	})

	tt = append(tt, InterfaceTestCase{
		name: "Valid Task with ErrFunc",
		task: &Task{
			Interval: testInterval,
			TaskFunc: func() error { return nil },
			ErrFunc:  func(_ error) {},
		},
	})

	tt = append(tt, InterfaceTestCase{
		name: "Valid Task with Context",
		task: &Task{
			Interval:    testInterval,
			TaskFunc:    func() error { return nil },
			ErrFunc:     func(_ error) {},
			TaskContext: TaskContext{Context: context.Background()},
		},
	})

	tt = append(tt, InterfaceTestCase{
		name: "Valid Task with Context and WithContextFuncs",
		task: &Task{
			Interval:               testInterval,
			FuncWithTaskContext:    func(_ TaskContext) error { return nil },
			ErrFuncWithTaskContext: func(_ TaskContext, _ error) {},
			TaskContext:            TaskContext{Context: context.Background()},
		},
	})

	tt = append(tt, InterfaceTestCase{
		name: "Valid Task without Context but WithContextFuncs",
		task: &Task{
			Interval:               testInterval,
			FuncWithTaskContext:    func(_ TaskContext) error { return nil },
			ErrFuncWithTaskContext: func(_ TaskContext, _ error) {},
		},
	})

	tt = append(tt, InterfaceTestCase{
		name: "Valid Task with StartAfter",
		task: &Task{
			Interval:   testInterval,
			TaskFunc:   func() error { return nil },
			StartAfter: time.Now().Add(testInterval),
		},
	})

	tt = append(tt, InterfaceTestCase{
		name: "Valid Task with StartAfter but in the past",
		task: &Task{
			Interval:   testInterval,
			TaskFunc:   func() error { return nil },
			StartAfter: time.Now().Add(time.Duration(-1 * time.Minute)),
		},
	})

	tt = append(tt, InterfaceTestCase{
		name: "Valid Task with RunOnce",
		task: &Task{
			Interval: testInterval,
			TaskFunc: func() error { return nil },
			RunOnce:  true,
		},
	})

	tt = append(tt, InterfaceTestCase{
		name: "No Interval",
		task: &Task{
			TaskFunc: func() error { return nil },
		},
		addErr: true,
	})

	tt = append(tt, InterfaceTestCase{
		name: "No TaskFunc or FuncWithTaskContext",
		task: &Task{
			Interval: testInterval,
		},
		addErr: true,
	})

	// Create a base scheduler to use
	scheduler := New()
	defer scheduler.Stop()

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			var err error
			id := tc.id

			// Schedule the task
			if tc.id != "" {
				err = scheduler.AddWithID(tc.id, tc.task)
			} else {
				id, err = scheduler.Add(tc.task)
			}
			if err != nil && !tc.addErr {
				t.Errorf("Unexpected errors when scheduling a valid task - %s", err)
			}
			if err == nil && tc.addErr {
				t.Errorf("Expected errors when scheduling an invalid task")
			}
			defer scheduler.Del(id)

			if tc.id != "" {
				t.Run(tc.name+" - Duplicate Task", func(t *testing.T) {
					// Schedule the task
					err := scheduler.AddWithID(tc.id, tc.task)
					if !errors.Is(err, ErrIDInUse) {
						t.Errorf("Expected errors when scheduling a duplicate task")
					}
				})
			}

			t.Run(tc.name+" - Lookup", func(t *testing.T) {
				// Verify if task exists
				_, err = scheduler.Lookup(id)
				if err != nil && !tc.addErr {
					t.Errorf("Unable to find newly scheduled task with Lookup - %s", err)
				}
				if err == nil && tc.addErr {
					t.Errorf("Found task that should not exist - %s", id)
				}
			})

			t.Run(tc.name+" - Task List", func(t *testing.T) {
				// Check Task Map
				tasks := scheduler.Tasks()
				if len(tasks) != 1 && !tc.addErr {
					t.Errorf("Unable to find newly scheduled task with Tasks")
				}
				if len(tasks) > 0 && tc.addErr {
					t.Errorf("Found task that should not exist - %s", id)
				}
			})

			// Reset for the next test
			scheduler.Del(id)
		})
	}
}

func TestTaskExecution(t *testing.T) {
	// Create a base scheduler to use
	scheduler := New()
	defer scheduler.Stop()

	cases := []struct {
		name string
		new  func(*testing.T) ExecutionTestCase
	}{
		{
			name: "Valid Task",
			new: func(_ *testing.T) ExecutionTestCase {
				ctx, cancel := context.WithCancel(context.Background())
				return ExecutionTestCase{
					ctx:       ctx,
					callsFunc: true,
					task: &Task{
						Interval: testInterval,
						TaskFunc: func() error { return fmt.Errorf("fake error") },
						ErrFunc: func(e error) {
							if e != nil {
								cancel()
							}
						},
					},
				}
			},
		},
		{
			name: "Valid Task with TaskContext",
			new: func(t *testing.T) ExecutionTestCase {
				ctx, cancel := context.WithCancel(context.Background())
				task := &Task{
					Interval:    testInterval,
					TaskContext: TaskContext{Context: ctx},
					FuncWithTaskContext: func(taskCtx TaskContext) error {
						if taskCtx.Context != ctx {
							t.Logf("TaskContext.Context does not match expected context")
							// return with no error to trigger a timeout failure
							return nil
						}
						return fmt.Errorf("fake error")
					},
					ErrFuncWithTaskContext: func(taskCtx TaskContext, e error) {
						if taskCtx.Context == ctx && e != nil {
							cancel()
						}
						if taskCtx.Context.Err() != context.Canceled {
							t.Errorf("TaskContext.Context should be canceled")
						}
					},
				}

				return ExecutionTestCase{
					ctx:       ctx,
					task:      task,
					callsFunc: true,
				}
			},
		},
		{
			name: "Cancel a Task before it's called",
			new: func(_ *testing.T) ExecutionTestCase {
				ctx, cancel := context.WithCancel(context.Background())
				return ExecutionTestCase{
					ctx: ctx,
					task: &Task{
						Interval:    testInterval,
						StartAfter:  time.Now().Add(testStartDelay),
						TaskContext: TaskContext{Context: ctx},
						TaskFunc: func() error {
							cancel()
							return nil
						},
					},
				}
			},
		},
		{
			name: "Only call ErrFunc if error",
			new: func(t *testing.T) ExecutionTestCase {
				ctx, cancel := context.WithCancel(context.Background())
				return ExecutionTestCase{
					ctx:       ctx,
					callsFunc: true,
					task: &Task{
						Interval: testInterval,
						TaskFunc: func() error {
							cancel()
							return nil
						},
						ErrFunc: func(error) {
							t.Errorf("ErrFunc should not be called")
						},
					},
				}
			},
		},
		{
			name: "Only call ErrFuncWithTaskContext if error",
			new: func(t *testing.T) ExecutionTestCase {
				ctx, cancel := context.WithCancel(context.Background())
				return ExecutionTestCase{
					ctx:       ctx,
					callsFunc: true,
					task: &Task{
						Interval:    testInterval,
						TaskContext: TaskContext{Context: ctx},
						FuncWithTaskContext: func(_ TaskContext) error {
							cancel()
							return nil
						},
						ErrFuncWithTaskContext: func(_ TaskContext, _ error) {
							t.Errorf("ErrFuncWithTaskContext should not be called")
						},
					},
				}
			},
		},
		{
			name: "Validate TaskContext ID",
			new: func(t *testing.T) ExecutionTestCase {
				ctx, cancel := context.WithCancel(context.Background())
				const id = "test-id"
				return ExecutionTestCase{
					id:        id,
					ctx:       ctx,
					callsFunc: true,
					task: &Task{
						Interval:    testInterval,
						TaskContext: TaskContext{Context: ctx},
						FuncWithTaskContext: func(taskCtx TaskContext) error {
							if taskCtx.ID() != id {
								t.Errorf("TaskContext.ID does not match expected ID")
							}
							cancel()
							return nil
						},
					},
				}
			},
		},
		{
			name: "Verify StartAfter time is respected",
			new: func(t *testing.T) ExecutionTestCase {
				ctx, cancel := context.WithCancel(context.Background())
				startAfter := time.Now().Add(testStartDelay)
				return ExecutionTestCase{
					ctx:       ctx,
					callsFunc: true,
					task: &Task{
						Interval:    testInterval,
						StartAfter:  startAfter,
						TaskContext: TaskContext{Context: ctx},
						FuncWithTaskContext: func(_ TaskContext) error {
							if time.Now().Before(startAfter) {
								t.Errorf("Task should not have been called before StartAfter time")
								return nil
							}
							cancel()
							return nil
						},
					},
				}
			},
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			tc := testCase.new(t)
			var err error
			id := tc.id

			if tc.id != "" {
				err = scheduler.AddWithID(tc.id, tc.task)
			} else {
				id, err = scheduler.Add(tc.task)
			}
			if err != nil {
				t.Errorf("Unexpected errors when scheduling a task - %s", err)
			}
			defer scheduler.Del(id)

			// Cancel the task if it's not supposed to be called
			if !tc.callsFunc {
				scheduler.Del(id)
			}

			timeout := testTimeout
			if !tc.callsFunc {
				timeout = testNoRunTimeout
			}

			select {
			case <-tc.ctx.Done():
				if tc.callsFunc {
					return
				}
				t.Errorf("Task was executed when it should not have been")
			case <-time.After(timeout):
				if !tc.callsFunc {
					return
				}
				t.Errorf("Task did not execute within %s", timeout)
			}
		})
	}
}

func TestAdd(t *testing.T) {
	// Create a base scheduler to use
	scheduler := New()
	defer scheduler.Stop()

	errorTests := []struct {
		name     string
		id       string
		task     *Task
		addWith  bool
		wantID   string
		wantErr  error
		lookupID string
	}{
		{
			name:     "Add a nil task returns ErrNilTask",
			wantErr:  ErrNilTask,
			lookupID: "",
		},
		{
			name:     "AddWithID a nil task returns ErrNilTask",
			id:       "nil-task",
			addWith:  true,
			wantErr:  ErrNilTask,
			lookupID: "nil-task",
		},
		{
			name: "Check for nil callback",
			task: &Task{
				Interval: time.Minute,
				ErrFunc:  func(_ error) {},
			},
			wantErr: ErrMissingTaskFunc,
		},
		{
			name: "Check for nil interval",
			task: &Task{
				TaskFunc: func() error { return nil },
				ErrFunc:  func(_ error) {},
			},
			wantErr: ErrInvalidInterval,
		},
	}

	for _, tc := range errorTests {
		t.Run(tc.name, func(t *testing.T) {
			id := tc.wantID
			var err error
			if tc.addWith {
				err = scheduler.AddWithID(tc.id, tc.task)
			} else {
				id, err = scheduler.Add(tc.task)
			}
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("expected %v, got %v", tc.wantErr, err)
			}
			if id != tc.wantID {
				t.Fatalf("expected ID %q, got %q", tc.wantID, id)
			}
			if tc.wantErr != nil {
				lookupID := tc.lookupID
				if lookupID == "" {
					lookupID = id
				}
				if _, lookupErr := scheduler.Lookup(lookupID); !errors.Is(lookupErr, ErrTaskNotFound) {
					t.Fatalf("expected ErrTaskNotFound for lookup %q, got %v", lookupID, lookupErr)
				}
			}
		})
	}

	t.Run("Add a valid task and look it up", func(t *testing.T) {
		id, err := scheduler.Add(&Task{
			Interval: time.Minute,
			TaskFunc: func() error { return nil },
			ErrFunc:  func(_ error) {},
		})
		if err != nil {
			t.Errorf("Unexpected errors when scheduling a valid task - %s", err)
		}

		_, err = scheduler.Lookup(id)
		if err != nil {
			t.Errorf("Unable to find newly scheduled task with Lookup - %s", err)
		}

		tt := scheduler.Tasks()
		if len(tt) < 1 {
			t.Errorf("Unable to find newly scheduled task with Tasks")
		}

	})

	t.Run("Add a valid task with an id and look it up", func(t *testing.T) {
		id := xid.New()
		err := scheduler.AddWithID(id.String(), &Task{
			Interval: time.Minute,
			TaskFunc: func() error { return nil },
			ErrFunc:  func(_ error) {},
		})
		if err != nil {
			t.Errorf("Unexpected errors when scheduling a valid task - %s", err)
		}

		_, err = scheduler.Lookup(id.String())
		if err != nil {
			t.Errorf("Unable to find newly scheduled task with Lookup - %s", err)
		}

		tt := scheduler.Tasks()
		if len(tt) < 1 {
			t.Errorf("Unable to find newly scheduled task with Tasks")
		}

	})

	t.Run("Add a invalid task with an duplicate id and look it up", func(t *testing.T) {
		// Setup A task
		id, err := scheduler.Add(&Task{
			Interval: time.Minute,
			TaskFunc: func() error { return nil },
			ErrFunc:  func(_ error) {},
		})
		if err != nil {
			t.Errorf("Unexpected errors when scheduling a valid task - %s", err)
		}

		err = scheduler.AddWithID(id, &Task{
			Interval: time.Minute,
			TaskFunc: func() error { return nil },
			ErrFunc:  func(_ error) {},
		})
		if !errors.Is(err, ErrIDInUse) {
			t.Errorf("Expected error for task with existing id")
		}

		_, err = scheduler.Lookup(id)
		if err != nil {
			t.Errorf("Unable to find previously scheduled task with Lookup - %s", err)
		}
	})

	t.Run("Duplicate id does not mutate caller task", func(t *testing.T) {
		id := xid.New().String()
		err := scheduler.AddWithID(id, &Task{
			Interval: time.Minute,
			TaskFunc: func() error { return nil },
		})
		if err != nil {
			t.Fatalf("Unexpected errors when scheduling a valid task - %s", err)
		}

		task := &Task{
			Interval:    time.Minute,
			TaskFunc:    func() error { return nil },
			TaskContext: TaskContext{Context: context.Background()},
		}

		err = scheduler.AddWithID(id, task)
		if !errors.Is(err, ErrIDInUse) {
			t.Fatalf("Expected duplicate id error, got %v", err)
		}

		if task.id != "" {
			t.Errorf("expected caller task id to remain empty, got %q", task.id)
		}
		if task.TaskContext.ID() != "" {
			t.Errorf(
				"expected caller task context id to remain empty, got %q",
				task.TaskContext.ID(),
			)
		}
		if task.ctx != nil {
			t.Errorf("expected caller task ctx to remain nil")
		}
		if task.cancel != nil {
			t.Errorf("expected caller task cancel to remain nil")
		}
		if task.timer != nil {
			t.Errorf("expected caller task timer to remain nil")
		}
	})

	t.Run("Lookup returns a reusable task definition without runtime state", func(t *testing.T) {
		sourceID := xid.New().String()
		err := scheduler.AddWithID(sourceID, &Task{
			Interval: time.Minute,
			TaskFunc: func() error { return nil },
		})
		if err != nil {
			t.Fatalf("Unexpected errors when scheduling source task - %s", err)
		}

		sourceTask, err := scheduler.Lookup(sourceID)
		if err != nil {
			t.Fatalf("Unexpected errors looking up source task - %s", err)
		}
		if sourceTask.id != "" {
			t.Fatalf("expected source task id to be omitted, got %q", sourceTask.id)
		}
		if sourceTask.TaskContext.ID() != "" {
			t.Fatalf(
				"expected source task context id to be omitted, got %q",
				sourceTask.TaskContext.ID(),
			)
		}
		if sourceTask.ctx != nil {
			t.Fatalf("expected source task context to be omitted")
		}
		if sourceTask.cancel != nil {
			t.Fatalf("expected source task cancel func to be omitted")
		}
		if sourceTask.timer != nil {
			t.Fatalf("expected source task timer to be omitted")
		}

		targetID := xid.New().String()
		err = scheduler.AddWithID(targetID, sourceTask)
		if err != nil {
			t.Fatalf("Unexpected errors when scheduling target task - %s", err)
		}
		defer scheduler.Del(targetID)
	})

	t.Run("Lookup missing task returns ErrTaskNotFound", func(t *testing.T) {
		const missingID = "missing-task"
		_, err := scheduler.Lookup(missingID)
		if !errors.Is(err, ErrTaskNotFound) {
			t.Fatalf("expected ErrTaskNotFound, got %v", err)
		}
		if !strings.Contains(err.Error(), missingID) {
			t.Errorf("expected error message to contain task ID %q, got %q", missingID, err.Error())
		}
	})
}

func TestScheduler(t *testing.T) {
	// Create a base scheduler to use
	scheduler := New()
	defer scheduler.Stop()

	t.Run("Verify Tasks Run when Added", func(t *testing.T) {
		// Channel for orchestrating when the task ran
		doneCh := make(chan struct{}, 6)

		// Setup A task
		id, err := scheduler.Add(&Task{
			Interval: testInterval,
			TaskFunc: func() error {
				doneCh <- struct{}{}
				return nil
			},
			ErrFunc: func(_ error) {},
		})
		if err != nil {
			t.Errorf("Unexpected errors when scheduling a valid task - %s", err)
		}
		defer scheduler.Del(id)

		// Make sure it runs especially when we want it too
		for i := 0; i < 6; i++ {
			select {
			case <-doneCh:
				continue
			case <-time.After(testTimeout):
				t.Errorf(
					"Scheduler failed to execute the scheduled tasks %d run within %v",
					i,
					testTimeout,
				)
			}
		}
	})

	t.Run("Verify TasksWithContext Run when Added", func(t *testing.T) {
		// Channel for orchestrating when the task ran
		doneCh := make(chan struct{}, 6)

		// User-defined context
		ctx, cancel := context.WithCancel(context.Background())

		// Setup A task
		id, err := scheduler.Add(&Task{
			Interval:    testInterval,
			TaskContext: TaskContext{Context: ctx},
			FuncWithTaskContext: func(_ TaskContext) error {
				cancel()
				return fmt.Errorf("Fake Error")
			},
			ErrFuncWithTaskContext: func(ctx TaskContext, _ error) {
				if ctx.Context != nil && ctx.Context.Err() == context.Canceled {
					doneCh <- struct{}{}
				}
			},
		})
		if err != nil {
			t.Errorf("Unexpected errors when scheduling a valid task - %s", err)
		}
		defer scheduler.Del(id)

		// Make sure it runs especially when we want it too
		for i := 0; i < 6; i++ {
			select {
			case <-doneCh:
				continue
			case <-time.After(testTimeout):
				t.Errorf(
					"Scheduler failed to execute the scheduled tasks %d run within %v",
					i,
					testTimeout,
				)
			}
		}
	})

	t.Run("Verify StartAfter works as expected", func(t *testing.T) {
		// Channel for orchestrating when the task ran
		doneCh := make(chan struct{}, 1)

		// Create a Start time
		sa := time.Now().Add(testStartDelay)

		// Setup A task
		id, err := scheduler.Add(&Task{
			Interval:   testInterval,
			StartAfter: sa,
			TaskFunc: func() error {
				doneCh <- struct{}{}
				return nil
			},
			ErrFunc: func(_ error) {},
		})
		if err != nil {
			t.Errorf("Unexpected errors when scheduling a valid task - %s", err)
		}
		defer scheduler.Del(id)

		// Make sure it runs especially when we want it too
		select {
		case <-doneCh:
			if time.Now().Before(sa) {
				t.Errorf(
					"Task executed before the defined start time now %s, supposed to be %s",
					time.Now().String(),
					sa.String(),
				)
			}
			return
		case <-time.After(testTimeout):
			t.Errorf("Scheduler failed to execute the scheduled tasks within %s", testTimeout)
		}
	})
}

func TestSchedulerDoesntRun(t *testing.T) {
	// Create a base scheduler to use
	scheduler := New()
	defer scheduler.Stop()

	t.Run("Verify Cancelling a StartAfter works as expected", func(t *testing.T) {
		// Channel for orchestrating when the task ran
		doneCh := make(chan struct{}, 1)

		// Create a Start time
		sa := time.Now().Add(testStartDelay)

		// Setup A task
		id, err := scheduler.Add(&Task{
			Interval:   testInterval,
			StartAfter: sa,
			TaskFunc: func() error {
				doneCh <- struct{}{}
				return nil
			},
			ErrFunc: func(_ error) {},
		})
		if err != nil {
			t.Errorf("Unexpected errors when scheduling a valid task - %s", err)
		}

		// Remove task before it can be scheduled
		scheduler.Del(id)

		// Make sure it doesn't run
		select {
		case <-doneCh:
			t.Errorf("Task executed it was supposed to be cancelled")
			return
		case <-time.After(testNoRunTimeout):
			return
		}
	})

	t.Run("Verify Tasks Dont run when Deleted", func(t *testing.T) {
		// Channel for orchestrating when the task ran
		doneCh := make(chan struct{}, 6)

		// Setup A task
		id, err := scheduler.Add(&Task{
			Interval: testInterval,
			TaskFunc: func() error {
				doneCh <- struct{}{}
				return nil
			},
			ErrFunc: func(_ error) {},
		})
		if err != nil {
			t.Errorf("Unexpected errors when scheduling a valid task - %s", err)
		}
		defer scheduler.Del(id)

		// Make sure it runs especially when we want it too
		for i := 0; i < 6; i++ {
			timeout := testTimeout
			if i > 2 {
				timeout = testNoRunTimeout
			}

			select {
			case <-doneCh:
				if i == 2 {
					scheduler.Del(id)
				}
				if i > 2 {
					t.Errorf("Task should not have exceeded 2, count is %d", i)
				}
				continue
			case <-time.After(timeout):
				if i > 2 {
					return
				}
				t.Errorf(
					"Scheduler failed to execute the scheduled tasks %d run within %v",
					i,
					timeout,
				)
			}
		}
	})

	t.Run("Verify Tasks Dont run when Deleted during execution", func(t *testing.T) {
		startedCtx, started := context.WithCancel(context.Background())
		releaseCtx, release := context.WithCancel(context.Background())
		finishedCtx, finished := context.WithCancel(context.Background())
		errCtx, errCancel := context.WithCancel(context.Background())
		defer errCancel()

		var runCount int32
		var deleted uint32

		id, err := scheduler.Add(&Task{
			Interval: testInterval,
			TaskFunc: func() error {
				currentRun := atomic.AddInt32(&runCount, 1)

				if currentRun == 1 {
					started()
					<-releaseCtx.Done()
					finished()
					return nil
				}

				if atomic.LoadUint32(&deleted) == 1 {
					return fmt.Errorf("task executed after delete, run=%d", currentRun)
				}

				return nil
			},
			ErrFunc: func(_ error) {
				errCancel()
			},
		})
		if err != nil {
			t.Fatalf("Unexpected errors when scheduling a valid task - %s", err)
		}
		defer scheduler.Del(id)

		select {
		case <-startedCtx.Done():
		case <-time.After(testTimeout):
			t.Fatalf("Scheduler failed to start the scheduled task")
		}

		atomic.StoreUint32(&deleted, 1)
		scheduler.Del(id)
		release()

		select {
		case <-finishedCtx.Done():
		case <-time.After(testTimeout):
			t.Fatalf("Task did not finish after delete")
		}

		select {
		case <-errCtx.Done():
			t.Fatalf("Task executed again after delete")
		case <-time.After(testNoRunTimeout):
		}
	})
}

func TestStartAfterTimerLifecycle(t *testing.T) {
	t.Run("Del stops delayed StartAfter timer", func(t *testing.T) {
		scheduler := New()
		defer scheduler.Stop()

		id, err := scheduler.Add(&Task{
			Interval:   testInterval,
			StartAfter: time.Now().Add(time.Hour),
			TaskFunc:   func() error { return nil },
		})
		if err != nil {
			t.Fatalf("Unexpected errors when scheduling a valid task - %s", err)
		}

		task := scheduledTask(t, scheduler, id)
		timer := taskTimer(t, task)

		scheduler.Del(id)

		if timer.Stop() {
			t.Fatalf("expected Del to stop delayed StartAfter timer")
		}
		if _, err := scheduler.Lookup(id); !errors.Is(err, ErrTaskNotFound) {
			t.Fatalf("expected deleted task lookup to return ErrTaskNotFound, got %v", err)
		}
	})

	t.Run("Stop stops delayed StartAfter timer", func(t *testing.T) {
		scheduler := New()

		id, err := scheduler.Add(&Task{
			Interval:   testInterval,
			StartAfter: time.Now().Add(time.Hour),
			TaskFunc:   func() error { return nil },
		})
		if err != nil {
			t.Fatalf("Unexpected errors when scheduling a valid task - %s", err)
		}

		task := scheduledTask(t, scheduler, id)
		timer := taskTimer(t, task)

		scheduler.Stop()

		if timer.Stop() {
			t.Fatalf("expected Stop to stop delayed StartAfter timer")
		}
		if _, err := scheduler.Lookup(id); !errors.Is(err, ErrTaskNotFound) {
			t.Fatalf("expected stopped task lookup to return ErrTaskNotFound, got %v", err)
		}
	})
}

func TestSchedulerExtras(t *testing.T) {
	// Create a base scheduler to use
	scheduler := New()
	defer scheduler.Stop()

	t.Run("Verify RunOnce works as expected", func(t *testing.T) {
		// Channel for orchestrating when the task ran
		doneCh := make(chan struct{}, 2)

		// Setup A task
		id, err := scheduler.Add(&Task{
			Interval: testInterval,
			RunOnce:  true,
			TaskFunc: func() error {
				doneCh <- struct{}{}
				return nil
			},
			ErrFunc: func(_ error) {},
		})
		if err != nil {
			t.Errorf("Unexpected errors when scheduling a valid task - %s", err)
		}
		defer scheduler.Del(id)

		// Make sure it runs especially when we want it too
		for i := 0; i < 6; i++ {
			timeout := testTimeout
			if i >= 1 {
				timeout = testNoRunTimeout
			}

			select {
			case <-doneCh:
				if i >= 1 {
					t.Errorf("Task should not have exceeded 1, count is %d", i)
				}
				continue
			case <-time.After(timeout):
				if i == 1 {
					return
				}
				t.Errorf(
					"Scheduler failed to execute the scheduled tasks %d run within %v",
					i,
					timeout,
				)
			}
		}
	})

	t.Run("Test ErrFunc gets called on errors", func(t *testing.T) {
		// Create a channel to signal function exec
		doneCh := make(chan struct{}, 1)

		// Add task
		id, err := scheduler.Add(&Task{
			Interval: testInterval,
			TaskFunc: func() error { return fmt.Errorf("Errors are bad") },
			ErrFunc:  func(_ error) { doneCh <- struct{}{} },
		})
		if err != nil {
			t.Errorf("Unexpected errors when scheduling a valid task - %s", err)
		}
		defer scheduler.Del(id)

		// Wait for success, or timeout
		select {
		case <-doneCh:
			return
		case <-time.After(testTimeout):
			t.Errorf("Error function was not called when an error occurred")
		}
	})
}

func TestSingleInstance(t *testing.T) {
	// Create a base scheduler to use
	scheduler := New()
	defer scheduler.Stop()

	// Create a counter to track how many times the task is called
	counter := NewCounter()

	// Create a second counter to track number of executions
	counter2 := NewCounter()

	// Create channels to signal task completion or failure.
	doneCh := make(chan struct{}, 4)
	errCh := make(chan error, 1)

	// Add a task that will increment the counter
	id, err := scheduler.Add(&Task{
		Interval:          testInterval,
		RunSingleInstance: true,
		TaskFunc: func() error {
			// Increment Concurrent Counter
			counter.Inc()
			if counter.Val() > 1 {
				return fmt.Errorf("Task ran more than once - count %d", counter.Val())
			}
			// Increment Execution Counter
			counter2.Inc()

			// Keep the task running long enough for overlapping ticks to be skipped.
			<-time.After(testInterval + testInterval/4)

			// Decrement Concurrent Counter
			counter.Dec()
			doneCh <- struct{}{}
			return nil
		},
		ErrFunc: func(e error) {
			errCh <- e
		},
	})
	if err != nil {
		t.Fatalf("Unexpected errors when scheduling task - %s", err)
	}
	defer scheduler.Del(id)

	for i := 0; i < 4; i++ {
		select {
		case <-doneCh:
		case e := <-errCh:
			t.Fatalf("Error function was called - %s", e)
		case <-time.After(testTimeout):
			t.Fatalf("Task was not called more than once successfully - count %d", counter2.Val())
		}
	}
}

func TestTaskPanicsReturnSentinelError(t *testing.T) {
	tests := []struct {
		name       string
		task       *Task
		errHandler string
		wantDetail string
	}{
		{
			name:       "TaskFunc panic",
			errHandler: "basic",
			wantDetail: "task func panic",
			task: &Task{
				Interval: testInterval,
				TaskFunc: func() error {
					panic("task func panic")
				},
			},
		},
		{
			name:       "TaskFunc panic nil",
			errHandler: "basic",
			wantDetail: "task panicked",
			task: &Task{
				Interval: testInterval,
				TaskFunc: func() error {
					panic(nil)
				},
			},
		},
		{
			name:       "FuncWithTaskContext panic",
			errHandler: "context",
			wantDetail: "task context panic",
			task: &Task{
				Interval:    testInterval,
				TaskContext: TaskContext{Context: context.Background()},
				FuncWithTaskContext: func(_ TaskContext) error {
					panic("task context panic")
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			scheduler := New()
			defer scheduler.Stop()

			doneCh := make(chan struct{}, 1)
			switch tc.errHandler {
			case "basic":
				tc.task.ErrFunc = func(err error) {
					if !errors.Is(err, ErrTaskPanic) {
						t.Errorf("expected ErrTaskPanic, got %v", err)
					}
					if !strings.Contains(err.Error(), tc.wantDetail) {
						t.Errorf("expected panic detail in error, got %q", err.Error())
					}
					doneCh <- struct{}{}
				}
			case "context":
				tc.task.ErrFuncWithTaskContext = func(taskCtx TaskContext, err error) {
					if taskCtx.Context == nil {
						t.Errorf("expected task context to be preserved")
					}
					if !errors.Is(err, ErrTaskPanic) {
						t.Errorf("expected ErrTaskPanic, got %v", err)
					}
					if !strings.Contains(err.Error(), tc.wantDetail) {
						t.Errorf("expected panic detail in error, got %q", err.Error())
					}
					doneCh <- struct{}{}
				}
			}

			_, err := scheduler.Add(tc.task)
			if err != nil {
				t.Fatalf("Unexpected errors when scheduling task - %s", err)
			}

			select {
			case <-doneCh:
			case <-time.After(testTimeout):
				t.Fatalf("expected panic to be reported through error callback")
			}
		})
	}
}

func TestErrFuncPanicsAreRecovered(t *testing.T) {
	tests := []struct {
		name       string
		task       *Task
		errHandler string
	}{
		{
			name:       "ErrFunc panic",
			errHandler: "basic",
			task: &Task{
				Interval: testInterval,
				TaskFunc: func() error {
					return fmt.Errorf("task failed")
				},
			},
		},
		{
			name:       "ErrFuncWithTaskContext panic",
			errHandler: "context",
			task: &Task{
				Interval:    testInterval,
				TaskContext: TaskContext{Context: context.Background()},
				FuncWithTaskContext: func(_ TaskContext) error {
					return fmt.Errorf("task context failed")
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			scheduler := New()
			defer scheduler.Stop()

			errHandled := make(chan struct{}, 1)
			followUpRan := make(chan struct{}, 1)

			switch tc.errHandler {
			case "basic":
				tc.task.ErrFunc = func(err error) {
					if err == nil {
						t.Errorf("expected non-nil error")
					}
					defer func() {
						errHandled <- struct{}{}
						panic("err func panic")
					}()
				}
			case "context":
				tc.task.ErrFuncWithTaskContext = func(_ TaskContext, err error) {
					if err == nil {
						t.Errorf("expected non-nil error")
					}
					defer func() {
						errHandled <- struct{}{}
						panic("err func with context panic")
					}()
				}
			}

			_, err := scheduler.Add(tc.task)
			if err != nil {
				t.Fatalf("Unexpected errors when scheduling task - %s", err)
			}

			select {
			case <-errHandled:
			case <-time.After(testTimeout):
				t.Fatalf("expected error handler to run")
			}

			_, err = scheduler.Add(&Task{
				Interval: testInterval,
				TaskFunc: func() error {
					followUpRan <- struct{}{}
					return nil
				},
			})
			if err != nil {
				t.Fatalf("Unexpected errors when scheduling follow-up task - %s", err)
			}

			select {
			case <-followUpRan:
			case <-time.After(testTimeout):
				t.Fatalf("expected scheduler to keep running after recovered panic")
			}
		})
	}
}

func scheduledTask(t *testing.T, scheduler *Scheduler, id string) *Task {
	t.Helper()

	scheduler.RLock()
	defer scheduler.RUnlock()

	task, ok := scheduler.tasks[id]
	if !ok {
		t.Fatalf("expected scheduler to contain task %q", id)
	}
	return task
}

func taskTimer(t *testing.T, task *Task) *time.Timer {
	t.Helper()

	task.RLock()
	defer task.RUnlock()

	if task.timer == nil {
		t.Fatalf("expected scheduled task to have a timer")
	}
	return task.timer
}
