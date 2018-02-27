package gorouter

import (
	"strings"
)

type tree struct {
	idsLen   int
	ids      []string
	statics  map[int]*node
	regexps  []*node
	wildcard *node
}

func (t *tree) insert(n *node) {
	if n == nil {
		return
	}

	if n.isWildcard {
		if n.isRegexp {
			t.regexps = append(t.regexps, n)
		} else {
			if t.wildcard != nil {
				panic("Tree already contains a wildcard child!")
			}
			t.wildcard = n
		}
	} else {
		index := -1
		for i, id := range t.ids {
			if n.id > id {
				index = i
				break
			}
		}

		if index > -1 {
			t.ids = append(t.ids[:index], append([]string{n.id}, t.ids[index:]...)...)
			for i := t.idsLen - 1; i >= 0; i-- {
				if i < index {
					break
				} else {
					t.statics[i+1] = t.statics[i]
				}
			}
			t.statics[index] = n
		} else {
			t.ids = append(t.ids, n.id)
			t.statics[t.idsLen] = n
		}

		t.idsLen++
	}
}

func (t *tree) byID(id string) *node {
	if id != "" {
		if t.idsLen > 0 {
			for i, cID := range t.ids {
				if cID == id {
					return t.statics[i]
				}
			}
		}

		for _, child := range t.regexps {
			if child.regexp.MatchString(id) {
				return child
			}
		}

		return t.wildcard
	}

	return nil
}

func (t *tree) byPath(path string) (*node, string, string) {
	if len(path) == 0 {
		return nil, "", ""
	}

	if t.idsLen > 0 {
		for i, cID := range t.ids {
			pLen := len(cID)
			if len(path) >= pLen && cID == path[:pLen] {
				return t.statics[i], "", path[pLen:]
			}
		}
	}

	part := path
	if j := strings.IndexByte(path, '/'); j > 0 {
		part = path[:j]
	}

	for _, child := range t.regexps {
		if child.regexp.MatchString(part) {
			return child, part, path[len(part):]
		}
	}

	if t.wildcard != nil {
		return t.wildcard, part, path[len(part):]
	}

	return nil, "", ""
}

func newTree() *tree {
	return &tree{
		ids:     make([]string, 0),
		statics: make(map[int]*node),
		regexps: make([]*node, 0),
	}
}
