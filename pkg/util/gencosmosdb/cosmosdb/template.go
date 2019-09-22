package cosmosdb

import (
	"net/http"

	pkg "github.com/jim-minter/go-cosmosdb/pkg/gencosmosdb/cosmosdb/dummy"
)

type templateClient struct {
	*databaseClient
	path          string
	isPartitioned bool
}

// TemplateClient is a template client
type TemplateClient interface {
	Create(string, *pkg.Template) (*pkg.Template, error)
	List() TemplateIterator
	ListAll() (*pkg.Templates, error)
	Get(string, string) (*pkg.Template, error)
	Replace(string, *pkg.Template) (*pkg.Template, error)
	Delete(string, *pkg.Template) error
	Query(*Query) TemplateIterator
	QueryAll(*Query) (*pkg.Templates, error)
}

type templateListIterator struct {
	*templateClient
	continuation string
	done         bool
}

type templateQueryIterator struct {
	*templateClient
	query        *Query
	continuation string
	done         bool
}

// TemplateIterator is a template iterator
type TemplateIterator interface {
	Next() (*pkg.Templates, error)
}

// NewTemplateClient returns a new template client
func NewTemplateClient(collc CollectionClient, collid string, isPartitioned bool) TemplateClient {
	return &templateClient{
		databaseClient: collc.(*collectionClient).databaseClient,
		path:           collc.(*collectionClient).path + "/colls/" + collid,
		isPartitioned:  isPartitioned,
	}
}

func (c *templateClient) all(i TemplateIterator) (*pkg.Templates, error) {
	alltemplates := &pkg.Templates{}

	for {
		templates, err := i.Next()
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

func (c *templateClient) Create(partitionkey string, newtemplate *pkg.Template) (template *pkg.Template, err error) {
	headers := http.Header{}
	if partitionkey != "" {
		headers.Set("X-Ms-Documentdb-Partitionkey", `["`+partitionkey+`"]`)
	}
	err = c.do(http.MethodPost, c.path+"/docs", "docs", c.path, http.StatusCreated, &newtemplate, &template, headers)
	return
}

func (c *templateClient) List() TemplateIterator {
	return &templateListIterator{templateClient: c}
}

func (c *templateClient) ListAll() (*pkg.Templates, error) {
	return c.all(c.List())
}

func (c *templateClient) Get(partitionkey, templateid string) (template *pkg.Template, err error) {
	headers := http.Header{}
	if partitionkey != "" {
		headers.Set("X-Ms-Documentdb-Partitionkey", `["`+partitionkey+`"]`)
	}
	err = c.do(http.MethodGet, c.path+"/docs/"+templateid, "docs", c.path+"/docs/"+templateid, http.StatusOK, nil, &template, headers)
	return
}

func (c *templateClient) Replace(partitionkey string, newtemplate *pkg.Template) (template *pkg.Template, err error) {
	if newtemplate.ETag == "" {
		return nil, ErrETagRequired
	}
	headers := http.Header{}
	headers.Set("If-Match", newtemplate.ETag)
	if partitionkey != "" {
		headers.Set("X-Ms-Documentdb-Partitionkey", `["`+partitionkey+`"]`)
	}
	err = c.do(http.MethodPut, c.path+"/docs/"+newtemplate.ID, "docs", c.path+"/docs/"+newtemplate.ID, http.StatusOK, &newtemplate, &template, headers)
	return
}

func (c *templateClient) Delete(partitionkey string, template *pkg.Template) error {
	if template.ETag == "" {
		return ErrETagRequired
	}
	headers := http.Header{}
	headers.Set("If-Match", template.ETag)
	if partitionkey != "" {
		headers.Set("X-Ms-Documentdb-Partitionkey", `["`+partitionkey+`"]`)
	}
	return c.do(http.MethodDelete, c.path+"/docs/"+template.ID, "docs", c.path+"/docs/"+template.ID, http.StatusNoContent, nil, nil, headers)
}

func (c *templateClient) Query(query *Query) TemplateIterator {
	return &templateQueryIterator{templateClient: c, query: query}
}

func (c *templateClient) QueryAll(query *Query) (*pkg.Templates, error) {
	return c.all(c.Query(query))
}

func (i *templateListIterator) Next() (templates *pkg.Templates, err error) {
	if i.done {
		return
	}

	headers := http.Header{}
	if i.continuation != "" {
		headers.Set("X-Ms-Continuation", i.continuation)
	}

	err = i.do(http.MethodGet, i.path+"/docs", "docs", i.path, http.StatusOK, nil, &templates, headers)
	if err != nil {
		return
	}

	i.continuation = headers.Get("X-Ms-Continuation")
	i.done = i.continuation == ""

	return
}

func (i *templateQueryIterator) Next() (templates *pkg.Templates, err error) {
	if i.done {
		return
	}

	headers := http.Header{}
	headers.Set("X-Ms-Documentdb-Isquery", "True")
	headers.Set("Content-Type", "application/query+json")
	if i.isPartitioned {
		headers.Set("X-Ms-Documentdb-Query-Enablecrosspartition", "True")
	}
	if i.continuation != "" {
		headers.Set("X-Ms-Continuation", i.continuation)
	}

	err = i.do(http.MethodPost, i.path+"/docs", "docs", i.path, http.StatusOK, &i.query, &templates, headers)
	if err != nil {
		return
	}

	i.continuation = headers.Get("X-Ms-Continuation")
	i.done = i.continuation == ""

	return
}
