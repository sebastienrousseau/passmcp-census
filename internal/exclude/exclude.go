// SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
// SPDX-License-Identifier: Apache-2.0

// Package exclude reads the census's exclusion list: the registry names of
// servers whose owners asked not to be contacted. One name per line, or a
// namespace ending in "/*" for every server under it; blank lines and lines
// starting with "#" are ignored. The format is passmcp-registry's opt-out
// list, so an owner who opted out there can be carried over as is.
package exclude

import (
	"bufio"
	"io"
	"strings"
)

// List is a parsed exclusion list.
type List struct {
	names      map[string]bool
	namespaces []string
}

// Parse reads an exclusion list.
func Parse(r io.Reader) (*List, error) {
	l := &List{names: map[string]bool{}}
	sc := bufio.NewScanner(r)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		switch {
		case line == "" || strings.HasPrefix(line, "#"):
		case strings.HasSuffix(line, "/*"):
			l.namespaces = append(l.namespaces, strings.TrimSuffix(line, "*"))
		default:
			l.names[line] = true
		}
	}
	return l, sc.Err()
}

// Len is the number of entries in the list.
func (l *List) Len() int {
	if l == nil {
		return 0
	}
	return len(l.names) + len(l.namespaces)
}

// Excludes reports whether the server named name is on the list.
func (l *List) Excludes(name string) bool {
	if l == nil {
		return false
	}
	if l.names[name] {
		return true
	}
	for _, ns := range l.namespaces {
		if strings.HasPrefix(name, ns) {
			return true
		}
	}
	return false
}
