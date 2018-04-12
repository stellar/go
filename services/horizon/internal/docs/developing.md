---
title: Horizon Development Guide
---

This document contains topics related to the development of horizon.

- [Regenerating generated code](#regen)
- [Running tests](#tests)
- [Logging](#logging)


---
## <a name="regen"></a> Regenerating generated code

Horizon uses two go tools you'll need to install:
1. [go-bindata](https://github.com/jteeuwen/go-bindata) is used to bundle test data
1. [go-codegen](https://github.com/nullstyle/go-codegen) is used to generate some boilerplate code

After the above are installed, simply run `go generate github.com/stellar/go/services/horizon/...`.

## <a name="tests"></a> Running Tests

start a redis server on port `6379`

```bash
redis-server
```

then, run the all the go tests like so (assuming you are at stellar/go):

```bash
bash ./support/scripts/run_tests
```

or just run horizon tests like so:

```bash
go test github.com/stellar/go/services/horizon/...
```

## <a name="logging"></a> Logging

All logging infrastructure is in the `github.com/stellar/go/tree/master/services/horizon/internal/log` package.  This package provides "level-based" logging:  Each logging statement has a severity, one of "Debug", "Info", "Warn", "Error" or "Panic".  The horizon server has a configured level "filter", specified either using the `--log-level` command line flag or the `LOG_LEVEL` environment variable.  When a logging statement is executed, the statements declared severity is checked against the filter and will only be emitted if the severity of the statement is equal or higher severity than the filter.

In addition, the logging subsystem has support for fields: Arbitrary key-value pairs that will be associated with an entry to allow for filtering and additional contextual information.

### Making Logging statements

Assuming that you've imports the log package, making a simple logging call is just:

```go

log.Info("my log line")
log.Infof("I take a %s", "format string")

```

Adding fields to a statement happens with a call to `WithField` or `WithFields`

```go
log.WithField("pid", 1234).Warn("i'm scared")

log.WithFields(log.F{
	"some_field": 123,
	"second_field": "hello",
}).Debug("here")
```

The return value from `WithField` or `WithFields` is a `*log.Entry`, which you can save to emit multiple logging
statements that all share the same field.  For example, the action system for horizon attaches a log entry to `action.Log` on every request that can be used to emit log entries that have the request's id attached as a field.

### Logging and Context

The logging package provides the root logger at `log.DefaultLogger` and the package level funcs such as `log.Info` operate against the default logger.  However, often it is important to include request-specific fields in a logging statement that are not available in the local scope.  For example, it is useful to include an http request's id in every log statement that is emitted by code running on behalf of the request.  This allows for easier debugging, as an operator can filter the log stream to a specific request id and not have to wade through the entirety of the log.

Unfortunately, it is not prudent to thread an `*http.Request` parameter to every downstream subroutine and so we need another way to make that information available.  The idiomatic way to do this is go is with a context parameter, as describe [on the go blog](https://blog.golang.org/context).  The logging provides a func to bind a logger to a context using `log.Set` and allows you to retrieve a bound logger using `log.Ctx(ctx)`.  Functions that need to log on behalf of an server request should take a context parameter.

Here's an example of using context:

```go

// create a new sublogger
sub := log.WithField("val", 1)

// bind it to a context
ctx := log.Set(context.Background(), sub)

log.Info("no fields on this statement")
log.Ctx(ctx).Info("This statement will use the sub logger")

```

### Logging Best Practices

It's recommended that you try to avoid contextual information in your logging messages.  Instead, use fields to establish context and use a static string for your message.  This practice allows horizon operators to more easily filter log lines to provide better insight into the health of the server.  Lets take an example:

```go
// BAD
log.Infof("running initializer: %s", i.Name)

//GOOD
log.WithField("init_name", i.Name).Info("running initializer")
```

With the "bad" form of the logging example above, an operator can filter on both the message as well as the initializer name independently.  This gets more powerful when multiple fields are combined, allowing for all sorts of slicing and dicing.


## <a name="TLS"></a> Enabling TLS on your local workstation

Horizon support HTTP/2 when served using TLS.  To enable TLS on your local workstation, you must generate a certificate and configure horizon to use it.  We've written a helper script at `tls/regen.sh` to make this simple.  Run the script from your terminal, and simply choose all the default options.  This will create two files: `tls/server.crt` and `tls/server.key`.  

Now you must configure horizon to use them: You can simply add `--tls-cert tls/server.crt --tls-key tls/server.key` to your command line invocations of horizon, or you may specify `TLS_CERT` and `TLS_KEY` environment variables.
