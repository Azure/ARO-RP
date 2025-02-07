// Code generated by github.com/bennerv/go-cosmosdb, DO NOT EDIT.

package cosmosdb

import (
	"context"
	"fmt"
	"net/http"
	"sync"

	"github.com/ugorji/go/codec"

	pkg "github.com/Azure/ARO-RP/pkg/api"
)

type fakeGatewayDocumentTriggerHandler func(context.Context, *pkg.GatewayDocument) error
type fakeGatewayDocumentQueryHandler func(GatewayDocumentClient, *Query, *Options) GatewayDocumentRawIterator

var _ GatewayDocumentClient = &FakeGatewayDocumentClient{}

// NewFakeGatewayDocumentClient returns a FakeGatewayDocumentClient
func NewFakeGatewayDocumentClient(h *codec.JsonHandle) *FakeGatewayDocumentClient {
	return &FakeGatewayDocumentClient{
		jsonHandle:       h,
		gatewayDocuments: make(map[string]*pkg.GatewayDocument),
		triggerHandlers:  make(map[string]fakeGatewayDocumentTriggerHandler),
		queryHandlers:    make(map[string]fakeGatewayDocumentQueryHandler),
	}
}

// FakeGatewayDocumentClient is a FakeGatewayDocumentClient
type FakeGatewayDocumentClient struct {
	lock             sync.RWMutex
	jsonHandle       *codec.JsonHandle
	gatewayDocuments map[string]*pkg.GatewayDocument
	triggerHandlers  map[string]fakeGatewayDocumentTriggerHandler
	queryHandlers    map[string]fakeGatewayDocumentQueryHandler
	sorter           func([]*pkg.GatewayDocument)
	etag             int

	// returns true if documents conflict
	conflictChecker func(*pkg.GatewayDocument, *pkg.GatewayDocument) bool

	// err, if not nil, is an error to return when attempting to communicate
	// with this Client
	err error
}

// SetError sets or unsets an error that will be returned on any
// FakeGatewayDocumentClient method invocation
func (c *FakeGatewayDocumentClient) SetError(err error) {
	c.lock.Lock()
	defer c.lock.Unlock()

	c.err = err
}

// SetSorter sets or unsets a sorter function which will be used to sort values
// returned by List() for test stability
func (c *FakeGatewayDocumentClient) SetSorter(sorter func([]*pkg.GatewayDocument)) {
	c.lock.Lock()
	defer c.lock.Unlock()

	c.sorter = sorter
}

// SetConflictChecker sets or unsets a function which can be used to validate
// additional unique keys in a GatewayDocument
func (c *FakeGatewayDocumentClient) SetConflictChecker(conflictChecker func(*pkg.GatewayDocument, *pkg.GatewayDocument) bool) {
	c.lock.Lock()
	defer c.lock.Unlock()

	c.conflictChecker = conflictChecker
}

// SetTriggerHandler sets or unsets a trigger handler
func (c *FakeGatewayDocumentClient) SetTriggerHandler(triggerName string, trigger fakeGatewayDocumentTriggerHandler) {
	c.lock.Lock()
	defer c.lock.Unlock()

	c.triggerHandlers[triggerName] = trigger
}

// SetQueryHandler sets or unsets a query handler
func (c *FakeGatewayDocumentClient) SetQueryHandler(queryName string, query fakeGatewayDocumentQueryHandler) {
	c.lock.Lock()
	defer c.lock.Unlock()

	c.queryHandlers[queryName] = query
}

func (c *FakeGatewayDocumentClient) deepCopy(gatewayDocument *pkg.GatewayDocument) (*pkg.GatewayDocument, error) {
	var b []byte
	err := codec.NewEncoderBytes(&b, c.jsonHandle).Encode(gatewayDocument)
	if err != nil {
		return nil, err
	}

	gatewayDocument = nil
	err = codec.NewDecoderBytes(b, c.jsonHandle).Decode(&gatewayDocument)
	if err != nil {
		return nil, err
	}

	return gatewayDocument, nil
}

func (c *FakeGatewayDocumentClient) apply(ctx context.Context, partitionkey string, gatewayDocument *pkg.GatewayDocument, options *Options, isCreate bool) (*pkg.GatewayDocument, error) {
	c.lock.Lock()
	defer c.lock.Unlock()

	if c.err != nil {
		return nil, c.err
	}

	gatewayDocument, err := c.deepCopy(gatewayDocument) // copy now because pretriggers can mutate gatewayDocument
	if err != nil {
		return nil, err
	}

	if options != nil {
		err := c.processPreTriggers(ctx, gatewayDocument, options)
		if err != nil {
			return nil, err
		}
	}

	existingGatewayDocument, exists := c.gatewayDocuments[gatewayDocument.ID]
	if isCreate && exists {
		return nil, &Error{
			StatusCode: http.StatusConflict,
			Message:    "Entity with the specified id already exists in the system",
		}
	}
	if !isCreate {
		if !exists {
			return nil, &Error{StatusCode: http.StatusNotFound}
		}

		if gatewayDocument.ETag != existingGatewayDocument.ETag {
			return nil, &Error{StatusCode: http.StatusPreconditionFailed}
		}
	}

	if c.conflictChecker != nil {
		for _, gatewayDocumentToCheck := range c.gatewayDocuments {
			if c.conflictChecker(gatewayDocumentToCheck, gatewayDocument) {
				return nil, &Error{
					StatusCode: http.StatusConflict,
					Message:    "Entity with the specified id already exists in the system",
				}
			}
		}
	}

	gatewayDocument.ETag = fmt.Sprint(c.etag)
	c.etag++

	c.gatewayDocuments[gatewayDocument.ID] = gatewayDocument

	return c.deepCopy(gatewayDocument)
}

// Create creates a GatewayDocument in the database
func (c *FakeGatewayDocumentClient) Create(ctx context.Context, partitionkey string, gatewayDocument *pkg.GatewayDocument, options *Options) (*pkg.GatewayDocument, error) {
	return c.apply(ctx, partitionkey, gatewayDocument, options, true)
}

// Replace replaces a GatewayDocument in the database
func (c *FakeGatewayDocumentClient) Replace(ctx context.Context, partitionkey string, gatewayDocument *pkg.GatewayDocument, options *Options) (*pkg.GatewayDocument, error) {
	return c.apply(ctx, partitionkey, gatewayDocument, options, false)
}

// List returns a GatewayDocumentIterator to list all GatewayDocuments in the database
func (c *FakeGatewayDocumentClient) List(*Options) GatewayDocumentIterator {
	c.lock.RLock()
	defer c.lock.RUnlock()

	if c.err != nil {
		return NewFakeGatewayDocumentErroringRawIterator(c.err)
	}

	gatewayDocuments := make([]*pkg.GatewayDocument, 0, len(c.gatewayDocuments))
	for _, gatewayDocument := range c.gatewayDocuments {
		gatewayDocument, err := c.deepCopy(gatewayDocument)
		if err != nil {
			return NewFakeGatewayDocumentErroringRawIterator(err)
		}
		gatewayDocuments = append(gatewayDocuments, gatewayDocument)
	}

	if c.sorter != nil {
		c.sorter(gatewayDocuments)
	}

	return NewFakeGatewayDocumentIterator(gatewayDocuments, 0)
}

// ListAll lists all GatewayDocuments in the database
func (c *FakeGatewayDocumentClient) ListAll(ctx context.Context, options *Options) (*pkg.GatewayDocuments, error) {
	iter := c.List(options)
	return iter.Next(ctx, -1)
}

// Get gets a GatewayDocument from the database
func (c *FakeGatewayDocumentClient) Get(ctx context.Context, partitionkey string, id string, options *Options) (*pkg.GatewayDocument, error) {
	c.lock.RLock()
	defer c.lock.RUnlock()

	if c.err != nil {
		return nil, c.err
	}

	gatewayDocument, exists := c.gatewayDocuments[id]
	if !exists {
		return nil, &Error{StatusCode: http.StatusNotFound}
	}

	return c.deepCopy(gatewayDocument)
}

// Delete deletes a GatewayDocument from the database
func (c *FakeGatewayDocumentClient) Delete(ctx context.Context, partitionKey string, gatewayDocument *pkg.GatewayDocument, options *Options) error {
	c.lock.Lock()
	defer c.lock.Unlock()

	if c.err != nil {
		return c.err
	}

	_, exists := c.gatewayDocuments[gatewayDocument.ID]
	if !exists {
		return &Error{StatusCode: http.StatusNotFound}
	}

	delete(c.gatewayDocuments, gatewayDocument.ID)
	return nil
}

// ChangeFeed is unimplemented
func (c *FakeGatewayDocumentClient) ChangeFeed(*Options) GatewayDocumentIterator {
	c.lock.RLock()
	defer c.lock.RUnlock()

	if c.err != nil {
		return NewFakeGatewayDocumentErroringRawIterator(c.err)
	}

	return NewFakeGatewayDocumentErroringRawIterator(ErrNotImplemented)
}

func (c *FakeGatewayDocumentClient) processPreTriggers(ctx context.Context, gatewayDocument *pkg.GatewayDocument, options *Options) error {
	for _, triggerName := range options.PreTriggers {
		if triggerHandler := c.triggerHandlers[triggerName]; triggerHandler != nil {
			c.lock.Unlock()
			err := triggerHandler(ctx, gatewayDocument)
			c.lock.Lock()
			if err != nil {
				return err
			}
		} else {
			return ErrNotImplemented
		}
	}

	return nil
}

// Query calls a query handler to implement database querying
func (c *FakeGatewayDocumentClient) Query(name string, query *Query, options *Options) GatewayDocumentRawIterator {
	c.lock.RLock()
	defer c.lock.RUnlock()

	if c.err != nil {
		return NewFakeGatewayDocumentErroringRawIterator(c.err)
	}

	if queryHandler := c.queryHandlers[query.Query]; queryHandler != nil {
		c.lock.RUnlock()
		i := queryHandler(c, query, options)
		c.lock.RLock()
		return i
	}

	return NewFakeGatewayDocumentErroringRawIterator(ErrNotImplemented)
}

// QueryAll calls a query handler to implement database querying
func (c *FakeGatewayDocumentClient) QueryAll(ctx context.Context, partitionkey string, query *Query, options *Options) (*pkg.GatewayDocuments, error) {
	iter := c.Query("", query, options)
	return iter.Next(ctx, -1)
}

func NewFakeGatewayDocumentIterator(gatewayDocuments []*pkg.GatewayDocument, continuation int) GatewayDocumentRawIterator {
	return &fakeGatewayDocumentIterator{gatewayDocuments: gatewayDocuments, continuation: continuation}
}

type fakeGatewayDocumentIterator struct {
	gatewayDocuments []*pkg.GatewayDocument
	continuation     int
	done             bool
}

func (i *fakeGatewayDocumentIterator) NextRaw(ctx context.Context, maxItemCount int, out interface{}) error {
	return ErrNotImplemented
}

func (i *fakeGatewayDocumentIterator) Next(ctx context.Context, maxItemCount int) (*pkg.GatewayDocuments, error) {
	if i.done {
		return nil, nil
	}

	var gatewayDocuments []*pkg.GatewayDocument
	if maxItemCount == -1 {
		gatewayDocuments = i.gatewayDocuments[i.continuation:]
		i.continuation = len(i.gatewayDocuments)
		i.done = true
	} else {
		max := i.continuation + maxItemCount
		if max > len(i.gatewayDocuments) {
			max = len(i.gatewayDocuments)
		}
		gatewayDocuments = i.gatewayDocuments[i.continuation:max]
		i.continuation += max
		i.done = i.Continuation() == ""
	}

	return &pkg.GatewayDocuments{
		GatewayDocuments: gatewayDocuments,
		Count:            len(gatewayDocuments),
	}, nil
}

func (i *fakeGatewayDocumentIterator) Continuation() string {
	if i.continuation >= len(i.gatewayDocuments) {
		return ""
	}
	return fmt.Sprintf("%d", i.continuation)
}

// NewFakeGatewayDocumentErroringRawIterator returns a GatewayDocumentRawIterator which
// whose methods return the given error
func NewFakeGatewayDocumentErroringRawIterator(err error) GatewayDocumentRawIterator {
	return &fakeGatewayDocumentErroringRawIterator{err: err}
}

type fakeGatewayDocumentErroringRawIterator struct {
	err error
}

func (i *fakeGatewayDocumentErroringRawIterator) Next(ctx context.Context, maxItemCount int) (*pkg.GatewayDocuments, error) {
	return nil, i.err
}

func (i *fakeGatewayDocumentErroringRawIterator) NextRaw(context.Context, int, interface{}) error {
	return i.err
}

func (i *fakeGatewayDocumentErroringRawIterator) Continuation() string {
	return ""
}
