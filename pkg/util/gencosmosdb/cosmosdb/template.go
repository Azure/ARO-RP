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
	List(*Options) TemplateRawIterator
	ListAll(context.Context, *Options) (*pkg.Templates, error)
	Get(context.Context, string, string, *Options) (*pkg.Template, error)
	Replace(context.Context, string, *pkg.Template, *Options) (*pkg.Template, error)
	Delete(context.Context, string, *pkg.Template, *Options) error
	Query(string, *Query, *Options) TemplateRawIterator
	QueryAll(context.Context, string, *Query, *Options) (*pkg.Templates, error)
	ChangeFeed(*Options) TemplateIterator
}

type templateChangeFeedIterator struct {
	*templateClient
	continuation string
	options      *Options
}

type templateListIterator struct {
	*templateClient
	continuation string
	done         bool
	options      *Options
}

type templateQueryIterator struct {
	*templateClient
	partitionkey string
	query        *Query
	continuation string
	done         bool
	options      *Options
}

// TemplateIterator is a template iterator
type TemplateIterator interface {
	Next(context.Context) (*pkg.Templates, error)
	Continuation() string
}

// TemplateRawIterator is a template raw iterator
type TemplateRawIterator interface {
	TemplateIterator
	NextRaw(context.Context, interface{}) error
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

func (c *templateClient) List(options *Options) TemplateRawIterator {
	continuation := ""
	if options != nil {
		continuation = options.Continuation
	}

	return &templateListIterator{templateClient: c, options: options, continuation: continuation}
}

func (c *templateClient) ListAll(ctx context.Context, options *Options) (*pkg.Templates, error) {
	return c.all(ctx, c.List(options))
}

func (c *templateClient) Get(ctx context.Context, partitionkey, templateid string, options *Options) (template *pkg.Template, err error) {
	headers := http.Header{}
	headers.Set("X-Ms-Documentdb-Partitionkey", `["`+partitionkey+`"]`)

	err = c.setOptions(options, nil, headers)
	if err != nil {
		return
	}

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

func (c *templateClient) Query(partitionkey string, query *Query, options *Options) TemplateRawIterator {
	continuation := ""
	if options != nil {
		continuation = options.Continuation
	}

	return &templateQueryIterator{templateClient: c, partitionkey: partitionkey, query: query, options: options, continuation: continuation}
}

func (c *templateClient) QueryAll(ctx context.Context, partitionkey string, query *Query, options *Options) (*pkg.Templates, error) {
	return c.all(ctx, c.Query(partitionkey, query, options))
}

func (c *templateClient) ChangeFeed(options *Options) TemplateIterator {
	continuation := ""
	if options != nil {
		continuation = options.Continuation
	}

	return &templateChangeFeedIterator{templateClient: c, options: options, continuation: continuation}
}

func (c *templateClient) setOptions(options *Options, template *pkg.Template, headers http.Header) error {
	if options == nil {
		return nil
	}

	if template != nil && !options.NoETag {
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
	if len(options.PartitionKeyRangeID) > 0 {
		headers.Set("X-Ms-Documentdb-PartitionKeyRangeID", options.PartitionKeyRangeID)
	}

	return nil
}

func (i *templateChangeFeedIterator) Next(ctx context.Context) (templates *pkg.Templates, err error) {
	headers := http.Header{}
	headers.Set("A-IM", "Incremental feed")

	headers.Set("X-Ms-Max-Item-Count", "-1")
	if i.continuation != "" {
		headers.Set("If-None-Match", i.continuation)
	}

	err = i.setOptions(i.options, nil, headers)
	if err != nil {
		return
	}

	err = i.do(ctx, http.MethodGet, i.path+"/docs", "docs", i.path, http.StatusOK, nil, &templates, headers)
	if IsErrorStatusCode(err, http.StatusNotModified) {
		err = nil
	}
	if err != nil {
		return
	}

	i.continuation = headers.Get("Etag")

	return
}

func (i *templateChangeFeedIterator) Continuation() string {
	return i.continuation
}

func (i *templateListIterator) Next(ctx context.Context) (templates *pkg.Templates, err error) {
	err = i.NextRaw(ctx, &templates)
	return
}

func (i *templateListIterator) NextRaw(ctx context.Context, raw interface{}) (err error) {
	if i.done {
		return
	}

	headers := http.Header{}
	headers.Set("X-Ms-Max-Item-Count", "-1")
	if i.continuation != "" {
		headers.Set("X-Ms-Continuation", i.continuation)
	}

	err = i.setOptions(i.options, nil, headers)
	if err != nil {
		return
	}

	err = i.do(ctx, http.MethodGet, i.path+"/docs", "docs", i.path, http.StatusOK, nil, &raw, headers)
	if err != nil {
		return
	}

	i.continuation = headers.Get("X-Ms-Continuation")
	i.done = i.continuation == ""

	return
}

func (i *templateListIterator) Continuation() string {
	return i.continuation
}

func (i *templateQueryIterator) Next(ctx context.Context) (templates *pkg.Templates, err error) {
	err = i.NextRaw(ctx, &templates)
	return
}

func (i *templateQueryIterator) NextRaw(ctx context.Context, raw interface{}) (err error) {
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

	err = i.setOptions(i.options, nil, headers)
	if err != nil {
		return
	}

	err = i.do(ctx, http.MethodPost, i.path+"/docs", "docs", i.path, http.StatusOK, &i.query, &raw, headers)
	if err != nil {
		return
	}

	i.continuation = headers.Get("X-Ms-Continuation")
	i.done = i.continuation == ""

	return
}

func (i *templateQueryIterator) Continuation() string {
	return i.continuation
}
