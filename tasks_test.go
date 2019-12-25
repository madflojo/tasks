package tasks

import (
	"fmt"
	"testing"
	"time"
)

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
		if err != nil {
			t.Errorf("Unable to find scheduled tasks - %s", err)
		}
		if len(tt) < 1 {
			t.Errorf("Unable to find newly scheduled task with Tasks")
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
