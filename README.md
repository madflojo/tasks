# Tasks

[![Coverage Status](https://coveralls.io/repos/github/madflojo/tasks/badge.svg?branch=main)](https://coveralls.io/github/madflojo/tasks?branch=main)
[![Go Report Card](https://goreportcard.com/badge/github.com/madflojo/tasks)](https://goreportcard.com/report/github.com/madflojo/tasks) 
[![PkgGoDev](https://pkg.go.dev/badge/github.com/madflojo/tasks)](https://pkg.go.dev/github.com/madflojo/tasks)

Package tasks is an easy to use in-process scheduler for recurring tasks in Go. Tasks is focused on high frequency
tasks that run quick, and often. The goal of Tasks is to support concurrent running tasks at scale without scheduler
induced jitter.

Tasks is focused on accuracy of task execution. To do this each task is called within it's own goroutine. This ensures 
that long execution of a single invocation does not throw the schedule as a whole off track.

For simplicity this task scheduler uses the time.Duration type to specify intervals. This allows for a simple interface 
and flexible control over when tasks are executed.

## Key Features

- **Concurrent Execution**: Tasks are executed in their own goroutines, ensuring accurate scheduling even when individual tasks take longer to complete.
- **Optimized Goroutine Scheduling**: Tasks leverages Go's `time.AfterFunc()` function to reduce sleeping goroutines and optimize CPU scheduling.
- **Flexible Task Intervals**: Tasks uses the `time.Duration` type to specify intervals, offering a simple interface and flexible control over task execution timing.
- **Delayed Task Start**: Schedule tasks to start at a later time by specifying a start time, allowing for greater control over task execution.
- **One-Time Tasks**: Schedule tasks to run only once by setting the `RunOnce` flag, ideal for single-use tasks or one-time actions.
- **Custom Error Handling**: Define a custom error handling function to handle errors returned by tasks, enabling tailored error handling logic.

## Usage

Here are some examples to help you get started with Tasks:

### Basic Usage

```go
// Start the Scheduler
scheduler := tasks.New()
defer scheduler.Stop()

// Add a task
id, err := scheduler.Add(&tasks.Task{
  Interval: 30 * time.Second,
  TaskFunc: func() error {
    // Put your logic here
  },
})
if err != nil {
  // Do Stuff
}
```

### Delayed Scheduling

Sometimes schedules need to started at a later time. This package provides the ability to start a task only after a 
certain time. The below example shows this in practice.

```go
// Add a recurring task for every 30 days, starting 30 days from now
id, err := scheduler.Add(&tasks.Task{
  Interval: 30 * (24 * time.Hour),
  StartAfter: time.Now().Add(30 * (24 * time.Hour)),
  TaskFunc: func() error {
    // Put your logic here
  },
})
if err != nil {
  // Do Stuff
}
```

### One-Time Tasks

It is also common for applications to run a task only once. The below example shows scheduling a task to run only once 
after waiting for 60 seconds.

```go
// Add a one time only task for 60 seconds from now
id, err := scheduler.Add(&tasks.Task{
  Interval: 60 * time.Second,
  RunOnce:  true,
  TaskFunc: func() error {
    // Put your logic here
  },
})
if err != nil {
  // Do Stuff
}
```

### Custom Error Handling

One powerful feature of Tasks is that it allows users to specify custom error handling. This is done by allowing users 
to define a function that is called when a task returns an error. The below example shows scheduling a task that logs 
when an error occurs.

```go
// Add a task with custom error handling
id, err := scheduler.Add(&tasks.Task{
  Interval: 30 * time.Second,
  TaskFunc: func() error {
    // Put your logic here
  },
  ErrFunc: func(e error) {
    log.Printf("An error occurred when executing task %s - %s", id, e)
  },
})
if err != nil {
  // Do Stuff
}
```

For more details on usage, see the [GoDoc](https://pkg.go.dev/github.com/madflojo/tasks).

## Contributing

Contributions are welcome! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for more details.
