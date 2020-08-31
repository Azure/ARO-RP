package cosmosdb

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"

	"github.com/ugorji/go/codec"

	pkg "github.com/jim-minter/go-cosmosdb/pkg/gencosmosdb/cosmosdb/dummy"
)

type FakeTemplateDecoder func(*string) *pkg.Template
type FakeTemplateTrigger func(context.Context, *pkg.Template) error
type FakeTemplateQuery func(*Query, map[string]*string, FakeTemplateDecoder) []*string

var _ TemplateClient = &FakeTemplateClient{}

type FakeTemplateClient struct {
	docs       map[string]*string
	jsonHandle *codec.JsonHandle
	lock       sync.Locker
	triggers   map[string]FakeTemplateTrigger
	queries    map[string]FakeTemplateQuery
}

func NewFakeTemplateClient(h *codec.JsonHandle) *FakeTemplateClient {
	return &FakeTemplateClient{
		docs:       make(map[string]*string),
		triggers:   make(map[string]FakeTemplateTrigger),
		queries:    make(map[string]FakeTemplateQuery),
		jsonHandle: h,
		lock:       &sync.Mutex{},
	}
}

func (c *FakeTemplateClient) fromString(s *string) *pkg.Template {
	res := &pkg.Template{}
	d := codec.NewDecoder(bytes.NewBufferString(*s), c.jsonHandle)
	err := d.Decode(&res)
	if err != nil {
		panic(err)
	}
	return res
}

func (c *FakeTemplateClient) Create(ctx context.Context, partitionkey string, doc *pkg.Template, options *Options) (*pkg.Template, error) {
	c.lock.Lock()
	defer c.lock.Unlock()

	_, ext := c.docs[doc.ID]
	if ext {
		return nil, &Error{
			StatusCode: http.StatusPreconditionFailed,
			Message:    "Entity with the specified id already exists in the system",
		}
	}

	if options != nil {
		for _, o := range options.PreTriggers {
			err := c.processPreTrigger(ctx, doc, o)
			if err != nil {
				return nil, err
			}
		}
	}

	buf := &bytes.Buffer{}
	err := codec.NewEncoder(buf, c.jsonHandle).Encode(doc)
	if err != nil {
		return nil, err
	}

	out := buf.String()
	c.docs[doc.ID] = &out
	return c.fromString(&out), nil
}

func (c *FakeTemplateClient) List(*Options) TemplateIterator {
	c.lock.Lock()
	defer c.lock.Unlock()

	docs := make([]*string, 0, len(c.docs))
	for _, d := range c.docs {
		docs = append(docs, d)
	}

	return &FakeTemplateClientRawIterator{
		docs:       docs,
		jsonHandle: c.jsonHandle,
	}
}

func (c *FakeTemplateClient) ListAll(context.Context, *Options) (*pkg.Templates, error) {
	templates := &pkg.Templates{
		Count:     len(c.docs),
		Templates: make([]*pkg.Template, 0, len(c.docs)),
	}

	for _, d := range c.docs {
		dec := c.fromString(d)
		templates.Templates = append(templates.Templates, dec)
	}

	return templates, nil
}

func (c *FakeTemplateClient) Get(ctx context.Context, partitionkey string, documentId string, options *Options) (*pkg.Template, error) {
	c.lock.Lock()
	defer c.lock.Unlock()

	out, ext := c.docs[documentId]
	if !ext {
		return nil, &Error{StatusCode: http.StatusNotFound}
	}

	dec := c.fromString(out)
	return dec, nil
}
func (c *FakeTemplateClient) Replace(ctx context.Context, partitionkey string, doc *pkg.Template, options *Options) (*pkg.Template, error) {
	c.lock.Lock()
	defer c.lock.Unlock()

	_, exists := c.docs[doc.ID]
	if !exists {
		return nil, &Error{StatusCode: http.StatusNotFound}
	}

	if options != nil {
		for _, o := range options.PreTriggers {
			err := c.processPreTrigger(ctx, doc, o)
			if err != nil {
				return nil, err
			}
		}
	}

	buf := &bytes.Buffer{}
	err := codec.NewEncoder(buf, c.jsonHandle).Encode(doc)
	if err != nil {
		return nil, err
	}

	out := buf.String()
	c.docs[doc.ID] = &out
	return c.fromString(&out), nil
}

func (c *FakeTemplateClient) Delete(ctx context.Context, partitionKey string, doc *pkg.Template, options *Options) error {
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
	return &fakeTemplateNotImplementedIterator{}
}

func (c *FakeTemplateClient) processPreTrigger(ctx context.Context, doc *pkg.Template, trigger string) (err error) {
	trig, ok := c.triggers[trigger]
	if ok {
		return trig(ctx, doc)
	} else {
		panic(ErrNotImplemented)
	}
}

func (c *FakeTemplateClient) Query(name string, query *Query, options *Options) TemplateRawIterator {
	c.lock.Lock()
	defer c.lock.Unlock()

	quer, ok := c.queries[query.Query]
	if ok {
		docs := quer(query, c.docs, c.fromString)
		return &FakeTemplateClientRawIterator{
			docs:       docs,
			jsonHandle: c.jsonHandle,
		}
	} else {
		panic(ErrNotImplemented)
	}
}

func (c *FakeTemplateClient) QueryAll(ctx context.Context, partitionkey string, query *Query, options *Options) (*pkg.Templates, error) {
	c.lock.Lock()
	defer c.lock.Unlock()

	var results []*pkg.Template

	quer, ok := c.queries[query.Query]
	if ok {
		for _, doc := range quer(query, c.docs, c.fromString) {
			results = append(results, c.fromString(doc))
		}
	} else {
		panic(ErrNotImplemented)
	}

	return &pkg.Templates{
		Count:     len(results),
		Templates: results,
	}, nil
}

func (c *FakeTemplateClient) InjectTrigger(trigger string, impl FakeTemplateTrigger) {
	c.triggers[trigger] = impl
}

func (c *FakeTemplateClient) InjectQuery(query string, impl FakeTemplateQuery) {
	c.queries[query] = impl
}

type FakeTemplateClientRawIterator struct {
	docs         []*string
	jsonHandle   *codec.JsonHandle
	continuation int
}

func (i *FakeTemplateClientRawIterator) decode(inp []*string) (done []*pkg.Template, err error) {
	for _, doc := range inp {
		res := &pkg.Template{}
		d := codec.NewDecoder(bytes.NewBufferString(*doc), i.jsonHandle)
		err := d.Decode(&res)
		if err != nil {
			return nil, err
		}
		done = append(done, res)
	}
	return
}

func (i *FakeTemplateClientRawIterator) Next(ctx context.Context, maxItemCount int) (*pkg.Templates, error) {
	docs := &pkg.Templates{}
	err := i.NextRaw(ctx, maxItemCount, docs)
	return docs, err
}

func (i *FakeTemplateClientRawIterator) NextRaw(ctx context.Context, maxItemCount int, raw interface{}) (err error) {
	if i.continuation >= len(i.docs) {
		return nil
	}

	var docs []*string
	if maxItemCount == -1 {
		docs = i.docs[i.continuation:]
		i.continuation = len(i.docs)
	} else {
		docs = i.docs[i.continuation : i.continuation+maxItemCount]
		i.continuation += maxItemCount
	}

	d, ok := raw.(*pkg.Templates)
	if ok {
		var out []*pkg.Template
		out, err = i.decode(docs)
		d.Templates = out
		d.Count = len(d.Templates)
	} else {
		var f strings.Builder
		fmt.Fprintf(&f, `{"Count": %d, "Documents": [`, len(docs))
		for n, doc := range docs {
			fmt.Fprint(&f, *doc)
			if n != len(docs) {
				fmt.Fprint(&f, ",")
			}
		}
		fmt.Fprint(&f, "]}")
		d := codec.NewDecoder(bytes.NewBufferString(f.String()), i.jsonHandle)
		err = d.Decode(&raw)
	}
	return err
}

func (i *FakeTemplateClientRawIterator) Continuation() string {
	return ""
}

type fakeTemplateNotImplementedIterator struct {
}

func (i *fakeTemplateNotImplementedIterator) Next(ctx context.Context, maxItemCount int) (*pkg.Templates, error) {
	return nil, ErrNotImplemented
}

func (i *fakeTemplateNotImplementedIterator) NextRaw(context.Context, int, interface{}) error {
	return ErrNotImplemented
}

func (i *fakeTemplateNotImplementedIterator) Continuation() string {
	return ""
}
