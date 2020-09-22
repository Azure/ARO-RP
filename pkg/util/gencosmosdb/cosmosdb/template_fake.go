package cosmosdb

import (
	"context"
	"fmt"
	"net/http"
	"reflect"
	"sync"

	"github.com/ugorji/go/codec"

	pkg "github.com/jim-minter/go-cosmosdb/pkg/gencosmosdb/cosmosdb/dummy"
)

type fakeTemplateTrigger func(context.Context, *pkg.Template) error
type fakeTemplateQuery func(TemplateClient, *Query, *Options) TemplateRawIterator

var _ TemplateClient = &FakeTemplateClient{}

func NewFakeTemplateClient(h *codec.JsonHandle) *FakeTemplateClient {
	return &FakeTemplateClient{
		docs:              make(map[string][]byte),
		triggers:          make(map[string]fakeTemplateTrigger),
		queries:           make(map[string]fakeTemplateQuery),
		jsonHandle:        h,
		lock:              &sync.RWMutex{},
		sorter:            func(in []*pkg.Template) {},
		checkDocsConflict: func(*pkg.Template, *pkg.Template) bool { return false },
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

func (c *FakeTemplateClient) decodeTemplate(s []byte) (*pkg.Template, error) {
	res := &pkg.Template{}
	err := codec.NewDecoderBytes(s, c.jsonHandle).Decode(&res)
	return res, err
}

func (c *FakeTemplateClient) encodeTemplate(doc *pkg.Template) ([]byte, error) {
	res := make([]byte, 0)
	err := codec.NewEncoderBytes(&res, c.jsonHandle).Encode(doc)
	if err != nil {
		return nil, err
	}
	return res, err
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

func (c *FakeTemplateClient) apply(ctx context.Context, partitionkey string, doc *pkg.Template, options *Options, isNew bool) (*pkg.Template, error) {
	c.lock.Lock()
	defer c.lock.Unlock()

	if c.unavailable != nil {
		return nil, c.unavailable
	}

	var docExists bool

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

	for _, ext := range c.docs {
		dec, err := c.decodeTemplate(ext)
		if err != nil {
			return nil, err
		}

		if dec.ID == res.ID {
			// If the document exists in the database, we want to error out in a
			// create but mark the document as extant so it can be replaced if
			// it is an update
			if isNew {
				return nil, &Error{
					StatusCode: http.StatusConflict,
					Message:    "Entity with the specified id already exists in the system",
				}
			} else {
				docExists = true
			}
		} else {
			if c.checkDocsConflict(dec, res) {
				return nil, &Error{
					StatusCode: http.StatusConflict,
					Message:    "Entity with the specified id already exists in the system",
				}
			}
		}
	}

	if !isNew && !docExists {
		return nil, &Error{StatusCode: http.StatusNotFound}
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
		return NewFakeTemplateClientErroringRawIterator(c.unavailable)
	}

	docs := make([]*pkg.Template, 0, len(c.docs))
	for _, d := range c.docs {
		r, err := c.decodeTemplate(d)
		if err != nil {
			return NewFakeTemplateClientErroringRawIterator(err)
		}
		docs = append(docs, r)
	}
	c.sorter(docs)
	return NewFakeTemplateClientRawIterator(docs, 0)
}

func (c *FakeTemplateClient) ListAll(ctx context.Context, opts *Options) (*pkg.Templates, error) {
	iter := c.List(opts)
	templates, err := iter.Next(ctx, -1)
	if err != nil {
		return nil, err
	}
	return templates, nil
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
		return NewFakeTemplateClientErroringRawIterator(c.unavailable)
	}
	return NewFakeTemplateClientErroringRawIterator(ErrNotImplemented)
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
		return NewFakeTemplateClientErroringRawIterator(c.unavailable)
	}

	quer, ok := c.queries[query.Query]
	if ok {
		return quer(c, query, options)
	} else {
		return NewFakeTemplateClientErroringRawIterator(ErrNotImplemented)
	}
}

func (c *FakeTemplateClient) QueryAll(ctx context.Context, partitionkey string, query *Query, options *Options) (*pkg.Templates, error) {
	iter := c.Query("", query, options)
	return iter.Next(ctx, -1)
}

// NewFakeTemplateClientRawIterator creates a RawIterator that will produce only
// Templates from Next() and NextRaw().
func NewFakeTemplateClientRawIterator(docs []*pkg.Template, continuation int) TemplateRawIterator {
	return &fakeTemplateClientRawIterator{docs: docs, continuation: continuation}
}

type fakeTemplateClientRawIterator struct {
	docs         []*pkg.Template
	continuation int
	done         bool
}

func (i *fakeTemplateClientRawIterator) Next(ctx context.Context, maxItemCount int) (out *pkg.Templates, err error) {
	err = i.NextRaw(ctx, maxItemCount, &out)
	return
}

func (i *fakeTemplateClientRawIterator) NextRaw(ctx context.Context, maxItemCount int, out interface{}) error {
	if i.done {
		return nil
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

	y := reflect.ValueOf(out)
	d := &pkg.Templates{}
	d.Templates = docs
	d.Count = len(d.Templates)
	y.Elem().Set(reflect.ValueOf(d))
	return nil
}

func (i *fakeTemplateClientRawIterator) Continuation() string {
	if i.continuation >= len(i.docs) {
		return ""
	}
	return fmt.Sprintf("%d", i.continuation)
}

// fakeTemplateErroringRawIterator is a RawIterator that will return an error on use.
func NewFakeTemplateClientErroringRawIterator(err error) *fakeTemplateErroringRawIterator {
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
