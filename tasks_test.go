package tasks

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/rs/xid"
)

type InterfaceTestCase struct {
	name   string
	task   *Task
	id     string
	addErr bool
}

func TestTasksInterface(t *testing.T) {
	var tt []InterfaceTestCase

	tt = append(tt, InterfaceTestCase{
		name: "Basic Valid Task",
		task: &Task{
			Interval: time.Duration(1 * time.Second),
			TaskFunc: func() error { return nil },
		},
	})

	tt = append(tt, InterfaceTestCase{
		name: "Basic Valid Task with ID",
		task: &Task{
			Interval: time.Duration(1 * time.Second),
			TaskFunc: func() error { return nil },
		},
		id: xid.New().String(),
	})

	tt = append(tt, InterfaceTestCase{
		name: "Valid Task with ErrFunc",
		task: &Task{
			Interval: time.Duration(1 * time.Second),
			TaskFunc: func() error { return nil },
			ErrFunc:  func(e error) {},
		},
	})

	tt = append(tt, InterfaceTestCase{
		name: "Valid Task with Context",
		task: &Task{
			Interval:    time.Duration(1 * time.Second),
			TaskFunc:    func() error { return nil },
			ErrFunc:     func(e error) {},
			TaskContext: TaskContext{Context: context.Background()},
		},
	})

	tt = append(tt, InterfaceTestCase{
		name: "Valid Task with Context and WithContextFuncs",
		task: &Task{
			Interval:               time.Duration(1 * time.Second),
			FuncWithTaskContext:    func(_ TaskContext) error { return nil },
			ErrFuncWithTaskContext: func(_ TaskContext, e error) {},
			TaskContext:            TaskContext{Context: context.Background()},
		},
	})

	tt = append(tt, InterfaceTestCase{
		name: "Valid Task without Context but WithContextFuncs",
		task: &Task{
			Interval:               time.Duration(1 * time.Second),
			FuncWithTaskContext:    func(_ TaskContext) error { return nil },
			ErrFuncWithTaskContext: func(_ TaskContext, e error) {},
		},
	})

	tt = append(tt, InterfaceTestCase{
		name: "Valid Task with StartAfter",
		task: &Task{
			Interval:   time.Duration(1 * time.Second),
			TaskFunc:   func() error { return nil },
			StartAfter: time.Now().Add(time.Duration(1 * time.Second)),
		},
	})

	tt = append(tt, InterfaceTestCase{
		name: "Valid Task with StartAfter but in the past",
		task: &Task{
			Interval:   time.Duration(1 * time.Second),
			TaskFunc:   func() error { return nil },
			StartAfter: time.Now().Add(time.Duration(-1 * time.Minute)),
		},
	})

	tt = append(tt, InterfaceTestCase{
		name: "Valid Task with RunOnce",
		task: &Task{
			Interval: time.Duration(1 * time.Second),
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
			Interval: time.Duration(1 * time.Second),
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
					if err != ErrIDInUse {
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

func TestAdd(t *testing.T) {
	// Create a base scheduler to use
	scheduler := New()
	defer scheduler.Stop()

	t.Run("Add a valid task and look it up", func(t *testing.T) {
		id, err := scheduler.Add(&Task{
			Interval: time.Duration(1 * time.Minute),
			TaskFunc: func() error { return nil },
			ErrFunc:  func(e error) {},
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
			Interval: time.Duration(1 * time.Minute),
			TaskFunc: func() error { return nil },
			ErrFunc:  func(e error) {},
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
		// Channel for orchestrating when the task ran
		doneCh := make(chan struct{})

		// Setup A task
		id, err := scheduler.Add(&Task{
			Interval: time.Duration(1 * time.Second),
			TaskFunc: func() error {
				doneCh <- struct{}{}
				return nil
			},
			ErrFunc: func(e error) {},
		})
		if err != nil {
			t.Errorf("Unexpected errors when scheduling a valid task - %s", err)
		}

		err = scheduler.AddWithID(id, &Task{
			Interval: time.Duration(1 * time.Minute),
			TaskFunc: func() error { return nil },
			ErrFunc:  func(e error) {},
		})
		if err != ErrIDInUse {
			t.Errorf("Expected error for task with existing id")
		}

		_, err = scheduler.Lookup(id)
		if err != nil {
			t.Errorf("Unable to find previously scheduled task with Lookup - %s", err)
		}
	})

	t.Run("Check for nil callback", func(t *testing.T) {
		_, err := scheduler.Add(&Task{
			Interval: time.Duration(1 * time.Minute),
			ErrFunc:  func(e error) {},
		})
		if err == nil {
			t.Errorf("Unexpected success when scheduling an invalid task - %s", err)
		}
	})

	t.Run("Check for nil interval", func(t *testing.T) {
		_, err := scheduler.Add(&Task{
			TaskFunc: func() error { return nil },
			ErrFunc:  func(e error) {},
		})
		if err == nil {
			t.Errorf("Unexpected success when scheduling an invalid task - %s", err)
		}
	})
}

func TestScheduler(t *testing.T) {
	// Create a base scheduler to use
	scheduler := New()

	t.Run("Verify Tasks Run when Added", func(t *testing.T) {
		// Channel for orchestrating when the task ran
		doneCh := make(chan struct{})

		// Setup A task
		id, err := scheduler.Add(&Task{
			Interval: time.Duration(1 * time.Second),
			TaskFunc: func() error {
				doneCh <- struct{}{}
				return nil
			},
			ErrFunc: func(e error) {},
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
			case <-time.After(2 * time.Second):
				t.Errorf("Scheduler failed to execute the scheduled tasks %d run within 2 seconds", i)
			}
		}
	})

	t.Run("Verify TasksWithContext Run when Added", func(t *testing.T) {
		// Channel for orchestrating when the task ran
		doneCh := make(chan struct{})

		// User-defined context
		ctx, cancel := context.WithCancel(context.Background())

		// Setup A task
		id, err := scheduler.Add(&Task{
			Interval:    time.Duration(1 * time.Second),
			TaskContext: TaskContext{Context: ctx},
			FuncWithTaskContext: func(_ TaskContext) error {
				cancel()
				return fmt.Errorf("Fake Error")
			},
			ErrFuncWithTaskContext: func(ctx TaskContext, e error) {
				if ctx.Context.Err() == context.Canceled {
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
			case <-time.After(2 * time.Second):
				t.Errorf("Scheduler failed to execute the scheduled tasks %d run within 2 seconds", i)
			}
		}
	})

	t.Run("Verify StartAfter works as expected", func(t *testing.T) {
		// Channel for orchestrating when the task ran
		doneCh := make(chan struct{})

		// Create a Start time
		sa := time.Now().Add(10 * time.Second)

		// Setup A task
		id, err := scheduler.Add(&Task{
			Interval:   time.Duration(1 * time.Second),
			StartAfter: sa,
			TaskFunc: func() error {
				doneCh <- struct{}{}
				return nil
			},
			ErrFunc: func(e error) {},
		})
		if err != nil {
			t.Errorf("Unexpected errors when scheduling a valid task - %s", err)
		}
		defer scheduler.Del(id)

		// Make sure it runs especially when we want it too
		select {
		case <-doneCh:
			if time.Now().Before(sa) {
				t.Errorf("Task executed before the defined start time now %s, supposed to be %s", time.Now().String(), sa.String())
			}
			return
		case <-time.After(15 * time.Second):
			t.Errorf("Scheduler failed to execute the scheduled tasks within 15 seconds")
		}
	})
}

func TestSchedulerDoesntRun(t *testing.T) {
	// Create a base scheduler to use
	scheduler := New()

	t.Run("Verify Cancelling a StartAfter works as expected", func(t *testing.T) {
		// Channel for orchestrating when the task ran
		doneCh := make(chan struct{})

		// Create a Start time
		sa := time.Now().Add(10 * time.Second)

		// Setup A task
		id, err := scheduler.Add(&Task{
			Interval:   time.Duration(1 * time.Second),
			StartAfter: sa,
			TaskFunc: func() error {
				doneCh <- struct{}{}
				return nil
			},
			ErrFunc: func(e error) {},
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
		case <-time.After(15 * time.Second):
			return
		}
	})

	t.Run("Verify Tasks Dont run when Deleted", func(t *testing.T) {
		// Channel for orchestrating when the task ran
		doneCh := make(chan struct{})

		// Setup A task
		id, err := scheduler.Add(&Task{
			Interval: time.Duration(1 * time.Second),
			TaskFunc: func() error {
				doneCh <- struct{}{}
				return nil
			},
			ErrFunc: func(e error) {},
		})
		if err != nil {
			t.Errorf("Unexpected errors when scheduling a valid task - %s", err)
		}
		defer scheduler.Del(id)

		// Make sure it runs especially when we want it too
		for i := 0; i < 6; i++ {
			select {
			case <-doneCh:
				if i == 2 {
					scheduler.Del(id)
				}
				if i > 2 {
					t.Errorf("Task should not have exceeded 2, count is %d", i)
				}
				continue
			case <-time.After(2 * time.Second):
				if i > 2 {
					return
				}
				t.Errorf("Scheduler failed to execute the scheduled tasks %d run within 2 seconds", i)
			}
		}
	})
}

func TestSchedulerExtras(t *testing.T) {
	// Create a base scheduler to use
	scheduler := New()

	t.Run("Verify RunOnce works as expected", func(t *testing.T) {
		// Channel for orchestrating when the task ran
		doneCh := make(chan struct{})

		// Setup A task
		id, err := scheduler.Add(&Task{
			Interval: time.Duration(1 * time.Second),
			RunOnce:  true,
			TaskFunc: func() error {
				doneCh <- struct{}{}
				return nil
			},
			ErrFunc: func(e error) {},
		})
		if err != nil {
			t.Errorf("Unexpected errors when scheduling a valid task - %s", err)
		}
		defer scheduler.Del(id)

		// Make sure it runs especially when we want it too
		for i := 0; i < 6; i++ {
			select {
			case <-doneCh:
				if i >= 1 {
					t.Errorf("Task should not have exceeded 1, count is %d", i)
				}
				continue
			case <-time.After(2 * time.Second):
				if i == 1 {
					return
				}
				t.Errorf("Scheduler failed to execute the scheduled tasks %d run within 2 seconds", i)
			}
		}
	})

	t.Run("Test ErrFunc gets called on errors", func(t *testing.T) {
		// Create a channel to signal function exec
		doneCh := make(chan struct{})

		// Add task
		_, err := scheduler.Add(&Task{
			Interval: time.Duration(1 * time.Second),
			TaskFunc: func() error { return fmt.Errorf("Errors are bad") },
			ErrFunc:  func(e error) { doneCh <- struct{}{} },
		})
		if err != nil {
			t.Errorf("Unexpected errors when scheduling a valid task - %s", err)
		}

		// Wait for success, or timeout
		select {
		case <-doneCh:
			return
		case <-time.After(2 * time.Second):
			t.Errorf("Error function was not called when an error occurred")
		}
	})
}
