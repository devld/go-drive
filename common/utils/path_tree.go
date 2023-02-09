package utils

import (
	"strings"
	"sync"
)

func NewPathTreeNode[T any](key string) *PathTreeNode[T] {
	return &PathTreeNode[T]{key: key, children: make(map[string]*PathTreeNode[T])}
}

type PathTreeNode[T any] struct {
	key      string
	children map[string]*PathTreeNode[T]
	mu       sync.RWMutex

	Data T
}

func (n *PathTreeNode[T]) Key() string {
	return n.key
}

func (n *PathTreeNode[T]) Get(path string) (*PathTreeNode[T], *PathTreeNode[T]) {
	var node *PathTreeNode[T] = n
	var parent *PathTreeNode[T] = nil
	if IsRootPath(path) {
		return node, parent
	}
	for _, i := range strings.Split(CleanPath(path), "/") {
		node.mu.RLock()
		t, ok := node.children[i]
		if !ok {
			node.mu.RUnlock()
			return nil, nil
		}
		parent = node
		node.mu.RUnlock()
		node = t
	}
	return node, parent
}

func (n *PathTreeNode[T]) Create(path string) *PathTreeNode[T] {
	node := n
	for _, i := range strings.Split(CleanPath(path), "/") {
		node.mu.Lock()
		t, ok := node.children[i]
		if !ok {
			t = NewPathTreeNode[T](i)
			node.children[i] = t
		}
		node.mu.Unlock()
		node = t
	}
	return node
}

func (n *PathTreeNode[T]) AddChild(path string, data T) {
	node := n.Create(path)
	node.L().Lock()
	node.Data = data
	node.L().Unlock()
}

func (n *PathTreeNode[T]) AddChildren(children map[string]T) {
	n.L().Lock()
	for k, d := range children {
		if strings.Contains(k, "/") {
			panic("PathTreeNode.AddChildren can only be called with single level keys")
		}
		t := NewPathTreeNode[T](k)
		t.Data = d
		n.children[k] = t
	}
	n.L().Unlock()
}

func (n *PathTreeNode[T]) RemoveChild(key string) {
	n.mu.Lock()
	delete(n.children, key)
	n.mu.Unlock()
}

func (n *PathTreeNode[T]) Children() []*PathTreeNode[T] {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return MapValues(n.children)
}

func (n *PathTreeNode[T]) L() *sync.RWMutex {
	return &n.mu
}
