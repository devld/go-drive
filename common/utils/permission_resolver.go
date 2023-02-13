package utils

import (
	"fmt"
	"go-drive/common/types"
	"sort"
)

var rootPath = ""

var privilegedPermMap PermMap

func init() {
	privilegedPermMap = NewPermMap([]types.PathPermission{{
		ID:         0,
		Path:       &rootPath,
		Subject:    types.AnySubject,
		Permission: types.PermissionReadWrite,
		Policy:     types.PolicyAccept,
	}})
}

// PermMap is map of [subject][path]
type PermMap map[string]*PathTreeNode[*pathPermItem]

func NewPermMap(permissions []types.PathPermission) PermMap {
	result := make(PermMap)
	for _, p := range permissions {
		sp, ok := result[p.Subject]
		if !ok {
			sp = NewPathTreeNodeNonLock[*pathPermItem]("")
			result[p.Subject] = sp
		}
		sp.Add(*p.Path, &pathPermItem{
			PathPermission: p,
			depth:          int8(PathDepth(*p.Path)),
		})
	}
	return result
}

func (pm PermMap) filter(subjects []string) PermMap {
	result := make(PermMap, len(subjects))
	for _, s := range subjects {
		if sp, ok := pm[s]; ok {
			result[s] = sp
		}
	}
	return result
}

func (pm PermMap) Filter(session types.Session) PermMap {
	if session.HasUserGroup(types.AdminUserGroup) {
		return privilegedPermMap
	}
	return pm.filter(makeSubjects(session))
}

// ResolvePath resolves permission of the path
func (pm PermMap) ResolvePath(path string) types.Permission {
	items := make([]*pathPermItem, 0)
	for _, p := range pm {
		p.GetCb(path, func(n *PathTreeNode[*pathPermItem]) {
			if n.Data != nil {
				items = append(items, n.Data)
			}
		})
	}
	return resolveAcceptedPermissions(items)
}

// ResolveDescendant resolves permissions of the path's descendant
func (pm PermMap) ResolveDescendant(path string) (types.Permission, bool) {
	result := make([]*pathPermItem, 0)
	for _, p := range pm {
		node, _ := p.Get(path)
		if node == nil {
			continue
		}
		node.Visit(func(n *PathTreeNode[*pathPermItem]) {
			if n.Data != nil {
				result = append(result, n.Data)
			}
		})
	}
	if len(result) == 0 {
		return 0, false
	}
	return resolveAcceptedPermissions(result), true
}

type pathPermItem struct {
	types.PathPermission
	// if depth == -1, this node is a virtual node that holds descendant
	depth int8
}

func (p pathPermItem) String() string {
	return fmt.Sprintf("%s,%s,%d,%d", *p.Path, p.Subject, p.Permission, p.Policy)
}

func makeSubjects(session types.Session) []string {
	subjects := make([]string, 0, 3)
	subjects = append(subjects, types.AnySubject) // Anonymous
	if !session.IsAnonymous() {
		subjects = append(subjects, types.UserSubject(session.User.Username))
		if session.User.Groups != nil {
			for _, g := range session.User.Groups {
				subjects = append(subjects, types.GroupSubject(g.Name))
			}
		}
	}
	return subjects
}

func resolveAcceptedPermissions(items []*pathPermItem) types.Permission {
	sort.Slice(items, func(i, j int) bool { return pathPermissionLess(items[i], items[j]) })
	acceptedPermission := types.PermissionEmpty
	rejectedPermission := types.PermissionEmpty
	for _, item := range items {
		if item.IsAccept() {
			acceptedPermission |= item.Permission & ^rejectedPermission
		}
		if item.IsReject() {
			// acceptedPermission - ( item.Permission(reject) - acceptedPermission )
			acceptedPermission &= ^(item.Permission & (^acceptedPermission))
			rejectedPermission |= item.Permission
		}
	}
	return acceptedPermission
}

func pathPermissionLess(a, b *pathPermItem) bool {
	if a.depth != b.depth {
		return a.depth > b.depth
	}
	if a.IsForAnonymous() {
		if b.IsForAnonymous() {
			return a.Policy < b.Policy
		} else {
			return false
		}
	} else {
		if b.IsForAnonymous() {
			return true
		} else {
			if a.IsForUser() {
				if b.IsForUser() {
					return a.Policy < b.Policy
				} else {
					return true
				}
			} else {
				if b.IsForUser() {
					return false
				} else {
					return a.Policy < b.Policy
				}
			}
		}
	}
}
