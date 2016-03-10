// Package util provides utility functions for the casper package.
package util

import (
	"net/url"
	"sort"
)

// sortURLMap sorts a given url.Values map m alphabetically by it's keys, whilst retaining the values.
func sortURLMap(m url.Values) string {
	var keys []string
	var sortedParamString string

	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		v := m[k]
		sortedParamString += k + v[0]
	}
	return sortedParamString
}
