# Writing MIMO Tasks

Writing a MIMO task consists of three major steps:

1. Writing the new functions in [`pkg/mimo/steps/`](../../pkg/mimo/steps/) which implement the specific behaviour (e.g. rotating a certificate), along with tests.
2. Writing the new Task in [`pkg/mimo/tasks/`](../../pkg/mimo/tasks/) which combines the Step you have written with any pre-existing "check" steps (e.g. `EnsureAPIServerIsUp`).
3. Adding the task with a new ID to [`pkg/mimo/const.go`](../../pkg/mimo/const.go) and `DEFAULT_MAINTENANCE_TASKS` in [`pkg/mimo/tasks/taskrunner.go`](../../pkg/mimo/tasks/taskrunner.go).

## New Step Functions

MIMO Step functions are similar to functions used in `pkg/cluster/install.go` but have additional information on the `Context` to prevent the explosion of struct members as seen in that package. Instead, the `GetTaskContext` function will return a `TaskContext` with various methods that can be used to retrieve information about the cluster, clients to perform actions in Azure, or Kubernetes clients to perform actions in the cluster.

Steps with similar logical domains should live in the same file/package. Currently, `pkg/mimo/steps/cluster/` is the only package, but functionality specific to the cluster's Azure resources may be better in a package called `pkg/mimo/steps/azure/` to make navigation easier.

Your base Action Step will look something like this:

```go
func DoSomething(ctx context.Context) error {
	tc, err := mimo.GetTaskContext(ctx)
	if err != nil {
		return mimo.TerminalError(err)
	}

    return nil
}
```

Like `pkg/cluster/`, you can also implement `Condition`s which allow you to wait for some state. However, MIMO's design is such that it should not sit around for long periods of time waiting for things which should already be the case -- for example, the API server not being up should instead be a usual Action which returns one of either `mimo.TerminalError` or `mimo.TransientError`.

`TransientError`s will be retried, and do not indicate a permanent failure. This is a good fit for errors that are possibly because of timeouts, random momentary outages, or cosmic winds flipping all the bits on your NIC for a nanosecond. MIMO will retry a task (at least, a few times) whose steps return a `TransientError`.

`TerminalError`s are used when there is no likelihood of automatic recovery. For example, if an API server is healthy and returning data, but it says that some essential OpenShift object that we require is missing, it is unlikely that object will return after one or many retries in a short period of time. These failures ought to require either manual intervention because they are unexpected or indicate that a cluster is unservicable. When a `TerminalError` is returned, it will cause the Task to hard fail and MIMO will not retry it.

## Testing

MIMO provides a fake `TaskContext`, created by `test/mimo/tasks.NewFakeTestContext`. This fake takes a number of mandatory items, such as an inner `Context` for cancellation, an `env.Interface`, a `*logrus.Entry`, and a stand-in clock for testing timing. Additional parts of the `TaskContext` used can be provided by `WithXXX` functions provided at the end of the instantiator, such as `WithClientHelper` to add a `ClientHelper` that is accessible on the `TaskContext`.

Attempting to use additional parts of the `TaskContext` without providing them will cause a panic or an error to be returned, in both the fake and real `TaskContext`. This behaviour is intended to make it clearer when some dependency is required.

## Assembling a Task

Once you have your Steps, you can assemble them into a Task in [`pkg/mimo/steps/`](../../pkg/mimo/steps/). See existing Tasks for examples.

## Assumptions MIMO Makes Of Your Code

- Your Steps may be run more than once -- both if they are in a Task more than once, or because a Task has been retried. Your Step must be resilient to being reran from a partial run.
- Steps should fail fast and not sit around unless they have caused something to happen. Right now, Tasks only have a 60 minute timeout total, so use it wisely.
- Steps use the `TaskContext` interface to get clients, and should not build them itself. If a Task requires a new client, it should be implemented in `TaskContext` to ensure that it can be tested the same way as other used clients.
