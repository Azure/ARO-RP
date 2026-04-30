# selenium-until-extra

[![GoDoc](https://godoc.org/github.com/serge1peshcoff/selenium-go-conditions?status.svg)](https://godoc.org/github.com/serge1peshcoff/selenium-go-conditions)
[![Travis](https://img.shields.io/travis/serge1peshcoff/selenium-go-conditions.svg)](https://travis-ci.org/serge1peshcoff/selenium-go-conditions)
[![Code Climate](https://codeclimate.com/github/serge1peshcoff/selenium-go-conditions/badges/gpa.svg)](https://codeclimate.com/github/serge1peshcoff/selenium-go-conditions)
[![Issue Count](https://codeclimate.com/github/serge1peshcoff/selenium-go-conditions/badges/issue_count.svg)](https://codeclimate.com/github/serge1peshcoff/selenium-go-conditions)
[![Test Coverage](https://codeclimate.com/github/serge1peshcoff/selenium-go-conditions/badges/coverage.svg)](https://codeclimate.com/github/serge1peshcoff/selenium-go-conditions/coverage)


A library that provides conditions for `WebDriver.Wait()` function from `github.com/tebeka/selenium` package.

## Motivation

Recently `WebDriver.Wait()` function was added to Golang's Selenium binding (I've implemented it and I've made a PR on that, so I kind of know how it's working). It allows waiting for some condition to be true.

There was a decision to implement only `WebDriver.Wait()` in this `github.com/tebeka/selenium` package, and leave the implementation of conditions to another library. So, here it is!

## How to use it

First you download it with:

```sh
go get github.com/serge1peshcoff/selenium-go-condition
```

Then you import it within your package with

```go
import "github.com/serge1peshcoff/selenium-go-conditions"
```

and then you use the `conditions` package. See `examples/example.go` for a complete example.

## API

The API is available at https://godoc.org/github.com/serge1peshcoff/selenium-go-conditions

## How does it work

There is a `type Condition func (selenium.WebDriver) (bool, error)` and `WebDriver.Wait(cond Condition, timeout, interval time.Duration)` in Golang's Selenium binding. The `WebDriver.Wait()`'s implementaion is pretty simple: it starts an endless loop, and return either `nil` if the condition would evaluate to true, or `error` if there would be an error executing a condition or on timeout.

So, what this package does is provides functions that returns `Condition`s, so you would be able to pass it as an argument for `WebDriver.Wait()` function.



## Contribution

All issues and PRs are welcomed and appreciated!

If you want to suggest something new, you can make an issue about that, and we'll figure that out!

### Testing

Please make sure that all tests are passing before submitting a PR and that code coverage is good enough.

Before running tests:
- install `firefox` and `java` (to run `selenium` server)
- run `testing/setup.sh`, that would download `selenium` webserver and `geckodriver` and copy these files into necesssary folders. If you won't have `testing/selenium-server-standalone-3.5.1.jar` file, it would probably crash.

Then run tests with `go test`.
You can also get the code coverage with:

```sh
go test -coverprofile=coverage.out
go tool cover -func=coverage.out
```

Also, the Travis CI setup is run at each push to the repository, so please make sure that your build is passing.