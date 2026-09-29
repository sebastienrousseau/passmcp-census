// SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
// SPDX-License-Identifier: Apache-2.0

package aggregate

import "strings"

// A Frictionless Data Package descriptor
// (https://specs.frictionlessdata.io/data-package/), so the dataset can be
// validated and loaded by tools that know nothing about the census.
type dataPackage struct {
	Profile     string     `json:"profile"`
	Name        string     `json:"name"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Version     string     `json:"version"`
	Created     string     `json:"created"`
	Homepage    string     `json:"homepage"`
	Licenses    []license  `json:"licenses"`
	Sources     []source   `json:"sources"`
	Resources   []resource `json:"resources"`
}

type license struct {
	Name  string `json:"name"`
	Path  string `json:"path"`
	Title string `json:"title"`
}

type source struct {
	Title string `json:"title"`
	Path  string `json:"path"`
}

type resource struct {
	Name      string  `json:"name"`
	Path      string  `json:"path"`
	Profile   string  `json:"profile"`
	Format    string  `json:"format"`
	MediaType string  `json:"mediatype"`
	Encoding  string  `json:"encoding"`
	Schema    *schema `json:"schema,omitempty"`
}

type schema struct {
	Fields []field `json:"fields"`
}

type field struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

// integerFields are the count columns; rates are numbers; the rest strings.
var integerFields = map[string]bool{
	"endpoints": true, "pass": true, "warn": true, "fail": true, "info": true, "skip": true,
	"reports": true, "errors": true, "timeouts": true, "blocked": true, "with_failure": true, "count": true,
}

func (d Dataset) descriptor() dataPackage {
	p := dataPackage{
		Profile: "tabular-data-package",
		Name:    "passmcp-census-" + d.Edition,
		Title:   "passmcp reliability census, edition " + d.Edition,
		Description: "Aggregate results of passmcp " + d.Run.PassmcpVersion + " run read-only and without credentials " +
			"against the remote servers listed in the MCP Registry. Counts and rates only; no server is identified.",
		Version:  d.Run.CensusVersion,
		Created:  d.Run.RunFinished.UTC().Format("2006-01-02T15:04:05Z"),
		Homepage: "https://github.com/sebastienrousseau/passmcp-census",
		Licenses: []license{{Name: "CC-BY-4.0", Path: "https://creativecommons.org/licenses/by/4.0/", Title: "Creative Commons Attribution 4.0"}},
		Sources:  []source{{Title: "MCP Registry", Path: d.Run.Registry}},
	}
	for _, t := range d.tables() {
		p.Resources = append(p.Resources, resource{
			Name: strings.TrimSuffix(t.name, ".csv"), Path: t.name, Profile: "tabular-data-resource",
			Format: "csv", MediaType: "text/csv", Encoding: "utf-8", Schema: fieldsOf(t.header),
		})
	}
	p.Resources = append(p.Resources, resource{
		Name: "census", Path: CensusFile, Profile: "data-resource",
		Format: "json", MediaType: "application/json", Encoding: "utf-8",
	})
	return p
}

func fieldsOf(header []string) *schema {
	s := &schema{}
	for _, h := range header {
		typ := "string"
		switch {
		case integerFields[h]:
			typ = "integer"
		case strings.HasSuffix(h, "rate") || h == "share":
			typ = "number"
		}
		s.Fields = append(s.Fields, field{Name: h, Type: typ})
	}
	return s
}
