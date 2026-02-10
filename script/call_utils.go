package script

import (
	"go-drive/common/drive_util"
	"go-drive/common/types"
	"go-drive/common/utils"
)

type entryTreeNode struct {
	Entry    Entry
	Children []entryTreeNode
	Excluded bool
}

func convertEntryTreeNode(vm *VM, root drive_util.EntryTreeNode) entryTreeNode {
	var children []entryTreeNode
	if root.Children != nil {
		children = make([]entryTreeNode, 0, len(root.Children))
		for _, e := range root.Children {
			children = append(children, convertEntryTreeNode(vm, e))
		}
	}
	return entryTreeNode{NewEntry(vm, root.Entry), children, root.Excluded}
}

func flattenEntriesTree(root entryTreeNode, result []entryTreeNode, deepFirst bool) []entryTreeNode {
	if !deepFirst && !root.Excluded {
		result = append(result, root)
	}
	for _, e := range root.Children {
		result = flattenEntriesTree(e, result, deepFirst)
	}
	if deepFirst && !root.Excluded {
		result = append(result, root)
	}
	return result
}

// vm_buildEntriesTree: (ctx TaskCtx, entry Entry, byteProgress bool) entryTreeNode
func vm_buildEntriesTree(vm *VM, args Values) any {
	ctx := GetTaskCtx(args.Get(0).Raw())
	entry := GetEntry(args.Get(1).Raw())
	byteProgress := args.Get(2).Bool()
	r, e := drive_util.BuildEntriesTree(ctx, entry, byteProgress)
	if e != nil {
		vm.ThrowError(e)
	}
	return convertEntryTreeNode(vm, r)
}

// vm_buildEntriesTreeWithPattern: (ctx TaskCtx, root Drive, pattern string, bytesProgress bool) []Entry
func vm_findEntries(vm *VM, args Values) any {
	ctx := GetTaskCtx(args.Get(0).Raw())
	drive := GetDrive(args.Get(1).Raw())
	pattern := args.Get(2).String()
	byteProgress := args.Get(3).Bool()
	r, e := drive_util.FindEntries(ctx, drive, pattern, byteProgress)
	if e != nil {
		vm.ThrowError(e)
	}
	return utils.ArrayMap(r, func(t *types.IEntry) Entry { return NewEntry(vm, *t) })
}

// vm_flattenEntriesTree: (root entryTreeNode, deepFirst bool) []entryTreeNode
func vm_flattenEntriesTree(vm *VM, args Values) any {
	entry, ok := args.Get(0).Raw().(entryTreeNode)
	if !ok {
		vm.ThrowTypeError("not a EntryTreeNode")
	}
	deepFirst := args.Get(1).Bool()
	r := make([]entryTreeNode, 0)
	return flattenEntriesTree(entry, r, deepFirst)
}
