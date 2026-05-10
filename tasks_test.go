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
)

const (
	testInterval     = 25 * time.Millisecond
	testStartDelay   = 75 * time.Millisecond
	testTimeout      = 2 * time.Second
	testNoRunTimeout = 150 * time.Millisecond
)

type executionTestCase struct {
	id        string
	ctx       context.Context
	task      *Task
	callsFunc bool
}

func TestAddValidation(t *testing.T) {
	tests := []struct {
		name     string
		task     *Task
		add      func(*Scheduler, *Task) (string, error)
		wantID   string
		wantErr  error
		lookupID string
	}{
		{
			name:    "Add nil task",
			add:     addTask,
			wantErr: ErrNilTask,
		},
		{
			name:     "AddWithID nil task",
			add:      addTaskWithID("nil-task"),
			wantID:   "nil-task",
			wantErr:  ErrNilTask,
			lookupID: "nil-task",
		},
		{
			name: "missing callback",
			task: &Task{
				Interval: time.Minute,
				ErrFunc:  func(_ error) {},
			},
			add:     addTask,
			wantErr: ErrMissingTaskFunc,
		},
		{
			name: "missing interval",
			task: &Task{
				TaskFunc: func() error { return nil },
				ErrFunc:  func(_ error) {},
			},
			add:     addTask,
			wantErr: ErrInvalidInterval,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			scheduler := newTestScheduler(t)

			id, err := tc.add(scheduler, tc.task)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("expected %v, got %v", tc.wantErr, err)
			}
			if id != tc.wantID {
				t.Fatalf("expected ID %q, got %q", tc.wantID, id)
			}

			lookupID := tc.lookupID
			if lookupID == "" {
				lookupID = id
			}
			if _, lookupErr := scheduler.Lookup(lookupID); !errors.Is(lookupErr, ErrTaskNotFound) {
				t.Fatalf("expected ErrTaskNotFound for lookup %q, got %v", lookupID, lookupErr)
			}
		})
	}
}

func TestAddAcceptsValidTaskShapes(t *testing.T) {
	tests := []struct {
		name string
		task *Task
		add  func(*Scheduler, *Task) (string, error)
	}{
		{
			name: "TaskFunc",
			task: validTask(),
			add:  addTask,
		},
		{
			name: "TaskFunc with custom ID",
			task: validTask(),
			add:  addTaskWithID("custom-task"),
		},
		{
			name: "ErrFunc",
			task: &Task{
				Interval: testInterval,
				TaskFunc: func() error { return nil },
				ErrFunc:  func(_ error) {},
			},
			add: addTask,
		},
		{
			name: "TaskContext",
			task: &Task{
				Interval:    testInterval,
				TaskFunc:    func() error { return nil },
				ErrFunc:     func(_ error) {},
				TaskContext: TaskContext{Context: context.Background()},
			},
			add: addTask,
		},
		{
			name: "FuncWithTaskContext with context",
			task: &Task{
				Interval:               testInterval,
				FuncWithTaskContext:    func(_ TaskContext) error { return nil },
				ErrFuncWithTaskContext: func(_ TaskContext, _ error) {},
				TaskContext:            TaskContext{Context: context.Background()},
			},
			add: addTask,
		},
		{
			name: "FuncWithTaskContext without context",
			task: &Task{
				Interval:               testInterval,
				FuncWithTaskContext:    func(_ TaskContext) error { return nil },
				ErrFuncWithTaskContext: func(_ TaskContext, _ error) {},
			},
			add: addTask,
		},
		{
			name: "future StartAfter",
			task: &Task{
				Interval:   testInterval,
				TaskFunc:   func() error { return nil },
				StartAfter: time.Now().Add(testInterval),
			},
			add: addTask,
		},
		{
			name: "past StartAfter",
			task: &Task{
				Interval:   testInterval,
				TaskFunc:   func() error { return nil },
				StartAfter: time.Now().Add(-time.Minute),
			},
			add: addTask,
		},
		{
			name: "RunOnce",
			task: &Task{
				Interval: testInterval,
				TaskFunc: func() error { return nil },
				RunOnce:  true,
			},
			add: addTask,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			scheduler := newTestScheduler(t)

			id, err := tc.add(scheduler, tc.task)
			if err != nil {
				t.Fatalf("expected add to succeed, got %v", err)
			}
			t.Cleanup(func() { scheduler.Del(id) })

			if _, err := scheduler.Lookup(id); err != nil {
				t.Fatalf("expected task %q to be found, got %v", id, err)
			}
			if tasks := scheduler.Tasks(); len(tasks) != 1 {
				t.Fatalf("expected one scheduled task, got %d", len(tasks))
			}
		})
	}
}

func TestAddWithIDRejectsDuplicateID(t *testing.T) {
	scheduler := newTestScheduler(t)
	id := mustAddTask(t, scheduler, validTask())

	err := scheduler.AddWithID(id, validTask())
	if !errors.Is(err, ErrIDInUse) {
		t.Fatalf("expected ErrIDInUse, got %v", err)
	}
	if _, err := scheduler.Lookup(id); err != nil {
		t.Fatalf("expected original task to remain scheduled, got %v", err)
	}
}

func TestTaskExecution(t *testing.T) {
	// Create a base scheduler to use
	scheduler := New()
	defer scheduler.Stop()

	cases := []struct {
		name string
		new  func(*testing.T) executionTestCase
	}{
		{
			name: "Valid Task",
			new: func(_ *testing.T) executionTestCase {
				ctx, cancel := context.WithCancel(context.Background())
				return executionTestCase{
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
			new: func(t *testing.T) executionTestCase {
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

				return executionTestCase{
					ctx:       ctx,
					task:      task,
					callsFunc: true,
				}
			},
		},
		{
			name: "Cancel a Task before it's called",
			new: func(_ *testing.T) executionTestCase {
				ctx, cancel := context.WithCancel(context.Background())
				return executionTestCase{
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
			new: func(t *testing.T) executionTestCase {
				ctx, cancel := context.WithCancel(context.Background())
				return executionTestCase{
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
			new: func(t *testing.T) executionTestCase {
				ctx, cancel := context.WithCancel(context.Background())
				return executionTestCase{
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
			new: func(t *testing.T) executionTestCase {
				ctx, cancel := context.WithCancel(context.Background())
				const id = "test-id"
				return executionTestCase{
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
			new: func(t *testing.T) executionTestCase {
				ctx, cancel := context.WithCancel(context.Background())
				startAfter := time.Now().Add(testStartDelay)
				return executionTestCase{
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

func TestDuplicateIDDoesNotMutateCallerTask(t *testing.T) {
	scheduler := newTestScheduler(t)
	const id = "duplicate-id"

	if err := scheduler.AddWithID(id, validTask()); err != nil {
		t.Fatalf("expected initial add to succeed, got %v", err)
	}

	task := &Task{
		Interval:    time.Minute,
		TaskFunc:    func() error { return nil },
		TaskContext: TaskContext{Context: context.Background()},
	}

	err := scheduler.AddWithID(id, task)
	if !errors.Is(err, ErrIDInUse) {
		t.Fatalf("expected ErrIDInUse, got %v", err)
	}

	assertTaskHasNoRuntimeState(t, task)
}

func TestLookupReturnsReusableTaskDefinition(t *testing.T) {
	scheduler := newTestScheduler(t)
	const sourceID = "source-task"
	const targetID = "target-task"

	if err := scheduler.AddWithID(sourceID, validTask()); err != nil {
		t.Fatalf("expected source add to succeed, got %v", err)
	}

	sourceTask, err := scheduler.Lookup(sourceID)
	if err != nil {
		t.Fatalf("expected source lookup to succeed, got %v", err)
	}
	assertTaskHasNoRuntimeState(t, sourceTask)

	if err := scheduler.AddWithID(targetID, sourceTask); err != nil {
		t.Fatalf("expected lookup task to be reusable, got %v", err)
	}
}

func TestLookupMissingTaskReturnsErrTaskNotFound(t *testing.T) {
	scheduler := newTestScheduler(t)
	const missingID = "missing-task"

	_, err := scheduler.Lookup(missingID)
	if !errors.Is(err, ErrTaskNotFound) {
		t.Fatalf("expected ErrTaskNotFound, got %v", err)
	}
	if !strings.Contains(err.Error(), missingID) {
		t.Errorf("expected error message to contain task ID %q, got %q", missingID, err.Error())
	}
}

func TestAddWithIDRejectsEmptyID(t *testing.T) {
	scheduler := newTestScheduler(t)

	err := scheduler.AddWithID("", validTask())
	if !errors.Is(err, ErrInvalidID) {
		t.Fatalf("expected ErrInvalidID, got %v", err)
	}
	if tasks := scheduler.Tasks(); len(tasks) != 0 {
		t.Fatalf("expected no scheduled tasks, got %d", len(tasks))
	}
	if _, err := scheduler.Lookup(""); !errors.Is(err, ErrTaskNotFound) {
		t.Fatalf("expected Lookup(\"\") to return ErrTaskNotFound, got %v", err)
	}
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

		assertNoSignal(t, doneCh, testNoRunTimeout, "Task executed it was supposed to be cancelled")
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
	tests := []struct {
		name string
		stop func(*Scheduler, string)
	}{
		{
			name: "Del",
			stop: func(scheduler *Scheduler, id string) {
				scheduler.Del(id)
			},
		},
		{
			name: "Stop",
			stop: func(scheduler *Scheduler, _ string) {
				scheduler.Stop()
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name+" stops delayed StartAfter timer", func(t *testing.T) {
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

			tc.stop(scheduler, id)

			if timer.Stop() {
				t.Fatalf("expected %s to stop delayed StartAfter timer", tc.name)
			}
			if _, err := scheduler.Lookup(id); !errors.Is(err, ErrTaskNotFound) {
				t.Fatalf("expected stopped task lookup to return ErrTaskNotFound, got %v", err)
			}
		})
	}
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

		waitForSignal(t, doneCh, testTimeout, "Error function was not called when an error occurred")
	})
}

func TestSingleInstance(t *testing.T) {
	scheduler := New()
	defer scheduler.Stop()

	var concurrentRuns int32
	var completedRuns int32
	doneCh := make(chan struct{}, 4)
	errCh := make(chan error, 1)

	id, err := scheduler.Add(&Task{
		Interval:          testInterval,
		RunSingleInstance: true,
		TaskFunc: func() error {
			currentRuns := atomic.AddInt32(&concurrentRuns, 1)
			if currentRuns > 1 {
				return fmt.Errorf("task ran concurrently, count %d", currentRuns)
			}
			atomic.AddInt32(&completedRuns, 1)

			// Keep the task running long enough for overlapping ticks to be skipped.
			<-time.After(testInterval + testInterval/4)

			atomic.AddInt32(&concurrentRuns, -1)
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
			t.Fatalf(
				"Task was not called more than once successfully - count %d",
				atomic.LoadInt32(&completedRuns),
			)
		}
	}
}

func TestSingleInstanceWaitsForErrorHandler(t *testing.T) {
	scheduler := New()
	defer scheduler.Stop()

	errStarted := make(chan struct{})
	releaseErr := make(chan struct{})
	var runCount int32
	errStartedOnce := sync.Once{}
	releaseErrOnce := sync.Once{}
	t.Cleanup(func() {
		releaseErrOnce.Do(func() {
			close(releaseErr)
		})
	})

	id, err := scheduler.Add(&Task{
		Interval:          testInterval,
		RunSingleInstance: true,
		TaskFunc: func() error {
			atomic.AddInt32(&runCount, 1)
			return fmt.Errorf("task failed")
		},
		ErrFunc: func(error) {
			errStartedOnce.Do(func() {
				close(errStarted)
			})
			<-releaseErr
		},
	})
	if err != nil {
		t.Fatalf("Unexpected errors when scheduling task - %s", err)
	}
	defer scheduler.Del(id)

	waitForSignal(t, errStarted, testTimeout, "Error handler did not run")

	<-time.After(testNoRunTimeout)

	if got := atomic.LoadInt32(&runCount); got != 1 {
		t.Fatalf("expected single task run while error handler is running, got %d", got)
	}

	releaseErrOnce.Do(func() {
		close(releaseErr)
	})
}

func TestRunOnceDeletesAfterErrorHandler(t *testing.T) {
	scheduler := New()
	defer scheduler.Stop()

	errStarted := make(chan struct{})
	releaseErr := make(chan struct{})
	errStartedOnce := sync.Once{}
	releaseErrOnce := sync.Once{}
	t.Cleanup(func() {
		releaseErrOnce.Do(func() {
			close(releaseErr)
		})
	})

	id, err := scheduler.Add(&Task{
		Interval: testInterval,
		RunOnce:  true,
		TaskFunc: func() error {
			return fmt.Errorf("task failed")
		},
		ErrFunc: func(error) {
			errStartedOnce.Do(func() {
				close(errStarted)
			})
			<-releaseErr
		},
	})
	if err != nil {
		t.Fatalf("Unexpected errors when scheduling task - %s", err)
	}
	defer scheduler.Del(id)

	waitForSignal(t, errStarted, testTimeout, "Error handler did not run")

	if _, err := scheduler.Lookup(id); err != nil {
		t.Fatalf("expected RunOnce task to remain scheduled while error handler is running: %v", err)
	}

	releaseErrOnce.Do(func() {
		close(releaseErr)
	})

	eventually(t, testTimeout, func() bool {
		_, err := scheduler.Lookup(id)
		return errors.Is(err, ErrTaskNotFound)
	}, "expected RunOnce task to self delete after error handler returns")
}

func TestTaskPanicsReturnSentinelError(t *testing.T) {
	tests := []struct {
		name             string
		task             *Task
		wantDetail       string
		attachErrHandler func(*testing.T, *Task, chan struct{}, string)
	}{
		{
			name:       "TaskFunc",
			wantDetail: "task func panic",
			task: &Task{
				Interval: testInterval,
				TaskFunc: func() error {
					panic("task func panic")
				},
			},
			attachErrHandler: attachPanicErrFunc,
		},
		{
			name:       "FuncWithTaskContext",
			wantDetail: "task context panic",
			task: &Task{
				Interval:    testInterval,
				TaskContext: TaskContext{Context: context.Background()},
				FuncWithTaskContext: func(_ TaskContext) error {
					panic("task context panic")
				},
			},
			attachErrHandler: attachPanicErrFuncWithTaskContext,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			scheduler := New()
			defer scheduler.Stop()

			doneCh := make(chan struct{}, 1)
			tc.attachErrHandler(t, tc.task, doneCh, tc.wantDetail)

			_, err := scheduler.Add(tc.task)
			if err != nil {
				t.Fatalf("Unexpected errors when scheduling task - %s", err)
			}

			waitForSignal(
				t,
				doneCh,
				testTimeout,
				"expected panic to be reported through error callback",
			)
		})
	}
}

func TestErrFuncPanicsAreRecovered(t *testing.T) {
	tests := []struct {
		name             string
		task             *Task
		attachErrHandler func(*testing.T, *Task, chan struct{})
	}{
		{
			name: "ErrFunc",
			task: &Task{
				Interval: testInterval,
				TaskFunc: func() error {
					return fmt.Errorf("task failed")
				},
			},
			attachErrHandler: attachPanickingErrFunc,
		},
		{
			name: "ErrFuncWithTaskContext",
			task: &Task{
				Interval:    testInterval,
				TaskContext: TaskContext{Context: context.Background()},
				FuncWithTaskContext: func(_ TaskContext) error {
					return fmt.Errorf("task context failed")
				},
			},
			attachErrHandler: attachPanickingErrFuncWithTaskContext,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			scheduler := New()
			defer scheduler.Stop()

			errHandled := make(chan struct{}, 1)
			followUpRan := make(chan struct{}, 1)

			tc.attachErrHandler(t, tc.task, errHandled)

			_, err := scheduler.Add(tc.task)
			if err != nil {
				t.Fatalf("Unexpected errors when scheduling task - %s", err)
			}

			waitForSignal(t, errHandled, testTimeout, "expected error handler to run")

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

			waitForSignal(
				t,
				followUpRan,
				testTimeout,
				"expected scheduler to keep running after recovered panic",
			)
		})
	}
}

func scheduledTask(t *testing.T, scheduler *Scheduler, id string) *Task {
	t.Helper()

	scheduler.mu.RLock()
	defer scheduler.mu.RUnlock()

	task, ok := scheduler.tasks[id]
	if !ok {
		t.Fatalf("expected scheduler to contain task %q", id)
	}
	return task
}

func taskTimer(t *testing.T, task *Task) *time.Timer {
	t.Helper()

	task.mu.RLock()
	defer task.mu.RUnlock()

	if task.timer == nil {
		t.Fatalf("expected scheduled task to have a timer")
	}
	return task.timer
}

func eventually(t *testing.T, timeout time.Duration, condition func() bool, failure string) {
	t.Helper()

	deadline := time.After(timeout)
	ticker := time.NewTicker(testInterval)
	defer ticker.Stop()

	for {
		if condition() {
			return
		}

		select {
		case <-deadline:
			t.Fatal(failure)
		case <-ticker.C:
		}
	}
}

func waitForSignal(t *testing.T, ch <-chan struct{}, timeout time.Duration, failure string) {
	t.Helper()

	select {
	case <-ch:
	case <-time.After(timeout):
		t.Fatal(failure)
	}
}

func assertNoSignal(t *testing.T, ch <-chan struct{}, timeout time.Duration, failure string) {
	t.Helper()

	select {
	case <-ch:
		t.Fatal(failure)
	case <-time.After(timeout):
	}
}

func newTestScheduler(t *testing.T) *Scheduler {
	t.Helper()

	scheduler := New()
	t.Cleanup(scheduler.Stop)
	return scheduler
}

func validTask() *Task {
	return &Task{
		Interval: testInterval,
		TaskFunc: func() error { return nil },
	}
}

func addTask(scheduler *Scheduler, task *Task) (string, error) {
	return scheduler.Add(task)
}

func addTaskWithID(id string) func(*Scheduler, *Task) (string, error) {
	return func(scheduler *Scheduler, task *Task) (string, error) {
		return id, scheduler.AddWithID(id, task)
	}
}

func mustAddTask(t *testing.T, scheduler *Scheduler, task *Task) string {
	t.Helper()

	id, err := scheduler.Add(task)
	if err != nil {
		t.Fatalf("expected add to succeed, got %v", err)
	}
	t.Cleanup(func() { scheduler.Del(id) })
	return id
}

func assertTaskHasNoRuntimeState(t *testing.T, task *Task) {
	t.Helper()

	if task.id != "" {
		t.Errorf("expected task id to remain empty, got %q", task.id)
	}
	if task.TaskContext.ID() != "" {
		t.Errorf("expected task context id to remain empty, got %q", task.TaskContext.ID())
	}
	if task.ctx != nil {
		t.Errorf("expected task ctx to remain nil")
	}
	if task.cancel != nil {
		t.Errorf("expected task cancel to remain nil")
	}
	if task.timer != nil {
		t.Errorf("expected task timer to remain nil")
	}
}

func attachPanicErrFunc(t *testing.T, task *Task, doneCh chan struct{}, wantDetail string) {
	t.Helper()

	task.ErrFunc = func(err error) {
		assertTaskPanicError(t, err, wantDetail)
		doneCh <- struct{}{}
	}
}

func attachPanicErrFuncWithTaskContext(
	t *testing.T,
	task *Task,
	doneCh chan struct{},
	wantDetail string,
) {
	t.Helper()

	task.ErrFuncWithTaskContext = func(taskCtx TaskContext, err error) {
		if taskCtx.Context == nil {
			t.Errorf("expected task context to be preserved")
		}
		assertTaskPanicError(t, err, wantDetail)
		doneCh <- struct{}{}
	}
}

func assertTaskPanicError(t *testing.T, err error, wantDetail string) {
	t.Helper()

	if !errors.Is(err, ErrTaskPanic) {
		t.Errorf("expected ErrTaskPanic, got %v", err)
	}
	if !strings.Contains(err.Error(), wantDetail) {
		t.Errorf("expected panic detail in error, got %q", err.Error())
	}
}

func attachPanickingErrFunc(t *testing.T, task *Task, errHandled chan struct{}) {
	t.Helper()

	task.ErrFunc = func(err error) {
		if err == nil {
			t.Errorf("expected non-nil error")
		}
		defer func() {
			errHandled <- struct{}{}
			panic("err func panic")
		}()
	}
}

func attachPanickingErrFuncWithTaskContext(t *testing.T, task *Task, errHandled chan struct{}) {
	t.Helper()

	task.ErrFuncWithTaskContext = func(_ TaskContext, err error) {
		if err == nil {
			t.Errorf("expected non-nil error")
		}
		defer func() {
			errHandled <- struct{}{}
			panic("err func with context panic")
		}()
	}
}
