package sources

import (
	"context"
	"encoding/json"
	"encoding/xml"
)

// fakeHTTP is a test HTTPClient: it records the requested URL and decodes a canned
// body, so adapter tests exercise field mapping without the network. Shared by every
// adapter test in this package.
type fakeHTTP struct {
	body   string
	err    error
	gotURL string
}

func (f *fakeHTTP) GetJSON(_ context.Context, url string, v any) error {
	f.gotURL = url
	if f.err != nil {
		return f.err
	}
	return json.Unmarshal([]byte(f.body), v)
}

func (f *fakeHTTP) GetXML(_ context.Context, url string, v any) error {
	f.gotURL = url
	if f.err != nil {
		return f.err
	}
	return xml.Unmarshal([]byte(f.body), v)
}

func (f *fakeHTTP) PostJSON(_ context.Context, url string, _, v any) error {
	f.gotURL = url
	if f.err != nil {
		return f.err
	}
	return json.Unmarshal([]byte(f.body), v)
}
