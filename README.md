# Tasks

[![tests](https://github.com/madflojo/tasks/actions/workflows/tests.yml/badge.svg?branch=main)](https://github.com/madflojo/tasks/actions/workflows/tests.yml)
[![codecov](https://codecov.io/gh/madflojo/tasks/graph/badge.svg?token=882QTXA7PX)](https://codecov.io/gh/madflojo/tasks)
[![Go Report Card](https://goreportcard.com/badge/github.com/madflojo/tasks)](https://goreportcard.com/report/github.com/madflojo/tasks)
[![PkgGoDev](https://pkg.go.dev/badge/github.com/madflojo/tasks)](https://pkg.go.dev/github.com/madflojo/tasks)

A small in-process scheduler for Go tasks that need to run on time.

Tasks is built for recurring, quick-running jobs where scheduler-induced jitter needs to stay low and the scheduler
should stay out of the way. Each invocation runs in its own goroutine, so one slow task does not make the whole schedule
trip over its shoelaces.

Use Tasks when you want recurring work without cron wiring, a worker fleet, or a pile of scheduling boilerplate.
It is an in-process scheduler, not a durable queue or distributed job runner; if your process exits, your schedule exits
with it. That tradeoff keeps the package small, fast, and easy to reason about.

## Install

```bash
go get github.com/madflojo/tasks
```

## Why Tasks

- **Accurate recurring execution**: Each invocation runs independently, so a long-running task does not block unrelated schedules.
- **Small API**: Intervals use Go's `time.Duration`; no custom cron language required.
- **Delayed and one-time runs**: Use `StartAfter` and `RunOnce` for jobs that should begin later or run just once.
- **Overlap control**: Use `RunSingleInstance` to skip a run when the previous invocation is still working. No dogpiling.
- **Task context support**: Pass user-defined context and task metadata into callbacks with `FuncWithTaskContext`, `ErrFuncWithTaskContext`, and `TaskContext.ID()`.
- **Stable IDs and branchable errors**: Use `AddWithID` for deterministic identifiers and `errors.Is` for scheduler errors.

## Lifecycle Guarantees

Calling `Del` or `Stop` prevents delayed or future invocations, including tasks waiting on `StartAfter`. These methods do
not interrupt task functions that have already started; once your callback is running, it gets to finish its lap.

## Error Handling

Scheduler validation and lookup errors are exposed as sentinel errors so callers can branch with `errors.Is`:

- `ErrNilTask`
- `ErrIDInUse`
- `ErrMissingTaskFunc`
- `ErrInvalidInterval`
- `ErrTaskNotFound`
- `ErrTaskPanic`

```go
if errors.Is(err, tasks.ErrIDInUse) {
  // Pick another ID or update the existing task.
}
```

If a task callback panics, Tasks recovers it and reports `ErrTaskPanic` through the task error callback.
If the error callback itself panics, Tasks recovers and drops that panic.

## Usage

Here are some examples to help you get Tasks doing useful work without much ceremony.

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
    return nil
  },
})
if err != nil {
  // Handle error
}
```

### Delayed Scheduling

Sometimes schedules need to start later, not right now with a tiny starter pistol. Set `StartAfter` to delay the start
of a task's interval schedule. Deleting the task or stopping the scheduler before `StartAfter` prevents the delayed run
from being scheduled.

```go
// Add a recurring task for every 30 days, starting 30 days from now
id, err := scheduler.Add(&tasks.Task{
  Interval: 30 * (24 * time.Hour),
  StartAfter: time.Now().Add(30 * (24 * time.Hour)),
  TaskFunc: func() error {
    // Put your logic here
    return nil
  },
})
if err != nil {
  // Handle error
}
```

### One-Time Tasks

Some jobs only need one lap. The example below schedules a task to run once after waiting for 60 seconds.

```go
// Add a one-time task for 60 seconds from now
id, err := scheduler.Add(&tasks.Task{
  Interval: 60 * time.Second,
  RunOnce:  true,
  TaskFunc: func() error {
    // Put your logic here
    return nil
  },
})
if err != nil {
  // Handle error
}
```

### Custom Error Handling

Tasks lets callers define custom error handling with a callback that runs when a task returns an error. The example
below schedules a task that logs when things go sideways.

If both `ErrFunc` and `ErrFuncWithTaskContext` are set, `ErrFuncWithTaskContext` is used.

```go
// Add a task with custom error handling
id, err := scheduler.Add(&tasks.Task{
  Interval: 30 * time.Second,
  TaskFunc: func() error {
    // Put your logic here
    return nil
  },
  ErrFunc: func(e error) {
    log.Printf("An error occurred when executing task %s - %s", id, e)
  },
})
if err != nil {
  // Handle error
}
```

### Single-Instance Tasks

Use `RunSingleInstance` when a task might take longer than its interval and overlapping executions should be skipped.
No dogpiling, no duplicate workers stepping on each other.

```go
id, err := scheduler.Add(&tasks.Task{
  Interval:          30 * time.Second,
  RunSingleInstance: true,
  TaskFunc: func() error {
    // Put your logic here
    return nil
  },
})
if err != nil {
  // Handle error
}
```

### Task Context

Use the context-aware callbacks when you want to pass a user-defined context into task execution and error handling.

```go
ctx := context.Background()

id, err := scheduler.Add(&tasks.Task{
  Interval:    30 * time.Second,
  TaskContext: tasks.TaskContext{Context: ctx},
  FuncWithTaskContext: func(taskCtx tasks.TaskContext) error {
    log.Printf("running task %s", taskCtx.ID())
    return nil
  },
  ErrFuncWithTaskContext: func(taskCtx tasks.TaskContext, err error) {
    log.Printf("task %s failed: %v", taskCtx.ID(), err)
  },
})
if err != nil {
  // Handle error
}
```

### Custom Task IDs

Use `AddWithID` when you want to provide your own stable identifier for a task. Handy when "whatever ID the scheduler
picked" is not quite descriptive enough for future you.

```go
err := scheduler.AddWithID("nightly-report", &tasks.Task{
  Interval: time.Hour,
  TaskFunc: func() error {
    // Put your logic here
    return nil
  },
})
if err != nil {
  // Handle error
}
```

For more details on usage, see the [GoDoc](https://pkg.go.dev/github.com/madflojo/tasks).

## Contributing

Contributions are welcome! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for more details.

## Development

Common local workflows are available through the repository `Makefile`:

- `make build`
- `make tests`
- `make benchmarks`
- `make coverage`
- `make lint`
- `make format`
