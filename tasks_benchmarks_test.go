package tasks

import (
	"testing"
	"time"
)

func BenchmarkTasks(b *testing.B) {
	// Create base scheduler to use
	scheduler := New()
	defer scheduler.Stop()

	// Setup a single task for re-use
	taskID, err := scheduler.Add(&Task{
		Interval: time.Duration(1 * time.Minute),
		TaskFunc: func() error { return nil },
		ErrFunc:  func(e error) {},
	})
	if err != nil {
		b.Fatalf("Unable to schedule example task - %s", err)
	}
	defer scheduler.Del(taskID)

	// Grab example for re-use
	exampleTask, err := scheduler.Lookup(taskID)
	if err != nil {
		b.Fatalf("Unable to lookup newly added task - %s", err)
	}

	b.Run("Adding a scheduler", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := scheduler.Add(exampleTask)
			if err != nil {
				b.Fatalf("Unable to add new scheduled task - %s", err)
			}
		}
	})

	// Clear Excess Tasks
	for id := range scheduler.Tasks() {
		if id == taskID {
			continue
		}
		scheduler.Del(id)
	}

	b.Run("Looking up a scheduled task", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := scheduler.Lookup(taskID)
			if err != nil {
				b.Fatalf("Unable to lookup scheduled tasks - %s", err)
			}
		}
	})
}
