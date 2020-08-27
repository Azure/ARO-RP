package cosmosdb

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"

	pkg "github.com/jim-minter/go-cosmosdb/pkg/gencosmosdb/cosmosdb/dummy"
	"github.com/ugorji/go/codec"
)

type fakeTemplateClient struct {
	docs       map[string]*string
	jsonHandle *codec.JsonHandle
	lock       sync.Locker
}

func NewFakeTemplateClient(h *codec.JsonHandle) *fakeTemplateClient {
	return &fakeTemplateClient{
		docs:       make(map[string]*string),
		jsonHandle: h,
		lock:       &sync.Mutex{},
	}
}

func (c *fakeTemplateClient) fromString(s *string) (*pkg.Template, error) {
	res := &pkg.Template{}
	d := codec.NewDecoder(bytes.NewBufferString(*s), c.jsonHandle)
	err := d.Decode(&res)
	return res, err
}

func (c *fakeTemplateClient) processPreTrigger(ctx context.Context, doc *pkg.Template, trigger string) error {
	return ErrNotImplemented
}

func (c *fakeTemplateClient) Create(ctx context.Context, partitionkey string, doc *pkg.Template, options *Options) (*pkg.Template, error) {
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
	return c.fromString(&out)
}

func (c *fakeTemplateClient) List(*Options) TemplateRawIterator {
	c.lock.Lock()
	defer c.lock.Unlock()

	docs := make([]*string, 0, len(c.docs))
	for _, d := range c.docs {
		docs = append(docs, d)
	}

	return &fakeTemplateClientRawIterator{
		docs:       docs,
		jsonHandle: c.jsonHandle,
	}
}

func (c *fakeTemplateClient) ListAll(context.Context, *Options) (*pkg.Templates, error) {
	templates := &pkg.Templates{
		Count:     len(c.docs),
		Templates: make([]*pkg.Template, 0, len(c.docs)),
	}

	for _, d := range c.docs {
		dec, err := c.fromString(d)
		if err != nil {
			return nil, err
		}
		templates.Templates = append(templates.Templates, dec)
	}

	return templates, nil
}

func (c *fakeTemplateClient) Get(ctx context.Context, partitionkey string, documentId string, options *Options) (*pkg.Template, error) {
	c.lock.Lock()
	defer c.lock.Unlock()

	out, ext := c.docs[documentId]
	if !ext {
		return nil, &Error{StatusCode: http.StatusNotFound}
	}

	dec, err := c.fromString(out)
	if err != nil {
		return nil, err
	}

	return dec, err
}
func (c *fakeTemplateClient) Replace(ctx context.Context, partitionkey string, doc *pkg.Template, options *Options) (*pkg.Template, error) {
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
	return c.fromString(&out)
}

func (c *fakeTemplateClient) Delete(ctx context.Context, partitionKey string, doc *pkg.Template, options *Options) error {
	c.lock.Lock()
	defer c.lock.Unlock()

	_, ext := c.docs[doc.ID]
	if !ext {
		return &Error{StatusCode: http.StatusNotFound}
	}

	delete(c.docs, doc.ID)
	return nil
}

func (c *fakeTemplateClient) ChangeFeed(*Options) TemplateIterator {
	return &fakeTemplateNotImplementedIterator{}
}

type fakeTemplateClientRawIterator struct {
	docs         []*string
	jsonHandle   *codec.JsonHandle
	continuation int
}

func (i *fakeTemplateClientRawIterator) decode(inp []*string) ([]*pkg.Template, error) {
	done := make([]*pkg.Template, 0)

	for _, doc := range inp {
		res := &pkg.Template{}
		d := codec.NewDecoder(bytes.NewBufferString(*doc), i.jsonHandle)
		err := d.Decode(&res)
		if err != nil {
			return nil, err
		}
		done = append(done, res)
	}

	return done, nil
}

func (i *fakeTemplateClientRawIterator) Next(ctx context.Context, maxItemCount int) (*pkg.Templates, error) {
	docs := &pkg.Templates{}
	err := i.NextRaw(ctx, maxItemCount, docs)
	return docs, err
}

func (i *fakeTemplateClientRawIterator) NextRaw(ctx context.Context, maxItemCount int, raw interface{}) error {
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
		out, err := i.decode(docs)
		if err != nil {
			return err
		}
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
		err := d.Decode(&raw)

		if err != nil {
			return err
		}
	}
	return nil
}

func (i *fakeTemplateClientRawIterator) Continuation() string {
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
