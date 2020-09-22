package cosmosdb

import (
	"context"
	"fmt"
	"net/http"
	"sync"

	"github.com/ugorji/go/codec"

	pkg "github.com/jim-minter/go-cosmosdb/pkg/gencosmosdb/cosmosdb/dummy"
)

type fakeTemplateTriggerHandler func(context.Context, *pkg.Template) error
type fakeTemplateQueryHandler func(TemplateClient, *Query, *Options) TemplateRawIterator

var _ TemplateClient = &FakeTemplateClient{}

// NewFakeTemplateClient returns a FakeTemplateClient
func NewFakeTemplateClient(h *codec.JsonHandle) *FakeTemplateClient {
	return &FakeTemplateClient{
		templates:       make(map[string][]byte),
		triggerHandlers: make(map[string]fakeTemplateTriggerHandler),
		queryHandlers:   make(map[string]fakeTemplateQueryHandler),
		jsonHandle:      h,
		lock:            &sync.RWMutex{},
	}
}

// FakeTemplateClient is a FakeTemplateClient
type FakeTemplateClient struct {
	templates       map[string][]byte
	jsonHandle      *codec.JsonHandle
	lock            *sync.RWMutex
	triggerHandlers map[string]fakeTemplateTriggerHandler
	queryHandlers   map[string]fakeTemplateQueryHandler
	sorter          func([]*pkg.Template)

	// returns true if documents conflict
	conflictChecker func(*pkg.Template, *pkg.Template) bool

	// err, if not nil, is an error to return when attempting to communicate
	// with this Client
	err error
}

func (c *FakeTemplateClient) decodeTemplate(s []byte) (template *pkg.Template, err error) {
	err = codec.NewDecoderBytes(s, c.jsonHandle).Decode(&template)
	return
}

func (c *FakeTemplateClient) encodeTemplate(template *pkg.Template) (b []byte, err error) {
	err = codec.NewEncoderBytes(&b, c.jsonHandle).Encode(template)
	return
}

// SetError sets or unsets an error that will be returned on any
// FakeTemplateClient method invocation
func (c *FakeTemplateClient) SetError(err error) {
	c.lock.Lock()
	defer c.lock.Unlock()

	c.err = err
}

// SetSorter sets or unsets a sorter function which will be used to sort values
// returned by List() for test stability
func (c *FakeTemplateClient) SetSorter(sorter func([]*pkg.Template)) {
	c.lock.Lock()
	defer c.lock.Unlock()

	c.sorter = sorter
}

// SetConflictChecker sets or unsets a function which can be used to validate
// additional unique keys in a Template
func (c *FakeTemplateClient) SetConflictChecker(conflictChecker func(*pkg.Template, *pkg.Template) bool) {
	c.lock.Lock()
	defer c.lock.Unlock()

	c.conflictChecker = conflictChecker
}

// SetTriggerHandler sets or unsets a trigger handler
func (c *FakeTemplateClient) SetTriggerHandler(triggerName string, trigger fakeTemplateTriggerHandler) {
	c.lock.Lock()
	defer c.lock.Unlock()

	c.triggerHandlers[triggerName] = trigger
}

// SetQueryHandler sets or unsets a query handler
func (c *FakeTemplateClient) SetQueryHandler(queryName string, query fakeTemplateQueryHandler) {
	c.lock.Lock()
	defer c.lock.Unlock()

	c.queryHandlers[queryName] = query
}

func (c *FakeTemplateClient) deepCopy(template *pkg.Template) (*pkg.Template, error) {
	b, err := c.encodeTemplate(template)
	if err != nil {
		return nil, err
	}

	return c.decodeTemplate(b)
}

func (c *FakeTemplateClient) apply(ctx context.Context, partitionkey string, template *pkg.Template, options *Options, isCreate bool) (*pkg.Template, error) {
	c.lock.Lock()
	defer c.lock.Unlock()

	if c.err != nil {
		return nil, c.err
	}

	template, err := c.deepCopy(template) // copy now because pretriggers can mutate template
	if err != nil {
		return nil, err
	}

	if options != nil {
		err := c.processPreTriggers(ctx, template, options)
		if err != nil {
			return nil, err
		}
	}

	_, exists := c.templates[template.ID]
	if isCreate && exists {
		return nil, &Error{
			StatusCode: http.StatusConflict,
			Message:    "Entity with the specified id already exists in the system",
		}
	}
	if !isCreate && !exists {
		return nil, &Error{StatusCode: http.StatusNotFound}
	}

	if c.conflictChecker != nil {
		for id := range c.templates {
			templateToCheck, err := c.decodeTemplate(c.templates[id])
			if err != nil {
				return nil, err
			}

			if c.conflictChecker(templateToCheck, template) {
				return nil, &Error{
					StatusCode: http.StatusConflict,
					Message:    "Entity with the specified id already exists in the system",
				}
			}
		}
	}

	b, err := c.encodeTemplate(template)
	if err != nil {
		return nil, err
	}

	c.templates[template.ID] = b

	return template, nil
}

// Create creates a Template in the database
func (c *FakeTemplateClient) Create(ctx context.Context, partitionkey string, template *pkg.Template, options *Options) (*pkg.Template, error) {
	return c.apply(ctx, partitionkey, template, options, true)
}

// Replace replaces a Template in the database
func (c *FakeTemplateClient) Replace(ctx context.Context, partitionkey string, template *pkg.Template, options *Options) (*pkg.Template, error) {
	return c.apply(ctx, partitionkey, template, options, false)
}

// List returns a TemplateIterator to list all Templates in the database
func (c *FakeTemplateClient) List(*Options) TemplateIterator {
	c.lock.RLock()
	defer c.lock.RUnlock()

	if c.err != nil {
		return NewFakeTemplateErroringRawIterator(c.err)
	}

	templates := make([]*pkg.Template, 0, len(c.templates))
	for _, d := range c.templates {
		r, err := c.decodeTemplate(d)
		if err != nil {
			return NewFakeTemplateErroringRawIterator(err)
		}
		templates = append(templates, r)
	}

	if c.sorter != nil {
		c.sorter(templates)
	}

	return newFakeTemplateIterator(templates, 0)
}

// ListAll lists all Templates in the database
func (c *FakeTemplateClient) ListAll(ctx context.Context, options *Options) (*pkg.Templates, error) {
	iter := c.List(options)
	return iter.Next(ctx, -1)
}

// Get gets a Template from the database
func (c *FakeTemplateClient) Get(ctx context.Context, partitionkey string, id string, options *Options) (*pkg.Template, error) {
	c.lock.RLock()
	defer c.lock.RUnlock()

	if c.err != nil {
		return nil, c.err
	}

	template, exists := c.templates[id]
	if !exists {
		return nil, &Error{StatusCode: http.StatusNotFound}
	}

	return c.decodeTemplate(template)
}

// Delete deletes a Template from the database
func (c *FakeTemplateClient) Delete(ctx context.Context, partitionKey string, template *pkg.Template, options *Options) error {
	c.lock.Lock()
	defer c.lock.Unlock()

	if c.err != nil {
		return c.err
	}

	_, exists := c.templates[template.ID]
	if !exists {
		return &Error{StatusCode: http.StatusNotFound}
	}

	delete(c.templates, template.ID)
	return nil
}

// ChangeFeed is unimplemented
func (c *FakeTemplateClient) ChangeFeed(*Options) TemplateIterator {
	c.lock.RLock()
	defer c.lock.RUnlock()

	if c.err != nil {
		return NewFakeTemplateErroringRawIterator(c.err)
	}

	return NewFakeTemplateErroringRawIterator(ErrNotImplemented)
}

func (c *FakeTemplateClient) processPreTriggers(ctx context.Context, template *pkg.Template, options *Options) error {
	for _, triggerName := range options.PreTriggers {
		if triggerHandler := c.triggerHandlers[triggerName]; triggerHandler != nil {
			err := triggerHandler(ctx, template)
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
func (c *FakeTemplateClient) Query(name string, query *Query, options *Options) TemplateRawIterator {
	c.lock.RLock()
	defer c.lock.RUnlock()

	if c.err != nil {
		return NewFakeTemplateErroringRawIterator(c.err)
	}

	if queryHandler := c.queryHandlers[query.Query]; queryHandler != nil {
		return queryHandler(c, query, options)
	}

	return NewFakeTemplateErroringRawIterator(ErrNotImplemented)
}

// QueryAll calls a query handler to implement database querying
func (c *FakeTemplateClient) QueryAll(ctx context.Context, partitionkey string, query *Query, options *Options) (*pkg.Templates, error) {
	iter := c.Query("", query, options)
	return iter.Next(ctx, -1)
}

func newFakeTemplateIterator(templates []*pkg.Template, continuation int) TemplateIterator {
	return &fakeTemplateIterator{templates: templates, continuation: continuation}
}

type fakeTemplateIterator struct {
	templates    []*pkg.Template
	continuation int
	done         bool
}

func (i *fakeTemplateIterator) Next(ctx context.Context, maxItemCount int) (*pkg.Templates, error) {
	if i.done {
		return nil, nil
	}

	var templates []*pkg.Template
	if maxItemCount == -1 {
		templates = i.templates[i.continuation:]
		i.continuation = len(i.templates)
		i.done = true
	} else {
		max := i.continuation + maxItemCount
		if max > len(i.templates) {
			max = len(i.templates)
		}
		templates = i.templates[i.continuation:max]
		i.continuation += max
		i.done = i.Continuation() == ""
	}

	return &pkg.Templates{
		Templates: templates,
		Count:     len(templates),
	}, nil
}

func (i *fakeTemplateIterator) Continuation() string {
	if i.continuation >= len(i.templates) {
		return ""
	}
	return fmt.Sprintf("%d", i.continuation)
}

// NewFakeTemplateErroringRawIterator returns a TemplateRawIterator which
// whose methods return the given error
func NewFakeTemplateErroringRawIterator(err error) TemplateRawIterator {
	return &fakeTemplateErroringRawIterator{err: err}
}

type fakeTemplateErroringRawIterator struct {
	err error
}

func (i *fakeTemplateErroringRawIterator) Next(ctx context.Context, maxItemCount int) (*pkg.Templates, error) {
	return nil, i.err
}

func (i *fakeTemplateErroringRawIterator) NextRaw(context.Context, int, interface{}) error {
	return i.err
}

func (i *fakeTemplateErroringRawIterator) Continuation() string {
	return ""
}
