// SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
// SPDX-License-Identifier: Apache-2.0

// Package registry lists the remote MCP servers an MCP Registry publishes,
// through the registry's v0 servers API, by the selection passmcp-registry
// documents: the remote (HTTP) endpoints of the latest version of every
// active server, less templated URLs, URLs carrying credentials and
// endpoints that need headers the user supplies.
//
// Two entry shapes are read, because the official registry has served both:
// the server document itself with registry metadata under "_meta", and the
// wrapped form {"server": …, "_meta": …}. The next page is named by
// metadata.nextCursor or metadata.next_cursor.
package registry

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// officialKey is the "_meta" key the official registry files its own
// metadata under.
const officialKey = "io.modelcontextprotocol.registry/official"

// Bounds on what one listing may cost.
const (
	maxPages    = 1000
	maxPageBody = 16 << 20
)

// Skip reasons, as categories: the census publishes how many endpoints were
// set aside for each, never which.
const (
	ReasonTransport   = "not a remote HTTP transport"
	ReasonTemplate    = "templated URL"
	ReasonNotHTTP     = "not an http(s) URL"
	ReasonCredentials = "URL carries credentials" // #nosec G101 -- a reason label, not a credential
	ReasonHeaders     = "requires headers the user supplies"
)

// Endpoint is one remote endpoint a listed server declares.
type Endpoint struct {
	Name      string `json:"name"`    // the server's registry name
	Version   string `json:"version"` // the server version listed
	Transport string `json:"transport"`
	URL       string `json:"url"`
}

// Skipped is a listed endpoint that is not checked, with its reason
// category.
type Skipped struct {
	Endpoint
	Reason string `json:"reason"`
}

// Listing is what one pass over the registry found.
type Listing struct {
	Entries   int        `json:"entries"`   // entries the registry returned
	Current   int        `json:"current"`   // latest versions of active servers
	Endpoints []Endpoint `json:"endpoints"` // checkable endpoints
	Skipped   []Skipped  `json:"skipped"`
}

// Client reads one registry.
type Client struct {
	Base      string // e.g. https://registry.modelcontextprotocol.io
	HTTP      *http.Client
	UserAgent string
	PageSize  int
	// Retries is how many more times a page is asked for after a network
	// error, a 429 or a 5xx; Backoff is the first wait, doubled each time.
	// A long listing meets transient errors, and giving up on one discards
	// every page before it.
	Retries int
	Backoff time.Duration
	// Progress, when set, gets one line per page read.
	Progress io.Writer
}

// errTransient marks a page failure worth asking again for.
var errTransient = errors.New("transient")

type page struct {
	Servers  []json.RawMessage `json:"servers"`
	Metadata struct {
		NextCursor      string `json:"nextCursor"`
		NextCursorSnake string `json:"next_cursor"`
	} `json:"metadata"`
}

type remote struct {
	Type          string `json:"type"`
	TransportType string `json:"transport_type"`
	URL           string `json:"url"`
	Headers       []struct {
		Name       string `json:"name"`
		IsRequired bool   `json:"isRequired"`
	} `json:"headers"`
}

type document struct {
	Name    string                     `json:"name"`
	Version string                     `json:"version"`
	Status  string                     `json:"status"`
	Remotes []remote                   `json:"remotes"`
	Meta    map[string]json.RawMessage `json:"_meta"`
}

type envelope struct {
	Server *document                  `json:"server"`
	Meta   map[string]json.RawMessage `json:"_meta"`
}

type officialMeta struct {
	Status   string `json:"status"`
	IsLatest *bool  `json:"isLatest"`
}

// List pages through the registry.
func (c *Client) List(ctx context.Context) (Listing, error) {
	var out Listing
	cursor := ""
	for i := range maxPages {
		p, err := c.page(ctx, cursor)
		if err != nil {
			return out, err
		}
		for _, raw := range p.Servers {
			out.add(raw)
		}
		if c.Progress != nil {
			_, _ = fmt.Fprintf(c.Progress, "registry: page %d, %d entries so far\n", i+1, out.Entries)
		}
		next := p.Metadata.NextCursor
		if next == "" {
			next = p.Metadata.NextCursorSnake
		}
		if next == "" || next == cursor {
			return out, nil
		}
		cursor = next
	}
	return out, fmt.Errorf("registry: more than %d pages", maxPages)
}

// page fetches one page, asking again after a transient failure.
func (c *Client) page(ctx context.Context, cursor string) (page, error) {
	u, err := c.pageURL(cursor)
	if err != nil {
		return page{}, err
	}
	wait := c.Backoff
	for attempt := 0; ; attempt++ {
		p, err := c.fetch(ctx, u)
		if err == nil || !errors.Is(err, errTransient) || attempt >= c.Retries {
			return p, err
		}
		t := time.NewTimer(wait)
		select {
		case <-ctx.Done():
			t.Stop()
			return page{}, ctx.Err()
		case <-t.C:
		}
		wait *= 2
	}
}

func (c *Client) fetch(ctx context.Context, u string) (page, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return page{}, fmt.Errorf("registry: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	if c.UserAgent != "" {
		req.Header.Set("User-Agent", c.UserAgent)
	}
	hc := c.HTTP
	if hc == nil {
		hc = http.DefaultClient
	}
	resp, err := hc.Do(req)
	if err != nil {
		return page{}, fmt.Errorf("registry: %w: %w", errTransient, err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
		return page{}, fmt.Errorf("registry: %w: page answered HTTP %d", errTransient, resp.StatusCode)
	}
	if resp.StatusCode != http.StatusOK {
		return page{}, fmt.Errorf("registry: page answered HTTP %d", resp.StatusCode)
	}
	return decodePage(resp.Body)
}

// pageURL is the v0 servers URL for one page.
func (c *Client) pageURL(cursor string) (string, error) {
	u, err := url.Parse(strings.TrimRight(c.Base, "/") + "/v0/servers")
	if err != nil || u.Host == "" {
		return "", errors.New("registry: the base URL is not an absolute URL")
	}
	size := c.PageSize
	if size <= 0 {
		size = 100
	}
	// version=latest asks the registry for the latest version of each
	// server only: a small fraction of the pages, and so of the load a
	// listing puts on the registry. isCurrent still checks every entry, so
	// a registry that ignores the parameter yields the same selection.
	q := url.Values{"limit": {fmt.Sprint(size)}, "version": {"latest"}}
	if cursor != "" {
		q.Set("cursor", cursor)
	}
	u.RawQuery = q.Encode()
	return u.String(), nil
}

func decodePage(r io.Reader) (page, error) {
	var p page
	body, err := io.ReadAll(io.LimitReader(r, maxPageBody+1))
	if err != nil {
		return p, fmt.Errorf("registry: %w", err)
	}
	if len(body) > maxPageBody {
		return p, errors.New("registry: a page exceeded 16 MiB")
	}
	if err := json.Unmarshal(body, &p); err != nil {
		return p, fmt.Errorf("registry: not a v0 servers page: %w", err)
	}
	return p, nil
}

// add files one entry, in either shape.
func (l *Listing) add(raw json.RawMessage) {
	doc, meta, ok := parseEntry(raw)
	if !ok {
		return
	}
	l.Entries++
	if !isCurrent(doc.Status, meta) {
		return
	}
	l.Current++
	for _, r := range doc.Remotes {
		ep := Endpoint{Name: doc.Name, Version: doc.Version, Transport: r.transport(), URL: r.URL}
		if reason := r.unusable(); reason != "" {
			l.Skipped = append(l.Skipped, Skipped{Endpoint: ep, Reason: reason})
			continue
		}
		l.Endpoints = append(l.Endpoints, ep)
	}
}

func parseEntry(raw json.RawMessage) (*document, map[string]json.RawMessage, bool) {
	var env envelope
	if err := json.Unmarshal(raw, &env); err != nil {
		return nil, nil, false
	}
	doc, meta := env.Server, env.Meta
	if doc == nil {
		var d document
		if err := json.Unmarshal(raw, &d); err != nil {
			return nil, nil, false
		}
		doc, meta = &d, d.Meta
	}
	return doc, meta, doc.Name != ""
}

// isCurrent reports whether an entry is the latest version of an active
// server: the census is about what an agent would connect to today.
func isCurrent(status string, meta map[string]json.RawMessage) bool {
	if raw, ok := meta[officialKey]; ok {
		var off officialMeta
		_ = json.Unmarshal(raw, &off)
		if off.Status != "" {
			status = off.Status
		}
		if off.IsLatest != nil && !*off.IsLatest {
			return false
		}
	}
	return status == "" || status == "active"
}

func (r remote) transport() string {
	if r.Type != "" {
		return r.Type
	}
	return r.TransportType
}

// unusable is the reason category an endpoint cannot be checked
// unauthenticated and unconfigured, or "".
func (r remote) unusable() string {
	if t := r.transport(); t != "streamable-http" && t != "sse" {
		return ReasonTransport
	}
	if reason := urlReason(r.URL); reason != "" {
		return reason
	}
	for _, h := range r.Headers {
		if h.IsRequired {
			return ReasonHeaders
		}
	}
	return ""
}

// urlReason is why a URL cannot be connected to as listed, or "".
func urlReason(raw string) string {
	if strings.ContainsAny(raw, "{}") {
		return ReasonTemplate
	}
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" || (u.Scheme != "https" && u.Scheme != "http") {
		return ReasonNotHTTP
	}
	if u.User != nil {
		return ReasonCredentials
	}
	return ""
}

// Distinct returns the endpoints with each URL once, keeping the first
// listing of it, and how many duplicates were dropped. Several listed
// servers can share one URL; checking it once is both fairer to its
// operator and the honest unit for a census of endpoints.
func Distinct(eps []Endpoint) ([]Endpoint, int) {
	seen := make(map[string]bool, len(eps))
	out := make([]Endpoint, 0, len(eps))
	for _, e := range eps {
		if seen[e.URL] {
			continue
		}
		seen[e.URL] = true
		out = append(out, e)
	}
	return out, len(eps) - len(out)
}
