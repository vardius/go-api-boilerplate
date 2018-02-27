package gorouter

import (
	"regexp"
	"strings"
)

type node struct {
	id         string
	regexp     *regexp.Regexp
	route      *route
	parent     *node
	children   *tree
	params     uint8
	isWildcard bool
	isRegexp   bool
}

func (n *node) isRoot() bool {
	return n.parent == nil
}

func (n *node) isLeaf() bool {
	return n.children.idsLen == 0 && len(n.children.regexps) == 0 && n.children.wildcard == nil
}

func (n *node) regexpToString() string {
	if n.regexp == nil {
		return ""
	}
	return n.regexp.String()
}

func (n *node) setRegexp(exp string) {
	reg, err := regexp.Compile(exp)
	if err == nil {
		n.regexp = reg
		n.isRegexp = true
		n.isWildcard = true
	}
}

func (n *node) setRoute(r *route) {
	n.route = r
}

func (n *node) setChildren(children *tree) {
	n.children = children
}

func (n *node) addChild(ids []string) *node {
	if len(ids) > 0 && ids[0] != "" {
		node := n.children.byID(ids[0])

		if node == nil {
			node = newNode(n, ids[0])
			n.children.insert(node)
		}

		return node.addChild(ids[1:])
	}
	return n
}

func (n *node) child(ids []string) (*node, Params) {
	if len(ids) == 0 {
		return n, make(Params, n.params)
	}

	child := n.children.byID(ids[0])
	if child != nil {
		n, params := child.child(ids[1:])

		if child.isWildcard && params != nil {
			params[child.params-1].Value = ids[0]
			params[child.params-1].Key = child.id
		}

		return n, params
	}

	return nil, nil
}

func (n *node) childByPath(path string) (*node, Params) {
	pathLen := len(path)
	if pathLen > 0 && path[0] == '/' {
		path = path[1:]
		pathLen--
	}

	if pathLen == 0 {
		return n, make(Params, n.params)
	}

	child, part, path := n.children.byPath(path)

	if child != nil {
		n, params := child.childByPath(path)

		if part != "" && params != nil {
			params[child.params-1].Value = part
			params[child.params-1].Key = child.id
		}

		return n, params
	}

	return nil, nil
}

func newNode(root *node, id string) *node {
	var regexp string
	isWildcard := false
	isRegexp := false

	if len(id) > 0 && id[0] == '{' {
		id = id[1 : len(id)-1]
		isWildcard = true

		if parts := strings.Split(id, ":"); len(parts) == 2 {
			id = parts[0]
			regexp = parts[1]
			regexp = regexp[:len(regexp)-1]
			isRegexp = true
		}

		if id == "" {
			panic("Empty wildcard name")
		}
	}

	n := &node{
		id:         id,
		parent:     root,
		children:   newTree(),
		isWildcard: isWildcard,
		isRegexp:   isRegexp,
	}

	if root != nil {
		n.params = root.params
	}

	if isWildcard {
		n.params++
	}

	if isRegexp {
		n.setRegexp(regexp)
	}

	return n
}

func newRoot(id string) *node {
	return newNode(nil, id)
}
