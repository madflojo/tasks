# Tasks

[![Build Status](https://travis-ci.org/madflojo/tasks.svg?branch=master)](https://travis-ci.org/madflojo/tasks) [![Coverage Status](https://coveralls.io/repos/github/madflojo/tasks/badge.svg?branch=master)](https://coveralls.io/github/madflojo/tasks?branch=master) [![Go Report Card](https://goreportcard.com/badge/github.com/madflojo/tasks)](https://goreportcard.com/report/github.com/madflojo/tasks) [![Documentation](https://godoc.org/github.com/madflojo/tasks?status.svg)](http://godoc.org/github.com/madflojo/tasks)

This package provides both recurring and one-time task execution. Tasks are run within their own goroutine which improves time accuracy for execution. This package also allows users to specify error handling though custom error functions.

Below is a simple example of starting the scheduler and registering a new task.

```go
// Start the Scheduler
scheduler := tasks.New()
defer scheduler.Stop()

// Add a task
id, err := scheduler.Add(&tasks.Task{
  Interval: time.Duration(30 * time.Second),
  TaskFunc: func() error {
    // Put your logic here
  }(),
  ErrFunc: func(err error) {
    // Put custom error handling here
  }(),
})
if err != nil {
  // Do Stuff
}
```

For simplicity this task scheduler uses the time.Duration type to specify intervals. This allows for a simple interface and flexible control over when tasks are executed.

The below example shows scheduling a task to run only once 30 days from now.

```go
// Define time to execute
t := time.Now().Add(30 * (24 * time.Hour))

// Add a one time only task for 30 days from now
id, err := scheduler.Add(&tasks.Task{
  Interval: time.Until(t),
  RunOnce:  true,
  TaskFunc: func() error {
    // Put your logic here
  }(),
  ErrFunc: func(err error) {
    // Put custom error handling here
  }(),
})
if err != nil {
  // Do Stuff
}
```
