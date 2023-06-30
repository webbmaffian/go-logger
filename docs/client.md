# Client usage
These examples assumes that you already have:
- a private key
- a client certificate signed by CA
- the root CA certificate.

## Setup

First of, we need to load the key and certificates from files.
```go
var (
	key  auth.PrivateKey
	cert auth.Certificate
	root auth.Certificate
)

if err = key.FromFile("private.key"); err != nil {
	// Handle error
}

if err = cert.FromFile("certificate.pem"); err != nil {
	// Handle error
}

if err = root.FromFile("root.pem"); err != nil {
	// Handle error
}
```

Then, we need to initialize the client. A file will be created (with the size of `2^16 x BufferSize` bytes) and used as buffer. If logs entries are written faster than they are sent, up to `BufferSize` entries will be queued up until they are starting to get overwritten. A.k.a. "leaky bucket".
```go
cli, err := peer.NewTlsClient(ctx, peer.TlsClientOptions{
	Address:        "localhost:4610",
	PrivateKey:     key,
	Certificate:    cert,
	RootCa:         root,

	// These are optional - the values are the defaults.
	BufferFilepath: "logs.bin",
	BufferSize:     100
})

if err != nil {
	// Handle error
}
```

After that, we initialize a pool with our settings.
```go
pool, err := logger.NewPool(cli, logger.PoolOptions{
	BucketId:           123456,
	DefaultEntryTTL:    30, // days
	DefaultMetaTTL:     30, // days
})

if err != nil {
	// Handle error
}
```

## Acquiring a logger
A fresh logger can be acquired from the pool.
```go
log := pool.Logger()
```

You can apply settings on a logger. Those settings will be applied as default values to all entries created from the logger.
```go
log.Cat(123)
log.Tag("checkout")
log.TTL(90)
```

As all setting methods are chainable, the following does exactly the same thing.
```go
log.Cat(123).Tag("checkout").TTL(90)
```

You can also acquire a logger from another logger, hence inheriting all its current settings.
```go
log2 := log.Logger()
```

When you are done with the logger, you can drop it back to the pool. Despite not required, it will decrease allocations and increase performance.
```go
log2.Drop()
```

All these concepts are very useful for log contexts. You have a function that acquires its own logger, set context values to it, use it for logging, and then drop it back to the pool once finished.
```go
func doSomething(log *logger.Logger) {
	l = log.Logger().Cat(123).Tag("checkout").TTL(90)
	defer l.Drop()

	// Do something
}
```

## Generating log entries
Using any logger, you can create log entries of chosen severity. You must always end with sending it (which will write the entry to the buffer and then put it back to its pool, hence minimizing allocations).
```go
log.Info("created user").Send()
```

The first method in the chain is the severity, and it also accepts interpolated tags - printf-style.
```go
log.Info("created user %s", "foo@bar.baz").Send()
```

The `Send` method returns a unique XID of the entry, which can be handy if you want to give an error ID to the user.
```go
errorId := log.Err("something happened").Send()

// Show the error ID to the user
```

There are plenty of methods for adding data to a log entry, and all are chainable.
```go
log.Notice("lorem ipsum dolor sit amet").Cat(3).Meta("foobar", "baz").Trace().TTL(5).Send()
```

If we already have an error, we can send it directly to a logger. This will create a log entry and transfer the error message to it, and then send it.
```go
log.Send(err)
```

As the `Entry` type implements the `Error` interface, we can also return it just like a regular error, and then send it with `Logger.Send`. The method will recognize that it already is a log entry and send it directly, keeping the original XID.
```go
func doSomething(log *logger.Logger) error {
	// Oh no - an error occured
	return log.Err("opsie")
}

if err := doSomething(log); err != nil {
	log.Send(err)
}
```