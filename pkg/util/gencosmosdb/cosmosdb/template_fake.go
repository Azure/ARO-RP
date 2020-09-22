package cosmosdb

import (
	"context"
	"fmt"
	"net/http"
	"sync"

	"github.com/ugorji/go/codec"

	pkg "github.com/jim-minter/go-cosmosdb/pkg/gencosmosdb/cosmosdb/dummy"
)

type fakeTemplateTrigger func(context.Context, *pkg.Template) error
type fakeTemplateQuery func(TemplateClient, *Query, *Options) TemplateRawIterator

var _ TemplateClient = &FakeTemplateClient{}

func NewFakeTemplateClient(h *codec.JsonHandle) *FakeTemplateClient {
	return &FakeTemplateClient{
		docs:       make(map[string][]byte),
		triggers:   make(map[string]fakeTemplateTrigger),
		queries:    make(map[string]fakeTemplateQuery),
		jsonHandle: h,
		lock:       &sync.RWMutex{},
	}
}

type FakeTemplateClient struct {
	docs       map[string][]byte
	jsonHandle *codec.JsonHandle
	lock       *sync.RWMutex
	triggers   map[string]fakeTemplateTrigger
	queries    map[string]fakeTemplateQuery
	sorter     func([]*pkg.Template)

	// returns true if documents conflict
	checkDocsConflict func(*pkg.Template, *pkg.Template) bool

	// unavailable, if not nil, is an error to throw when attempting to
	// communicate with this Client
	unavailable error
}

func (c *FakeTemplateClient) decodeTemplate(s []byte) (res *pkg.Template, err error) {
	err = codec.NewDecoderBytes(s, c.jsonHandle).Decode(&res)
	return
}

func (c *FakeTemplateClient) encodeTemplate(doc *pkg.Template) (res []byte, err error) {
	err = codec.NewEncoderBytes(&res, c.jsonHandle).Encode(doc)
	return
}

func (c *FakeTemplateClient) MakeUnavailable(err error) {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.unavailable = err
}

func (c *FakeTemplateClient) UseSorter(sorter func([]*pkg.Template)) {
	c.lock.Lock()
	defer c.lock.Unlock()

	c.sorter = sorter
}

func (c *FakeTemplateClient) UseDocumentConflictChecker(checker func(*pkg.Template, *pkg.Template) bool) {
	c.lock.Lock()
	defer c.lock.Unlock()

	c.checkDocsConflict = checker
}

func (c *FakeTemplateClient) InjectTrigger(trigger string, impl fakeTemplateTrigger) {
	c.lock.Lock()
	defer c.lock.Unlock()

	c.triggers[trigger] = impl
}

func (c *FakeTemplateClient) InjectQuery(query string, impl fakeTemplateQuery) {
	c.lock.Lock()
	defer c.lock.Unlock()

	c.queries[query] = impl
}

func (c *FakeTemplateClient) encodeAndCopy(doc *pkg.Template) (*pkg.Template, []byte, error) {
	encoded, err := c.encodeTemplate(doc)
	if err != nil {
		return nil, nil, err
	}
	res, err := c.decodeTemplate(encoded)
	if err != nil {
		return nil, nil, err
	}
	return res, encoded, err
}

func (c *FakeTemplateClient) apply(ctx context.Context, partitionkey string, doc *pkg.Template, options *Options, isCreate bool) (*pkg.Template, error) {
	c.lock.Lock()
	defer c.lock.Unlock()

	if c.unavailable != nil {
		return nil, c.unavailable
	}

	if options != nil {
		err := c.processPreTriggers(ctx, doc, options)
		if err != nil {
			return nil, err
		}
	}

	res, enc, err := c.encodeAndCopy(doc)
	if err != nil {
		return nil, err
	}

	_, ext := c.docs[doc.ID]
	if isCreate && ext {
		return nil, &Error{
			StatusCode: http.StatusConflict,
			Message:    "Entity with the specified id already exists in the system",
		}
	}
	if !isCreate && !ext {
		return nil, &Error{StatusCode: http.StatusNotFound}
	}

	if c.checkDocsConflict != nil {
		for _, ext := range c.docs {
			dec, err := c.decodeTemplate(ext)
			if err != nil {
				return nil, err
			}

			if c.checkDocsConflict(dec, res) {
				return nil, &Error{
					StatusCode: http.StatusConflict,
					Message:    "Entity with the specified id already exists in the system",
				}
			}
		}
	}

	c.docs[doc.ID] = enc
	return res, nil
}

func (c *FakeTemplateClient) Create(ctx context.Context, partitionkey string, doc *pkg.Template, options *Options) (*pkg.Template, error) {
	return c.apply(ctx, partitionkey, doc, options, true)
}

func (c *FakeTemplateClient) Replace(ctx context.Context, partitionkey string, doc *pkg.Template, options *Options) (*pkg.Template, error) {
	return c.apply(ctx, partitionkey, doc, options, false)
}

func (c *FakeTemplateClient) List(*Options) TemplateIterator {
	c.lock.RLock()
	defer c.lock.RUnlock()

	if c.unavailable != nil {
		return NewFakeTemplateErroringRawIterator(c.unavailable)
	}

	docs := make([]*pkg.Template, 0, len(c.docs))
	for _, d := range c.docs {
		r, err := c.decodeTemplate(d)
		if err != nil {
			return NewFakeTemplateErroringRawIterator(err)
		}
		docs = append(docs, r)
	}

	if c.sorter != nil {
		c.sorter(docs)
	}

	return NewFakeTemplateIterator(docs, 0)
}

func (c *FakeTemplateClient) ListAll(ctx context.Context, opts *Options) (*pkg.Templates, error) {
	iter := c.List(opts)
	return iter.Next(ctx, -1)
}

func (c *FakeTemplateClient) Get(ctx context.Context, partitionkey string, documentId string, options *Options) (*pkg.Template, error) {
	c.lock.RLock()
	defer c.lock.RUnlock()

	if c.unavailable != nil {
		return nil, c.unavailable
	}

	out, ext := c.docs[documentId]
	if !ext {
		return nil, &Error{StatusCode: http.StatusNotFound}
	}
	return c.decodeTemplate(out)
}

func (c *FakeTemplateClient) Delete(ctx context.Context, partitionKey string, doc *pkg.Template, options *Options) error {
	c.lock.Lock()
	defer c.lock.Unlock()

	if c.unavailable != nil {
		return c.unavailable
	}

	_, ext := c.docs[doc.ID]
	if !ext {
		return &Error{StatusCode: http.StatusNotFound}
	}

	delete(c.docs, doc.ID)
	return nil
}

func (c *FakeTemplateClient) ChangeFeed(*Options) TemplateIterator {
	c.lock.RLock()
	defer c.lock.RUnlock()

	if c.unavailable != nil {
		return NewFakeTemplateErroringRawIterator(c.unavailable)
	}
	return NewFakeTemplateErroringRawIterator(ErrNotImplemented)
}

func (c *FakeTemplateClient) processPreTriggers(ctx context.Context, doc *pkg.Template, options *Options) error {
	for _, trigger := range options.PreTriggers {
		trig, ok := c.triggers[trigger]
		if ok {
			err := trig(ctx, doc)
			if err != nil {
				return err
			}
		} else {
			return ErrNotImplemented
		}
	}
	return nil
}

func (c *FakeTemplateClient) Query(name string, query *Query, options *Options) TemplateRawIterator {
	c.lock.RLock()
	defer c.lock.RUnlock()

	if c.unavailable != nil {
		return NewFakeTemplateErroringRawIterator(c.unavailable)
	}

	quer, ok := c.queries[query.Query]
	if ok {
		return quer(c, query, options)
	} else {
		return NewFakeTemplateErroringRawIterator(ErrNotImplemented)
	}
}

func (c *FakeTemplateClient) QueryAll(ctx context.Context, partitionkey string, query *Query, options *Options) (*pkg.Templates, error) {
	iter := c.Query("", query, options)
	return iter.Next(ctx, -1)
}

// NewFakeTemplateIterator creates a TemplateIterator that will produce
// only Templates from Next().
func NewFakeTemplateIterator(docs []*pkg.Template, continuation int) TemplateIterator {
	return &fakeTemplateIterator{docs: docs, continuation: continuation}
}

type fakeTemplateIterator struct {
	docs         []*pkg.Template
	continuation int
	done         bool
}

func (i *fakeTemplateIterator) Next(ctx context.Context, maxItemCount int) (*pkg.Templates, error) {
	if i.done {
		return nil, nil
	}

	var docs []*pkg.Template
	if maxItemCount == -1 {
		docs = i.docs[i.continuation:]
		i.continuation = len(i.docs)
		i.done = true
	} else {
		max := i.continuation + maxItemCount
		if max > len(i.docs) {
			max = len(i.docs)
		}
		docs = i.docs[i.continuation:max]
		i.continuation += max
		i.done = i.Continuation() == ""
	}

	return &pkg.Templates{
		Templates: docs,
		Count:     len(docs),
	}, nil
}

func (i *fakeTemplateIterator) Continuation() string {
	if i.continuation >= len(i.docs) {
		return ""
	}
	return fmt.Sprintf("%d", i.continuation)
}

func NewFakeTemplateErroringRawIterator(err error) *fakeTemplateErroringRawIterator {
	return &fakeTemplateErroringRawIterator{err: err}
}

// fakeTemplateErroringRawIterator is a RawIterator that will return an error on
// use.
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
