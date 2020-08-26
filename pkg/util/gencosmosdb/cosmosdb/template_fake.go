package cosmosdb

import (
	"context"
	"net/http"

	pkg "github.com/jim-minter/go-cosmosdb/pkg/gencosmosdb/cosmosdb/dummy"
)

// TODO: we should be doing lots of deep copying in this file - perhaps m should
// be of type map[string][]byte and we should serialise?

type fakeTemplateClient struct {
	m map[string]*pkg.Template
}

type fakeTemplateListIterator struct {
	c       *fakeTemplateClient
	options *Options
}

type fakeTemplateNotImplementedIterator struct {
}

// NewFakeTemplateClient returns a new fake template client
func NewFakeTemplateClient() TemplateClient {
	return &fakeTemplateClient{
		m: map[string]*pkg.Template{},
	}
}

func (c *fakeTemplateClient) Create(ctx context.Context, partitionkey string, newtemplate *pkg.Template, options *Options) (*pkg.Template, error) {
	if options != nil {
		return nil, ErrNotImplemented
	}

	if _, ok := c.m[newtemplate.ID]; ok {
		return nil, &Error{StatusCode: http.StatusConflict} // TODO: check this
	}

	c.m[newtemplate.ID] = newtemplate

	return newtemplate, nil
}

func (c *fakeTemplateClient) List(options *Options) TemplateIterator {
	return &fakeTemplateListIterator{c: c, options: options}
}

func (c *fakeTemplateClient) ListAll(ctx context.Context, options *Options) (*pkg.Templates, error) {
	if options != nil {
		return nil, ErrNotImplemented
	}

	templates := &pkg.Templates{
		Count:     len(c.m),
		Templates: make([]*pkg.Template, 0, len(c.m)),
	}

	for _, template := range c.m {
		templates.Templates = append(templates.Templates, template)
	}

	return templates, nil
}

func (c *fakeTemplateClient) Get(ctx context.Context, partitionkey, templateid string, options *Options) (*pkg.Template, error) {
	if options != nil {
		return nil, ErrNotImplemented
	}

	if _, ok := c.m[templateid]; !ok {
		return nil, &Error{StatusCode: http.StatusNotFound}
	}

	return c.m[templateid], nil
}

func (c *fakeTemplateClient) Replace(ctx context.Context, partitionkey string, newtemplate *pkg.Template, options *Options) (*pkg.Template, error) {
	if options != nil {
		return nil, ErrNotImplemented
	}

	c.m[newtemplate.ID] = newtemplate

	return newtemplate, nil
}

func (c *fakeTemplateClient) Delete(ctx context.Context, partitionkey string, template *pkg.Template, options *Options) error {
	if options != nil {
		return ErrNotImplemented
	}

	if _, ok := c.m[template.ID]; !ok {
		return &Error{StatusCode: http.StatusNotFound}
	}

	delete(c.m, template.ID)

	return nil
}

func (c *fakeTemplateClient) Query(partitionkey string, query *Query, options *Options) TemplateRawIterator {
	return &fakeTemplateNotImplementedIterator{}
}

func (c *fakeTemplateClient) QueryAll(ctx context.Context, partitionkey string, query *Query, options *Options) (*pkg.Templates, error) {
	return nil, ErrNotImplemented
}

func (c *fakeTemplateClient) ChangeFeed(options *Options) TemplateIterator {
	return &fakeTemplateNotImplementedIterator{}
}

func (i *fakeTemplateListIterator) Next(ctx context.Context, maxItemCount int) (*pkg.Templates, error) {
	if i.options != nil {
		return nil, ErrNotImplemented
	}

	templates := &pkg.Templates{
		Count:     len(i.c.m),
		Templates: make([]*pkg.Template, 0, len(i.c.m)),
	}

	for _, template := range i.c.m {
		templates.Templates = append(templates.Templates, template)
	}

	return templates, nil
}

func (i *fakeTemplateListIterator) Continuation() string {
	return ""
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
