/*
Package audit implements a smart client to a remote audit server. This handles any connection problems silently,
while providing ways to detect problems through whatever alerting method is desired. It also handles an exponential
backoff when trying to reconnect to the remote audit server as to prevent any overwhelming of the remote audit server.

This is the preferred way to connect to a remote audit server. There is a more low-level way to connect to a remote
audit server in the base package. This one allows for you to completely customize how you want to handle issues,
but it is more work to do so.

Example using the domainsocket package:

	// Create a function that will create a new connection to the remote audit server.
	// We use this function to create a new connection when the connection is broken.
	cc := func() (conn.Audit, error) {
		return conn.NewDomainSocket()
	}

	// Creates the smart client to the remote audit server.
	// You should only create one of these, preferrably in main().
	c, err := audit.New(cc)
	if err != nil {
		// Handle error.
	}
	defer c.Close(context.Background())

	// This is optional if you want to get notifications of logging problems.
	go func() {
		for notifyMsg := range c.Notify() {
			// Handle error notification.
			// You can log them or whatever you want to do.
		}
	}()

	// Send a message to the remote audit server.
	if err := c.Send(context.Background(), msgs.Msg{<add record information>}); err != nil {
		// Handle error.
		// Errors here will either be:
		// 1. base.ErrValidation , which means your message is invalid.
		// 2. base.ErrQueueFull, which means the queue is full and you are responsible for the message.
		// 3. A standard error, which means there is an error condition we haven't categorized yet.
		// If #3 happens, please file a bug report as we shouldn't send non-categorized errors to the user.
	}
*/
package audit

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"runtime"
	"sync/atomic"
	"testing"
	"time"

	"github.com/microsoft/go-otel-audit/audit/base"
	"github.com/microsoft/go-otel-audit/audit/conn"
	"github.com/microsoft/go-otel-audit/audit/msgs"

	"github.com/Azure/retry/exponential"
)

// NotifyError is an error that is sent to the Notify channel when the connection to the remote audit server is broken.
type NotifyError struct {
	// Time is the time the error occurred.
	Time time.Time
	// Err is the error that happened. This will be a an base.ErrConnection or base.ErrQueueFull.
	Err error
}

// Client is an audit server smart client. This handles any connection problems silently, while providing ways to
// detect problems through whatever alerting method is desired.
type Client struct {
	// client holds the current audit client.
	client *base.Client
	// replaceClient is used to ensure we only replace the client once. This is needed because we can have multiple
	// goroutines trying to replace the client at the same time.
	replacingClient atomic.Bool

	// create is the function that creates a new connection to the remote audit server. This is used when the
	// connection is broken.
	create CreateConn
	// backoff is the exponential backoff for making a new connection to the remote audit server.
	backoff *exponential.Backoff
	policy  exponential.Policy

	// notifier is the channel that will receive errors when the connection to the remote audit server is broken or
	// the queue is full. If this channel is full, the error will be dropped.
	notifier chan NotifyError
	// auditOptions are the options for the underlying base.Client.
	auditOptions []base.Option

	// replaceBackoffRunner is a function that replaces the audit client with a new one using an exponential backoff.
	// In production, this uses .replaceBackoff. In testing this can be replaced with a function that does nothing.
	replaceBackoffRunner func(ctx context.Context) error
	// sendRunner is a function that sends a message to the remote audit server.
	// In production, this uses .client.Send(). In testing this can be replaced with a function that does nothing.
	sendRunner func(ctx context.Context, msg msgs.Msg, options ...base.SendOption) error

	closed atomic.Bool
}

// CreateConn is a function that creates a connection to a remote audit server.
type CreateConn func() (conn.Audit, error)

// Option is an option for the smart client.
type Option func(*Client) error

// WithAuditOptions sets the audit options for the underlying audit client.
func WithAuditOptions(options ...base.Option) Option {
	return func(c *Client) error {
		c.auditOptions = options
		return nil
	}
}

// WithBackoffPolicy sets the exponential backoff policy for making a new connection to the remote audit server.
// By default this uses the default backoff settings from the exponential package.
func WithExponentialBackoff(policy exponential.Policy) Option {
	return func(c *Client) error {
		c.policy = policy
		return nil
	}
}

// New creates a new smart client to the remote audit server. You can get a CreateConn
// from conn.NewDomainSocket(), conn.NewTCP() or conn.NewNoop(). The last one is useful for testing
// when you don't want to send logs anywhere.
func New(create CreateConn, options ...Option) (*Client, error) {
	connect, err := create()
	if err != nil {
		return nil, err
	}

	if runtime.GOOS != "linux" && !testing.Testing() {
		if connect.Type() != conn.TypeNoOP {
			return nil, fmt.Errorf("audit: only linux clients can use Audit with a non-NoOp conn.Audit type")
		}
	}

	c := &Client{
		create:   create,
		notifier: make(chan NotifyError, 1),
	}
	c.replaceBackoffRunner = c.replaceBackoff
	c.replacingClient.Store(false)

	for _, o := range options {
		o(c)
	}

	if reflect.ValueOf(c.policy).IsZero() {
		var err error
		c.backoff, err = exponential.New()
		if err != nil {
			return nil, err
		}
	} else {
		c.backoff, err = exponential.New(exponential.WithPolicy(c.policy))
		if err != nil {
			return nil, err
		}
	}

	// Create the audit client.
	aClient, err := base.New(connect, c.auditOptions...)
	if err != nil {
		return nil, err
	}
	c.client = aClient
	c.sendRunner = c.client.Send

	c.closed.Store(false)
	return c, nil
}

// Notify returns a channel that will receive errors when the connection to the remote audit server is broken or
// the queue is full. If this channel is full, the error will be dropped.
func (c *Client) Notify() <-chan NotifyError {
	return c.notifier
}

// Send sends a message to the remote audit server. If the connection is broken, it will attempt to reconnect
// but you will not receive an error. The only errors that will be returned are due to the Record being
// invalid, trying to send a Msg with a type not DataPlane/ControlPlane, when we receive an
// uncategorized error (which always indicates a handling bug in the Client) or if Close() has been
// called. Context timeouts are not honored.
func (c *Client) Send(ctx context.Context, msg msgs.Msg, options ...base.SendOption) error {
	if c.closed.Load() {
		return fmt.Errorf("audit: client is closed")
	}

	if err := c.sendRunner(ctx, msg, options...); err != nil {
		if base.IsUnrecoverable(err) {
			c.replaceBackoffRunner(ctx) // Dropping error on purpose
			return nil
		}
		if errors.Is(err, base.ErrQueueFull) {
			c.sendNotification(err)
			return err
		}
		if errors.Is(err, base.ErrValidation) {
			return err
		}
		return fmt.Errorf("bug: error sending audit record that is uncategorized: %w", err)
	}
	return nil
}

// Close closes the connection to the remote audit server.
func (c *Client) Close(ctx context.Context) error {
	if c == nil {
		return nil
	}
	defer close(c.notifier)
	defer func() { c.closed.Store(true) }()
	if c.client == nil {
		return nil
	}
	return c.client.Close(ctx)
}

// replaceBackoff replaces the audit client with a new one using an exponential backoff.
func (c *Client) replaceBackoff(ctx context.Context) error {
	ctx = context.WithoutCancel(ctx)
	return c.backoff.Retry(
		ctx,
		func(ctx context.Context, r exponential.Record) error {
			c.client.Logger().Info(fmt.Sprintf("replacing audit client connection: Attempt %d, wait interval %v", r.Attempt, r.LastInterval))
			return c.replace(ctx)
		},
	)
}

// replace replaces the audit client with a new one.
func (c *Client) replace(ctx context.Context) error {
	swapped := c.replacingClient.CompareAndSwap(false, true)

	// This indicates that we are already replacing the client, so we don't need to do anything.
	if !swapped {
		return nil
	}

	// Indicate we are done changing out the client.
	defer c.replacingClient.Store(false)

	// Create a new connection.
	newConn, err := c.create()
	if err != nil {
		c.sendNotification(fmt.Errorf("error while trying to make a new connection: %w", err))
		return err
	}

	// Reset the client with the new connection.
	if err := c.client.Reset(ctx, newConn); err != nil {
		c.client.Logger().Error(fmt.Sprintf("error while trying to make a new client: %v", err))
		c.sendNotification(fmt.Errorf("error while trying to make a new client: %w", err))
		// This should nevever happen, because the only thing that causes an error is that create() is returning
		// nil. In that case, the error is permanent and we should cancel retries.
		return fmt.Errorf("%w: %w", err, exponential.ErrPermanent)
	}
	return nil
}

// sendNotification sends a notification to the notifier channel. If the channel is full, the notification is dropped.
func (c *Client) sendNotification(err error) {
	if err == nil {
		return
	}

	notice := NotifyError{
		Err:  err,
		Time: time.Now().UTC(),
	}

	select {
	case c.notifier <- notice:
	// Do nothing.
	default:
		// We don't care if the notifier is full.
	}
}
