package prometheus

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"bytes"
	"context"
	"io"
	"mime"
	"net"
	"net/http"
	"strconv"
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"

	"github.com/Azure/ARO-RP/pkg/api/validate"
	"github.com/Azure/ARO-RP/pkg/portal/util/responsewriter"
	"github.com/Azure/ARO-RP/pkg/util/portforward"
	"github.com/Azure/ARO-RP/pkg/util/restconfig"
)

// Unfortunately the signature of httputil.ReverseProxy.Director does not allow
// us to return errors.  We get around this limitation slightly naughtily by
// storing return information in the request context.

type contextKey int

const (
	contextKeyClient contextKey = iota
	contextKeyResponse
)

// Director modifies the request to point to the clusters prometheus instance
func (p *Prometheus) Director(r *http.Request) {
	ctx := r.Context()

	resourceID := strings.Join(strings.Split(r.URL.Path, "/")[:9], "/")
	if !validate.RxClusterID.MatchString(resourceID) {
		p.error(r, http.StatusBadRequest, nil)
		return
	}

	cli := p.clientCache.Get(resourceID)
	if cli == nil {
		var err error
		cli, err = p.cli(ctx, resourceID)
		if err != nil {
			p.error(r, http.StatusInternalServerError, err)
			return
		}

		p.clientCache.Put(resourceID, cli)
	}

	r.RequestURI = ""
	r.URL.Scheme = "http"
	r.URL.Host = "prometheus-k8s-0:9090"
	r.URL.Path = "/" + strings.Join(strings.Split(r.URL.Path, "/")[10:], "/")
	r.Header.Del("Cookie")
	r.Header.Del("Referer")
	r.Host = r.URL.Host

	// http.Request.WithContext returns a copy of the original Request with the
	// new context, but we have no way to return it, so we overwrite our
	// existing request.
	*r = *r.WithContext(context.WithValue(ctx, contextKeyClient, cli))
}

func (p *Prometheus) cli(ctx context.Context, resourceID string) (*http.Client, error) {
	openShiftDoc, err := p.dbOpenShiftClusters.Get(ctx, resourceID)
	if err != nil {
		return nil, err
	}

	restconfig, err := restconfig.RestConfig(p.dialer, openShiftDoc.OpenShiftCluster)
	if err != nil {
		return nil, err
	}

	return &http.Client{
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
				return portforward.DialContext(ctx, p.log, restconfig, "openshift-monitoring", "prometheus-k8s-0", "9090")
			},
		},
	}, nil
}

func (p *Prometheus) RoundTripper(r *http.Request) (*http.Response, error) {
	if resp, ok := r.Context().Value(contextKeyResponse).(*http.Response); ok {
		return resp, nil
	}

	cli := r.Context().Value(contextKeyClient).(*http.Client)
	return cli.Do(r)
}

// ModifyResponse: unfortunately Prometheus serves HTML files containing just a
// couple of absolute links.  Given that we're serving Prometheus under
// /subscriptions/.../clusterName/prometheus, we need to dig these out and
// rewrite them.  This is a hack which hopefully goes away once we forward all
// metrics to Kusto.
func (p *Prometheus) ModifyResponse(r *http.Response) error {
	mediaType, _, _ := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if mediaType != "text/html" {
		return nil
	}

	b, err := io.ReadAll(r.Body)
	if err != nil {
		return err
	}

	buf := &bytes.Buffer{}

	n, err := html.Parse(bytes.NewReader(b))
	if err != nil {
		buf.Write(b)
	} else {
		// walk the HTML parse tree calling makeRelative() on each node
		walk(n, makeRelative)

		err = html.Render(buf, n)
		if err != nil {
			return err
		}

		r.Header.Set("Content-Length", strconv.FormatInt(int64(buf.Len()), 10))
	}

	r.Body = io.NopCloser(buf)

	return nil
}

func makeRelative(n *html.Node) {
	switch n.DataAtom {
	case atom.A, atom.Link:
		// rewrite <a href="/foo"> -> <a href="./foo">
		// rewrite <link href="/foo"> -> <link href="./foo">
		for i, attr := range n.Attr {
			if attr.Namespace == "" && attr.Key == "href" && strings.HasPrefix(n.Attr[i].Val, "/") {
				n.Attr[i].Val = "." + n.Attr[i].Val
			}
		}
	case atom.Script:
		// rewrite <script src="/foo"> -> <script src="./foo">
		for i, attr := range n.Attr {
			if attr.Namespace == "" && attr.Key == "src" && strings.HasPrefix(n.Attr[i].Val, "/") {
				n.Attr[i].Val = "." + n.Attr[i].Val
			}
		}

		// special hack: find <script>...</script> and rewrite
		// `var PATH_PREFIX = "";` -> `var PATH_PREFIX = ".";` once.
		if len(n.Attr) == 0 {
			n.FirstChild.Data = strings.Replace(n.FirstChild.Data, `var PATH_PREFIX = "";`, `var PATH_PREFIX = ".";`, 1)
		}
	}
}

func walk(n *html.Node, f func(*html.Node)) {
	f(n)

	for c := n.FirstChild; c != nil; c = c.NextSibling {
		walk(c, f)
	}
}

func (p *Prometheus) error(r *http.Request, statusCode int, err error) {
	if err != nil {
		p.log.Print(err)
	}

	w := responsewriter.New(r)
	http.Error(w, http.StatusText(statusCode), statusCode)

	*r = *r.WithContext(context.WithValue(r.Context(), contextKeyResponse, w.Response()))
}
