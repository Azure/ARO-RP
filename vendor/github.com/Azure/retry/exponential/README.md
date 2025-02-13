# Exponential - The Exponential Backoff Package

[![GoDoc][godoc image]][godoc] [![Go Report Card](https://goreportcard.com/badge/github.com/Azure/retry)](https://goreportcard.com/report/github.com/Azure/retry)

## Introduction

This package provides an implementation of the exponential backoff algorithm.

[Exponential backoff][exponential backoff wiki]
is an algorithm that uses feedback to multiplicatively decrease the rate of some process,
in order to gradually find an acceptable rate.
The retries exponentially increase and stop increasing when a certain threshold is met.

This is a rewrite of an existing [package][cenkalti]. The original package works as intended, but I found that with the inclusions of generics in the latest version, it now has a lot of unnecessary function calls and return values that do similar things. This package is a rewrite of that package with the intention of being more efficient and easier to use. I also used this opportunity to add some features that I found useful.

Like that package, this package has its heritage from [Google's HTTP Client Library for Java][google-http-java-client].

## Usage

The import path is `github.com/Azure/retry/exponential`.

This package has a lot of different options, but can be used with the default settings like this:

```go
boff := exponential.New()

// Captured return data from the operation.
var data Data

// This sets the maximum time in the operation can be retried to 30 seconds.
// This is based on the parent context, so a cancel on the parent cancels
// this context.
ctx, cancel := context.WithTimeout(parentCtx, 30*time.Second)

err := boff.Retry(ctx, func(ctx context.Context, r Record) error {
	var err error
	data, err = getData(ctx)
	return err
})
cancel() // Always cancel the context when done to avoid lingering goroutines.
```

There are many different options for the backoff such as:

- Setting a custom `Policy` for the backoff.
- Logging backoff attempts with the `Record` object.
- Forcing the backoff to stop on permanent errors.
- Influence the backoff with a retry timer set to a specific time.
- Using a Transformer to deal with common errors like gRPC, HTTP, or SQL errors.
- Using the timetable tool to see the results of a custom backoff policy.
- ...

Use https://pkg.go.dev/github.com/Azure/retry/exponential to view the documentation.

[godoc]: https://pkg.go.dev/github.com/Azure/retry/exponential
[godoc image]: https://godoc.org/github.com/Azure/retry/exponential?status.png
[google-http-java-client]: https://github.com/google/google-http-java-client/blob/da1aa993e90285ec18579f1553339b00e19b3ab5/google-http-client/src/main/java/com/google/api/client/util/ExponentialBackOff.java
[exponential backoff wiki]: http://en.wikipedia.org/wiki/Exponential_backoff
[cenkalti]: https://github.com/cenkalti/backoff
