package utils

import (
	"strings"
	"sync"
)

func NewPathTreeNode[T any](key string) *PathTreeNode[T] {
	return newPathTreeNode[T](key, true)
}

func NewPathTreeNodeNonLock[T any](key string) *PathTreeNode[T] {
	return newPathTreeNode[T](key, false)
}

func newPathTreeNode[T any](key string, locking bool) *PathTreeNode[T] {
	var mu *sync.RWMutex
	if locking {
		mu = &sync.RWMutex{}
	}
	return &PathTreeNode[T]{key: key, children: make(map[string]*PathTreeNode[T]), mu: mu}
}

type PathTreeNode[T any] struct {
	key      string
	children map[string]*PathTreeNode[T]
	mu       *sync.RWMutex

	Data T
}

func (n *PathTreeNode[T]) Key() string {
	return n.key
}

func (n *PathTreeNode[T]) Get(path string) (*PathTreeNode[T], *PathTreeNode[T]) {
	return n.GetCb(path, nil)
}

func (n *PathTreeNode[T]) GetCb(path string, cb func(*PathTreeNode[T])) (*PathTreeNode[T], *PathTreeNode[T]) {
	var node *PathTreeNode[T] = n
	var parent *PathTreeNode[T] = nil
	if IsRootPath(path) {
		if cb != nil {
			cb(node)
		}
		return node, parent
	}
	for _, i := range strings.Split(CleanPath(path), "/") {
		if cb != nil {
			cb(node)
		}
		if node.mu != nil {
			node.mu.RLock()
		}
		t, ok := node.children[i]
		if !ok {
			if node.mu != nil {
				node.mu.RUnlock()
			}
			return nil, nil
		}
		parent = node
		if node.mu != nil {
			node.mu.RUnlock()
		}
		node = t
	}
	if cb != nil {
		cb(node)
	}
	return node, parent
}

func (n *PathTreeNode[T]) Visit(visit func(*PathTreeNode[T])) {
	visit(n)
	for _, node := range n.Children() {
		node.Visit(visit)
	}
}

func (n *PathTreeNode[T]) Create(path string) *PathTreeNode[T] {
	if IsRootPath(path) {
		return n
	}
	node := n
	for _, i := range strings.Split(CleanPath(path), "/") {
		if node.mu != nil {
			node.mu.Lock()
		}
		t, ok := node.children[i]
		if !ok {
			t = newPathTreeNode[T](i, n.mu != nil)
			node.children[i] = t
		}
		if node.mu != nil {
			node.mu.Unlock()
		}
		node = t
	}
	return node
}

func (n *PathTreeNode[T]) Add(path string, data T) *PathTreeNode[T] {
	node := n.Create(path)
	if node.mu != nil {
		node.mu.Lock()
	}
	node.Data = data
	if node.mu != nil {
		node.mu.Unlock()
	}
	return node
}

func (n *PathTreeNode[T]) AddChildren(children map[string]T) {
	if n.mu != nil {
		n.mu.Lock()
	}
	for k, d := range children {
		if strings.Contains(k, "/") {
			panic("PathTreeNode.AddChildren can only be called with single level keys")
		}
		t := newPathTreeNode[T](k, n.mu != nil)
		t.Data = d
		n.children[k] = t
	}
	if n.mu != nil {
		n.mu.Unlock()
	}
}

func (n *PathTreeNode[T]) RemoveChild(key string) {
	if n.mu != nil {
		n.mu.Lock()
	}
	delete(n.children, key)
	if n.mu != nil {
		n.mu.Unlock()
	}
}

func (n *PathTreeNode[T]) Children() []*PathTreeNode[T] {
	if n.mu != nil {
		n.mu.RLock()
		defer n.mu.RUnlock()
	}
	return MapValues(n.children)
}

func (n *PathTreeNode[T]) L() *sync.RWMutex {
	return n.mu
}
