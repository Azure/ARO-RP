/*
Package base provides an audit client that can be used to send audit records to an audit server. It is preferred
to use the audit/ package to connect to an audit server. This package is provided for those who need more
control over how the audit client behaves and wants to customize their own client.

Use is simple, first construct your audit server connection, we will use a domain socket here:

	c, err := conn.NewDomainSocket()
	if err != nil {
		// Do something
	}

Then construct your audit client:

	cli, err := audit.New(c)
	if err != nil {
		// Do something
	}

Then send audit records:

	if err := cli.Send(context.Background(), msgs.Msg{...}); err != nil {
		// Do something
	}

Finally, close the client when you are done:

	if err := cli.Close(context.Background()); err != nil {
		// Do something
	}

The client is asynchronous. This means that Send() will return immediately unless the queue is full or
your message doesn't validate. If the queue is full, Send() will return an error of type Error set to
ErrQueueFull.  You can detect this using IsQueueFull().

Send() returns one other type of error, which is if an unrecoverable error occurs. But that error will
not be for the message you are trying to send, but rather for the last message that failed.

You can check if it is an unrecoverable error by using IsUnrecoverable().
If it is unrecoverable, you should not use the client anymore. If it is unrecoverable, you can
recover the audit records that were not sent by using Recover(). You can then use those records
to send to another audit server or drop them. You can reset the client by using the Reset() method with
a new connection.

The audit client is designed to be used as a singleton. This means that you should only have one
at any time per application.

You can adjust the settings for the audit client by using the WithSettings() option.
*/
package base

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/microsoft/go-otel-audit/audit/conn"
	"github.com/microsoft/go-otel-audit/audit/internal/version"
	"github.com/microsoft/go-otel-audit/audit/msgs"
)

// ctxBack is used as a background context when we cannot use a context passed
// to a method (such as when sending a heartbeat). Putting it here avoids an
// allocation every time we need a background context in this manner.
var ctxBack = context.Background()

// DefaultQueueSize is the number of audit records that can be queued by default.
const DefaultQueueSize = 2048

var (
	// ErrValidation is an error that occurred during validation of an audit record.
	// This means the audit record is invalid and was not sent.
	ErrValidation = errors.New("validation error")
	// ErrConnection is an error that occurred while connecting to the audit server.
	// This means the audit client is in an unrecoverable state. Getting this error means that every
	// other call to the client will fail with the same message.
	ErrConnection = errors.New("connection error")
	// ErrQueueFull is an error that occurred because the queue is full. The message was not sent or
	// requeued.
	ErrQueueFull = errors.New("queue full")
	// ErrTimeout is an error that occurred because the send timed out.
	ErrTimeout = errors.New("timeout")
)

// IsUnrecoverable returns true if the error is unrecoverable.
// Unrecoverable errors mean that the audit client should not attempt to send any more audit
// records as the client is in an unrecoverable state (e.g. the connection is dead).
func IsUnrecoverable(err error) bool {
	if errors.Is(err, ErrConnection) {
		return true
	}
	return false
}

// Client represents a client connection to an audit server.
// Note that there should be only one client created per application. We do not use a singleton to prevent this,
// but rather we rely on the user to only create one client.
// Note: Context cancellation is only supported if explicitly noted in the method documentation.
type Client struct {
	// conn is the connection to the audit server.
	conn atomic.Pointer[conn.Audit]
	// settings are the settings for when the audit client sends audit records.
	settings Settings
	// sendCh is the channel for sending audit records to our async sender. It has a buffer
	// capacity of MaxQueueSize.
	sendCh   chan SendMsg
	stopSend chan chan struct{}
	// successSend tracks if we've successfully sent at least one message to the audit server.
	// This is so that we can start the heartbeat only once a successful message has been sent.
	successSend bool

	// done is closed when the internal sender goroutine is done.
	done chan struct{}

	// This next section represents our error handling. We use an atomic.Value to store the error
	// so that we can read it without a lock. We use a lock to write to the error. This is because
	// we want to ensure that the connection is only closed once. Reads of err are fast, setting an
	// error is slower, but only happens once.

	// err is the last error that occurred. This is used to determine if the client is in an
	// unrecoverable state.
	err atomic.Pointer[error]
	// setErrMu is used to to lock write changes to err. Reads can simply use atomic.Value.Load().
	setErrMu sync.Mutex

	closeOnceMu sync.Mutex
	// closeOnce is used to ensure that the client is only closed once.
	closeOnce sync.Once

	// log is the logger for the audit client. If not set, the default logger is used.
	log *slog.Logger

	// heartbeat is the message for the heartbeat.
	heartbeat msgs.Msg
	// heartbeatInterval is the interval for sending heartbeats. A value of
	// 0 means no heartbeats are sent.
	heartbeatInterval time.Duration
}

// Settings represents the settings for the audit client. These are used to configure how the
// client behaves. The zero value is valid and will use the default values. Negative values
// are invalid and will be set to the default value.
type Settings struct {
	// QueueSize is the maximum number of audit records that can be queued.
	// Defaults to MaxQueueSize. This queue is the queue not only for sending records
	// but Records that fail to send for any reason but validation and context timeouts
	// will be requeued here. When this is full, the record is dropped.
	QueueSize int
}

// defaults sets the default values for the settings. It returns a copy of settings to avoid an allocation.
func (s Settings) defaults() Settings {
	if s.QueueSize < 1 {
		s.QueueSize = DefaultQueueSize
	}
	return s
}

// Option represents an option for the audit client.
type Option func(c *Client) error

// WithSettings sets the settings for the audit client. If not set, the default values are used.
func WithSettings(s Settings) Option {
	return func(c *Client) error {
		c.settings = s.defaults()
		return nil
	}
}

// WithLogger sets the logger for the audit client. If not set, the default logger is used.
// This is for the few log messages that can occur, such as message drops. As a note, slog.Logger
// is not an interface. But loggers are created with a handler which tell the logger where to
// send the log messages. You can use that to convert a slog.Logger to a different logger if needed.
// Also, check with your logging package, as it may have a way to send slog.Logger messages to it.
func WithLogger(l *slog.Logger) Option {
	return func(c *Client) error {
		c.log = l
		return nil
	}
}

// New returns a new audit client.
func New(c conn.Audit, options ...Option) (client *Client, err error) {
	if c == nil {
		return nil, fmt.Errorf("cannot pass a conn.Audit that is nil: %w", ErrConnection)
	}

	cli := &Client{
		settings:          Settings{}.defaults(),
		log:               slog.Default(),
		heartbeatInterval: 30 * time.Minute,
		done:              make(chan struct{}),
	}
	cli.conn.Store(&c)

	defer func() {
		if err != nil {
			c.CloseSend(ctxBack)
		}
	}()

	for _, o := range options {
		if err := o(cli); err != nil {
			return nil, err
		}
	}

	kver, err := kernelVer()
	if err != nil {
		return nil, fmt.Errorf("could not determine kernel version on platform: %w", err)
	}

	cli.heartbeat = msgs.Msg{
		Type: msgs.Heartbeat,
		Heartbeat: msgs.HeartbeatMsg{
			AuditVersion: version.Semantic,
			OsVersion:    kver,
			Language:     runtime.Version(),
			Destination:  c.Type().String(),
		},
	}
	// This handles a special case where the version of the language is not goX.X.X, but instead
	// the git hash of the compiler. This happens on custom builds of the go compiler.
	if !strings.HasPrefix(cli.heartbeat.Heartbeat.Language, "go") {
		cli.heartbeat.Heartbeat.Language = "go" + cli.heartbeat.Heartbeat.Language
	}

	cli.sendCh = make(chan SendMsg, cli.settings.QueueSize)
	cli.stopSend = make(chan chan struct{}, 1)

	go cli.sender()

	return cli, nil
}

// Conn returns the current connection to the audit server. The one inside
// Client can be changed out from under you. This is provided for testing purposes.
// This is not supported for users.
func (c *Client) Conn() conn.Audit {
	ptr := c.conn.Load()
	if ptr == nil {
		return nil
	}
	return *ptr
}

// Logger returns the logger for the audit client.
func (c *Client) Logger() *slog.Logger {
	if c == nil || c.log == nil {
		return slog.Default()
	}
	return c.log
}

/*
Send sends an audit record to the audit server. Send is asynchronous and thread safe.

Send is designed around speed. It will return immediately if the queue is full or your message
doesn't validate. If the queue is full, Send() will return an error of type Error set to ErrQueueFull.
You can detect this using IsQueueFull(). It is up to the caller to handle this error and resend
the message.

If the queue is not full and the message validates, Send() will return nil. This means that the message was
queued to be sent to the audit server, not that the message was successfully sent. Send() is an asyncronous
method.

Any other errors are due to an unrecoverable error (such as a connection problem). If you get an error
from Send() that IsUnrecoverable() returns true to, you should not use the client anymore. The messages
that occur due to unrecoverable errors are put back into the  queue until that queue becomes full.
Whenever a ErrQueueFull message occurs, the message being sent is dropped and you must deal with the message
yourself.

Send does not honor Context cancellation.

Errors that are Unrecoverable or context cancelled will be sent to Recover(). It is up to the caller
to either handle these errors or ignore them. If you ignore them, you will lose audit records.
*/
func (c *Client) Send(ctx context.Context, msg msgs.Msg) error {
	if msg.Type == msgs.ATUnknown || msg.Type > msgs.ControlPlane {
		return fmt.Errorf("audit type (%v) is invalid: %w", msg.Type, ErrValidation)
	}

	if err := msg.Record.Validate(); err != nil {
		return fmt.Errorf("%w: %w", err, ErrValidation)
	}

	if msg.Record.Hook != nil {
		var err error
		msg.Record, err = msg.Record.Hook(msg.Record)
		if err != nil {
			return fmt.Errorf("%w: %w", err, ErrValidation)
		}
	}

	// We always want to send the message unless the queue is full or the context times out
	// before we get to send the message.
	select {
	case c.sendCh <- SendMsg{Ctx: ctx, Msg: msg}:
	default:
		return ErrQueueFull
	}

	// If we had any errors previously, we need to return them.
	if err := c.getErr(); err != nil {
		return err
	}
	return nil
}

// Reset resets the connection to the audit server.
// This can be used if the connection is in a bad state and needs to be reset.
// This will cause an existing connection to be closed and reset internal state.
// This is thread safe.
func (c *Client) Reset(ctx context.Context, newConn conn.Audit) error {
	if c == nil {
		return fmt.Errorf("cannot call Reset on a base.Client that is nil: %w", ErrConnection)
	}
	if newConn == nil {
		return fmt.Errorf("cannot pass a conn.Audit that is nil: %w", ErrConnection)
	}

	// If for some reason there is no error happening, we need to set one to prevents sends.

	c.setErr(fmt.Errorf("audit client Reset() called, resetting connection: %w", ErrConnection))

	// If the connection is live, we need to close it.
	c.close(ctx, false)
	c.conn.Store(&newConn)

	// Now let's bring everything back to life.
	c.done = make(chan struct{})

	c.closeOnceMu.Lock()
	c.closeOnce = sync.Once{}
	c.closeOnceMu.Unlock()

	c.setErr(nil)
	return nil
}

// Close closes the connection to the audit server.
func (c *Client) Close(ctx context.Context) error {
	c.wait()
	return c.close(ctx, true)
}

// wait waits for the client to finish sending all messages. This is determined by the queue
// being empty.
func (c *Client) wait() {
	doneWaiting := make(chan struct{})
	go func() {
		defer close(doneWaiting)
		for {
			if len(c.sendCh) == 0 {
				return
			}
			time.Sleep(100 * time.Millisecond)
		}
	}()
	for {
		select {
		case <-time.After(10 * time.Second):
			c.log.Info("waiting for audit log base client to send all messages after a Close()")
		case <-doneWaiting:
			return
		}
	}
}

// close handles closing the connection to the audit server. This is used by Close().
// It ensures that the connection is only closed once.
func (c *Client) close(ctx context.Context, stopSender bool) error {
	c.closeOnceMu.Lock()
	defer c.closeOnceMu.Unlock()

	c.closeOnce.Do(func() {
		if stopSender {
			// This might be new to some of you, but you can send a channel over a channel.
			// In this case, I create a channel and send it on the stopSend channel. When the
			// sender receives the channel, it will close the channel I sent, and then we know that
			// the sender is done.
			sig := make(chan struct{}, 1)
			c.stopSend <- sig
			<-sig
		}

		if err := c.getErr(); err != nil {
			ptr := c.conn.Load()
			if ptr != nil {
				(*ptr).CloseSend(ctx) // Ignore any error
			}
			return
		}

		ptr := c.conn.Load()
		if ptr != nil {
			if err := (*ptr).CloseSend(ctx); err != nil {
				c.setErr(fmt.Errorf("%w: %w", err, ErrConnection))
			}
		}
	})
	return c.getErr()
}

// sender is the async sender for the audit client.
func (c *Client) sender() {
	defer close(c.done)

	var ticker *time.Ticker
	defer func() {
		if ticker != nil {
			ticker.Stop()
		}
	}()

	var tickerCh <-chan time.Time

	for {
		if c.successSend && ticker == nil {
			c.write(ctxBack, c.heartbeat)
			ticker = time.NewTicker(c.heartbeatInterval)
			tickerCh = ticker.C
		}

		select {
		case sig := <-c.stopSend:
			// Let the other side know we are done.
			sig <- struct{}{}
			return
		case sm := <-c.sendCh:
			c.write(sm.Ctx, sm.Msg)
		case <-tickerCh:
			c.write(ctxBack, c.heartbeat)
		}
	}
}

func (c *Client) write(ctx context.Context, msg msgs.Msg) {
	ptr := c.conn.Load()
	if ptr == nil {
		return
	}

	if err := (*ptr).Write(ctx, msg); err != nil {
		// Requeue the message if we can.
		select {
		case c.sendCh <- SendMsg{Ctx: ctx, Msg: msg}:
		default:
			c.log.Error(fmt.Sprintf("audit message dropped due to queue being full: %v", err))
		}

		if err == context.Canceled {
			c.log.Error(fmt.Sprintf("audit message had context cancellation: %s", err))
			return
		}
		c.setErr(fmt.Errorf("%w: %w", err, ErrConnection))
		return
	}
	c.successSend = true
}

// setErr sets the error for the client if the error is not already set. If the error being set it nil,
// it will be set to nil.
func (c *Client) setErr(err error) {
	c.setErrMu.Lock()
	defer c.setErrMu.Unlock()

	if err == nil {
		c.err.Store(nil)
	}

	if c.err.Load() != nil {
		return
	}

	c.err.Store(&err)
}

// getErr gets the error for the client.
func (c *Client) getErr() error {
	if ptr := c.err.Load(); ptr != nil {
		return *ptr
	}
	return nil
}

// SendMsg holds the message to send and the context to use.
type SendMsg struct {
	// Ctx was the context sent with the message.
	Ctx context.Context
	// Msg is the message to send.
	Msg msgs.Msg
}
