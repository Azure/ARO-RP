package cosmosdb

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"

	"github.com/ugorji/go/codec"

	pkg "github.com/jim-minter/go-cosmosdb/pkg/gencosmosdb/cosmosdb/dummy"
)

type FakeTemplateTrigger func(context.Context, *pkg.Template) error
type FakeTemplateQuery func(TemplateClient, *Query, *Options) TemplateRawIterator

var _ TemplateClient = &FakeTemplateClient{}

func NewFakeTemplateClient(h *codec.JsonHandle, uniqueKeys []string) *FakeTemplateClient {
	return &FakeTemplateClient{
		docs:       make(map[string][]byte),
		triggers:   make(map[string]FakeTemplateTrigger),
		queries:    make(map[string]FakeTemplateQuery),
		uniqueKeys: uniqueKeys,
		jsonHandle: h,
		lock:       &sync.RWMutex{},
	}
}

type FakeTemplateClient struct {
	docs       map[string][]byte
	jsonHandle *codec.JsonHandle
	lock       *sync.RWMutex
	triggers   map[string]FakeTemplateTrigger
	queries    map[string]FakeTemplateQuery
	uniqueKeys []string

	// unavailable, if not nil, is an error to throw when attempting to
	// communicate with this Client
	unavailable error
}

func decodeTemplate(s []byte, handle *codec.JsonHandle) (*pkg.Template, error) {
	res := &pkg.Template{}
	err := codec.NewDecoder(bytes.NewBuffer(s), handle).Decode(&res)
	return res, err
}

func decodeTemplateToMap(s []byte, handle *codec.JsonHandle) (map[interface{}]interface{}, error) {
	var res interface{}
	err := codec.NewDecoder(bytes.NewBuffer(s), handle).Decode(&res)
	if err != nil {
		return nil, err
	}
	ret, ok := res.(map[interface{}]interface{})
	if !ok {
		return nil, errors.New("Could not coerce")
	}
	return ret, err
}

func encodeTemplate(doc *pkg.Template, handle *codec.JsonHandle) (res []byte, err error) {
	buf := &bytes.Buffer{}
	err = codec.NewEncoder(buf, handle).Encode(doc)
	if err != nil {
		return
	}
	res = buf.Bytes()
	return
}

func (c *FakeTemplateClient) MakeUnavailable(err error) {
	c.unavailable = err
}

func (c *FakeTemplateClient) encodeAndCopy(doc *pkg.Template) (*pkg.Template, []byte, error) {
	encoded, err := encodeTemplate(doc, c.jsonHandle)
	if err != nil {
		return nil, nil, err
	}
	res, err := decodeTemplate(encoded, c.jsonHandle)
	if err != nil {
		return nil, nil, err
	}
	return res, encoded, err
}

func (c *FakeTemplateClient) Create(ctx context.Context, partitionkey string, doc *pkg.Template, options *Options) (*pkg.Template, error) {
	if c.unavailable != nil {
		return nil, c.unavailable
	}
	c.lock.Lock()
	defer c.lock.Unlock()

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
	docAsMap, err := decodeTemplateToMap(enc, c.jsonHandle)
	if err != nil {
		return nil, err
	}

	for _, ext := range c.docs {
		extDecoded, err := decodeTemplateToMap(ext, c.jsonHandle)
		if err != nil {
			return nil, err
		}

		for _, key := range c.uniqueKeys {
			var ourKeyStr string
			var theirKeyStr string
			ourKey, ourKeyOk := docAsMap[key]
			if ourKeyOk {
				ourKeyStr, ourKeyOk = ourKey.(string)
			}
			theirKey, theirKeyOk := extDecoded[key]
			if theirKeyOk {
				theirKeyStr, theirKeyOk = theirKey.(string)
			}
			if ourKeyOk && theirKeyOk && ourKeyStr != "" && ourKeyStr == theirKeyStr {
				return nil, &Error{
					StatusCode: http.StatusPreconditionFailed,
					Message:    "Entity with the specified id already exists in the system",
				}
			}
		}
	}

	c.docs[doc.ID] = enc
	return res, nil
}

func (c *FakeTemplateClient) List(*Options) TemplateIterator {
	if c.unavailable != nil {
		return NewFakeTemplateClientErroringRawIterator(c.unavailable)
	}
	c.lock.RLock()
	defer c.lock.RUnlock()

	docs := make([]*pkg.Template, 0, len(c.docs))
	for _, d := range c.docs {
		r, err := decodeTemplate(d, c.jsonHandle)
		if err != nil {
			// todo: ??? what do we do here
			fmt.Print(err)
		}
		docs = append(docs, r)
	}
	return NewFakeTemplateClientRawIterator(docs, 0)
}

func (c *FakeTemplateClient) ListAll(context.Context, *Options) (*pkg.Templates, error) {
	if c.unavailable != nil {
		return nil, c.unavailable
	}
	c.lock.RLock()
	defer c.lock.RUnlock()

	templates := &pkg.Templates{
		Count:     len(c.docs),
		Templates: make([]*pkg.Template, 0, len(c.docs)),
	}

	for _, d := range c.docs {
		dec, err := decodeTemplate(d, c.jsonHandle)
		if err != nil {
			return nil, err
		}
		templates.Templates = append(templates.Templates, dec)
	}
	return templates, nil
}

func (c *FakeTemplateClient) Get(ctx context.Context, partitionkey string, documentId string, options *Options) (*pkg.Template, error) {
	if c.unavailable != nil {
		return nil, c.unavailable
	}
	c.lock.RLock()
	defer c.lock.RUnlock()

	out, ext := c.docs[documentId]
	if !ext {
		return nil, &Error{StatusCode: http.StatusNotFound}
	}
	return decodeTemplate(out, c.jsonHandle)
}

func (c *FakeTemplateClient) Replace(ctx context.Context, partitionkey string, doc *pkg.Template, options *Options) (*pkg.Template, error) {
	if c.unavailable != nil {
		return nil, c.unavailable
	}
	c.lock.Lock()
	defer c.lock.Unlock()

	_, exists := c.docs[doc.ID]
	if !exists {
		return nil, &Error{StatusCode: http.StatusNotFound}
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
	c.docs[doc.ID] = enc
	return res, nil
}

func (c *FakeTemplateClient) Delete(ctx context.Context, partitionKey string, doc *pkg.Template, options *Options) error {
	if c.unavailable != nil {
		return c.unavailable
	}
	c.lock.Lock()
	defer c.lock.Unlock()

	_, ext := c.docs[doc.ID]
	if !ext {
		return &Error{StatusCode: http.StatusNotFound}
	}

	delete(c.docs, doc.ID)
	return nil
}

func (c *FakeTemplateClient) ChangeFeed(*Options) TemplateIterator {
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
	if c.unavailable != nil {
		return NewFakeTemplateClientErroringRawIterator(c.unavailable)
	}
	c.lock.RLock()
	defer c.lock.RUnlock()

	quer, ok := c.queries[query.Query]
	if ok {
		return quer(c, query, options)
	} else {
		return NewFakeTemplateClientErroringRawIterator(ErrNotImplemented)
	}
}

func (c *FakeTemplateClient) QueryAll(ctx context.Context, partitionkey string, query *Query, options *Options) (*pkg.Templates, error) {
	if c.unavailable != nil {
		return nil, c.unavailable
	}
	c.lock.RLock()
	defer c.lock.RUnlock()

	quer, ok := c.queries[query.Query]
	if ok {
		items := quer(c, query, options)
		res := &pkg.Templates{}
		err := items.NextRaw(ctx, -1, res)
		return res, err
	} else {
		return nil, ErrNotImplemented
	}
}

func (c *FakeTemplateClient) InjectTrigger(trigger string, impl FakeTemplateTrigger) {
	c.triggers[trigger] = impl
}

func (c *FakeTemplateClient) InjectQuery(query string, impl FakeTemplateQuery) {
	c.queries[query] = impl
}

// NewFakeTemplateClientRawIterator creates a RawIterator that will produce only
// Templates from Next() and NextRaw().
func NewFakeTemplateClientRawIterator(docs []*pkg.Template, continuation int) TemplateRawIterator {
	return &fakeTemplateClientRawIterator{docs: docs, continuation: continuation}
}

type fakeTemplateClientRawIterator struct {
	docs         []*pkg.Template
	continuation int
}

func (i *fakeTemplateClientRawIterator) Next(ctx context.Context, maxItemCount int) (*pkg.Templates, error) {
	out := &pkg.Templates{}
	err := i.NextRaw(ctx, maxItemCount, out)

	if out.Count == 0 {
		return nil, nil
	}
	return out, err
}

func (i *fakeTemplateClientRawIterator) NextRaw(ctx context.Context, maxItemCount int, out interface{}) error {
	if i.continuation >= len(i.docs) {
		return nil
	}

	var docs []*pkg.Template
	if maxItemCount == -1 {
		docs = i.docs[i.continuation:]
		i.continuation = len(i.docs)
	} else {
		max := i.continuation + maxItemCount
		if max > len(i.docs) {
			max = len(i.docs)
		}
		docs = i.docs[i.continuation:max]
		i.continuation += max
	}

	d := out.(*pkg.Templates)
	d.Templates = docs
	d.Count = len(d.Templates)
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
