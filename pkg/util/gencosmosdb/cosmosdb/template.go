package cosmosdb

import (
	"context"
	"net/http"
	"strings"

	pkg "github.com/jim-minter/go-cosmosdb/pkg/gencosmosdb/cosmosdb/dummy"
)

type templateClient struct {
	*databaseClient
	path string
}

// TemplateClient is a template client
type TemplateClient interface {
	Create(context.Context, string, *pkg.Template, *Options) (*pkg.Template, error)
	List() TemplateIterator
	ListAll(context.Context) (*pkg.Templates, error)
	Get(context.Context, string, string) (*pkg.Template, error)
	Replace(context.Context, string, *pkg.Template, *Options) (*pkg.Template, error)
	Delete(context.Context, string, *pkg.Template, *Options) error
	Query(string, *Query) TemplateIterator
	QueryAll(context.Context, string, *Query) (*pkg.Templates, error)
}

type templateListIterator struct {
	*templateClient
	continuation string
	done         bool
}

type templateQueryIterator struct {
	*templateClient
	partitionkey string
	query        *Query
	continuation string
	done         bool
}

// TemplateIterator is a template iterator
type TemplateIterator interface {
	Next(context.Context) (*pkg.Templates, error)
}

// NewTemplateClient returns a new template client
func NewTemplateClient(collc CollectionClient, collid string) TemplateClient {
	return &templateClient{
		databaseClient: collc.(*collectionClient).databaseClient,
		path:           collc.(*collectionClient).path + "/colls/" + collid,
	}
}

func (c *templateClient) all(ctx context.Context, i TemplateIterator) (*pkg.Templates, error) {
	alltemplates := &pkg.Templates{}

	for {
		templates, err := i.Next(ctx)
		if err != nil {
			return nil, err
		}
		if templates == nil {
			break
		}

		alltemplates.Count += templates.Count
		alltemplates.ResourceID = templates.ResourceID
		alltemplates.Templates = append(alltemplates.Templates, templates.Templates...)
	}

	return alltemplates, nil
}

func (c *templateClient) Create(ctx context.Context, partitionkey string, newtemplate *pkg.Template, options *Options) (template *pkg.Template, err error) {
	headers := http.Header{}
	headers.Set("X-Ms-Documentdb-Partitionkey", `["`+partitionkey+`"]`)

	if options == nil {
		options = &Options{}
	}
	options.NoETag = true

	err = c.setOptions(options, newtemplate, headers)
	if err != nil {
		return
	}

	err = c.do(ctx, http.MethodPost, c.path+"/docs", "docs", c.path, http.StatusCreated, &newtemplate, &template, headers)
	return
}

func (c *templateClient) List() TemplateIterator {
	return &templateListIterator{templateClient: c}
}

func (c *templateClient) ListAll(ctx context.Context) (*pkg.Templates, error) {
	return c.all(ctx, c.List())
}

func (c *templateClient) Get(ctx context.Context, partitionkey, templateid string) (template *pkg.Template, err error) {
	headers := http.Header{}
	headers.Set("X-Ms-Documentdb-Partitionkey", `["`+partitionkey+`"]`)
	err = c.do(ctx, http.MethodGet, c.path+"/docs/"+templateid, "docs", c.path+"/docs/"+templateid, http.StatusOK, nil, &template, headers)
	return
}

func (c *templateClient) Replace(ctx context.Context, partitionkey string, newtemplate *pkg.Template, options *Options) (template *pkg.Template, err error) {
	headers := http.Header{}
	headers.Set("X-Ms-Documentdb-Partitionkey", `["`+partitionkey+`"]`)

	err = c.setOptions(options, newtemplate, headers)
	if err != nil {
		return
	}

	err = c.do(ctx, http.MethodPut, c.path+"/docs/"+newtemplate.ID, "docs", c.path+"/docs/"+newtemplate.ID, http.StatusOK, &newtemplate, &template, headers)
	return
}

func (c *templateClient) Delete(ctx context.Context, partitionkey string, template *pkg.Template, options *Options) (err error) {
	headers := http.Header{}
	headers.Set("X-Ms-Documentdb-Partitionkey", `["`+partitionkey+`"]`)

	err = c.setOptions(options, template, headers)
	if err != nil {
		return
	}

	err = c.do(ctx, http.MethodDelete, c.path+"/docs/"+template.ID, "docs", c.path+"/docs/"+template.ID, http.StatusNoContent, nil, nil, headers)
	return
}

func (c *templateClient) Query(partitionkey string, query *Query) TemplateIterator {
	return &templateQueryIterator{templateClient: c, partitionkey: partitionkey, query: query}
}

func (c *templateClient) QueryAll(ctx context.Context, partitionkey string, query *Query) (*pkg.Templates, error) {
	return c.all(ctx, c.Query(partitionkey, query))
}

func (c *templateClient) setOptions(options *Options, template *pkg.Template, headers http.Header) error {
	if options == nil {
		return nil
	}

	if !options.NoETag {
		if template.ETag == "" {
			return ErrETagRequired
		}
		headers.Set("If-Match", template.ETag)
	}
	if len(options.PreTriggers) > 0 {
		headers.Set("X-Ms-Documentdb-Pre-Trigger-Include", strings.Join(options.PreTriggers, ","))
	}
	if len(options.PostTriggers) > 0 {
		headers.Set("X-Ms-Documentdb-Post-Trigger-Include", strings.Join(options.PostTriggers, ","))
	}

	return nil
}

func (i *templateListIterator) Next(ctx context.Context) (templates *pkg.Templates, err error) {
	if i.done {
		return
	}

	headers := http.Header{}
	headers.Set("X-Ms-Max-Item-Count", "-1")
	if i.continuation != "" {
		headers.Set("X-Ms-Continuation", i.continuation)
	}

	err = i.do(ctx, http.MethodGet, i.path+"/docs", "docs", i.path, http.StatusOK, nil, &templates, headers)
	if err != nil {
		return
	}

	i.continuation = headers.Get("X-Ms-Continuation")
	i.done = i.continuation == ""

	return
}

func (i *templateQueryIterator) Next(ctx context.Context) (templates *pkg.Templates, err error) {
	if i.done {
		return
	}

	headers := http.Header{}
	headers.Set("X-Ms-Max-Item-Count", "-1")
	headers.Set("X-Ms-Documentdb-Isquery", "True")
	headers.Set("Content-Type", "application/query+json")
	if i.partitionkey != "" {
		headers.Set("X-Ms-Documentdb-Partitionkey", `["`+i.partitionkey+`"]`)
	} else {
		headers.Set("X-Ms-Documentdb-Query-Enablecrosspartition", "True")
	}
	if i.continuation != "" {
		headers.Set("X-Ms-Continuation", i.continuation)
	}

	err = i.do(ctx, http.MethodPost, i.path+"/docs", "docs", i.path, http.StatusOK, &i.query, &templates, headers)
	if err != nil {
		return
	}

	i.continuation = headers.Get("X-Ms-Continuation")
	i.done = i.continuation == ""

	return
}
