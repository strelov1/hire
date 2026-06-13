package linksource

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"golang.org/x/net/html"
)

// fakeClient is a test linksource.Client: it returns a canned HTML body for the first
// route whose match substring is contained in the requested URL, and reports a configured
// final URL (defaulting to the requested URL) so a shortener redirect can be simulated.
type fakeClient struct {
	routes []fakeRoute
}

type fakeRoute struct {
	match string
	body  string
	final string // resolved URL after redirects; "" = same as requested
}

func (c *fakeClient) route(match, body, final string) *fakeClient {
	c.routes = append(c.routes, fakeRoute{match: match, body: body, final: final})
	return c
}

func (c *fakeClient) find(url string) (fakeRoute, bool) {
	for _, r := range c.routes {
		if strings.Contains(url, r.match) {
			return r, true
		}
	}
	return fakeRoute{}, false
}

func (c *fakeClient) GetHTML(_ context.Context, url string) (*html.Node, error) {
	r, ok := c.find(url)
	if !ok {
		return nil, fmt.Errorf("fakeClient: no route for %s", url)
	}
	return html.Parse(strings.NewReader(r.body))
}

func (c *fakeClient) GetHTMLResolved(_ context.Context, url string) (*html.Node, string, error) {
	r, ok := c.find(url)
	if !ok {
		return nil, "", fmt.Errorf("fakeClient: no route for %s", url)
	}
	final := r.final
	if final == "" {
		final = url
	}
	n, err := html.Parse(strings.NewReader(r.body))
	return n, final, err
}

func (c *fakeClient) GetJSON(_ context.Context, url string, v any) error {
	r, ok := c.find(url)
	if !ok {
		return fmt.Errorf("fakeClient: no route for %s", url)
	}
	return json.Unmarshal([]byte(r.body), v)
}
