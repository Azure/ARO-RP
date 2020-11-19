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
		jsonHandle:      h,
		templates:       make(map[string]*pkg.Template),
		triggerHandlers: make(map[string]fakeTemplateTriggerHandler),
		queryHandlers:   make(map[string]fakeTemplateQueryHandler),
	}
}

// FakeTemplateClient is a FakeTemplateClient
type FakeTemplateClient struct {
	lock            sync.RWMutex
	jsonHandle      *codec.JsonHandle
	templates       map[string]*pkg.Template
	triggerHandlers map[string]fakeTemplateTriggerHandler
	queryHandlers   map[string]fakeTemplateQueryHandler
	sorter          func([]*pkg.Template)
	etag            int

	// returns true if documents conflict
	conflictChecker func(*pkg.Template, *pkg.Template) bool

	// err, if not nil, is an error to return when attempting to communicate
	// with this Client
	err error
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
	var b []byte
	err := codec.NewEncoderBytes(&b, c.jsonHandle).Encode(template)
	if err != nil {
		return nil, err
	}

	template = nil
	err = codec.NewDecoderBytes(b, c.jsonHandle).Decode(&template)
	if err != nil {
		return nil, err
	}

	return template, nil
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

	existingTemplate, exists := c.templates[template.ID]
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

		if template.ETag != existingTemplate.ETag {
			return nil, &Error{StatusCode: http.StatusPreconditionFailed}
		}
	}

	if c.conflictChecker != nil {
		for _, templateToCheck := range c.templates {
			if c.conflictChecker(templateToCheck, template) {
				return nil, &Error{
					StatusCode: http.StatusConflict,
					Message:    "Entity with the specified id already exists in the system",
				}
			}
		}
	}

	template.ETag = fmt.Sprint(c.etag)
	c.etag++

	c.templates[template.ID] = template

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
	for _, template := range c.templates {
		template, err := c.deepCopy(template)
		if err != nil {
			return NewFakeTemplateErroringRawIterator(err)
		}
		templates = append(templates, template)
	}

	if c.sorter != nil {
		c.sorter(templates)
	}

	return NewFakeTemplateIterator(templates, 0)
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

	return c.deepCopy(template)
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
			c.lock.Unlock()
			err := triggerHandler(ctx, template)
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
func (c *FakeTemplateClient) Query(name string, query *Query, options *Options) TemplateRawIterator {
	c.lock.RLock()
	defer c.lock.RUnlock()

	if c.err != nil {
		return NewFakeTemplateErroringRawIterator(c.err)
	}

	if queryHandler := c.queryHandlers[query.Query]; queryHandler != nil {
		c.lock.RUnlock()
		i := queryHandler(c, query, options)
		c.lock.RLock()
		return i
	}

	return NewFakeTemplateErroringRawIterator(ErrNotImplemented)
}

// QueryAll calls a query handler to implement database querying
func (c *FakeTemplateClient) QueryAll(ctx context.Context, partitionkey string, query *Query, options *Options) (*pkg.Templates, error) {
	iter := c.Query("", query, options)
	return iter.Next(ctx, -1)
}

func NewFakeTemplateIterator(templates []*pkg.Template, continuation int) TemplateRawIterator {
	return &fakeTemplateIterator{templates: templates, continuation: continuation}
}

type fakeTemplateIterator struct {
	templates    []*pkg.Template
	continuation int
	done         bool
}

func (i *fakeTemplateIterator) NextRaw(ctx context.Context, maxItemCount int, out interface{}) error {
	return ErrNotImplemented
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
