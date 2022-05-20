package utils

import (
	"fmt"
	"go-drive/common/types"
	"sort"
	"strings"
)

var rootPath = ""

var privilegedPermMap = PermMap{
	types.AnySubject: map[string]*PathPermItem{
		rootPath: {
			PathPermission: types.PathPermission{
				ID:         0,
				Path:       &rootPath,
				Subject:    types.AnySubject,
				Permission: types.PermissionReadWrite,
				Policy:     types.PolicyAccept,
			},
		},
	},
}

// PermMap is map of [subject][path]
type PermMap map[string]map[string]*PathPermItem

func NewPermMap(permissions []types.PathPermission) PermMap {
	result := make(PermMap)
	for _, p := range permissions {
		sp, ok := result[p.Subject]
		if !ok {
			sp = make(map[string]*PathPermItem)
			result[p.Subject] = sp
		}
		sp[*p.Path] = &PathPermItem{
			PathPermission: p,
			depth:          int8(PathDepth(*p.Path)),
			descendant:     make([]*PathPermItem, 0),
		}
	}
	for _, pms := range result {
		added := make(map[string]bool)
		for p := range pms {
			paths := PathParentTree(p)
			for _, pathPart := range paths {
				if _, ok := pms[pathPart]; !ok {
					added[pathPart] = true
				}
			}
		}
		for p := range added {
			// virtual path node helps finding path descendant
			pms[p] = &PathPermItem{
				PathPermission: types.PathPermission{Path: &p},
				depth:          -1,
				descendant:     make([]*PathPermItem, 0),
			}
		}
	}
	for _, sp := range result {
		for _, p := range sp {
			for _, c := range sp {
				if c.depth >= 0 && p.depth < c.depth && strings.HasPrefix(*c.Path, *p.Path) {
					p.descendant = append(p.descendant, c)
				}
			}
		}
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
	paths := PathParentTree(path)
	items := make([]*PathPermItem, 0)
	for _, p := range pm {
		for _, pathPart := range paths {
			if temp, ok := p[pathPart]; ok && temp.depth >= 0 {
				items = append(items, temp)
			}
		}
	}
	return resolveAcceptedPermissions(items)
}

// ResolveDescendant resolves permissions of the path's descendant
func (pm PermMap) ResolveDescendant(path string) map[string]types.Permission {
	result := make(map[string]types.Permission)
	for _, p := range pm {
		if item, ok := p[path]; ok {
			for _, t := range item.descendant {
				result[*item.Path] = resolveAcceptedPermissions(t.descendant)
			}
		}
	}
	return result
}

type PathPermItem struct {
	types.PathPermission
	// if depth == -1, this node is a virtual node that holds descendant
	depth      int8
	descendant []*PathPermItem
}

func (p PathPermItem) String() string {
	return fmt.Sprintf("%s,%s,%d,%d (%v)", *p.Path, p.Subject, p.Permission, p.Policy, p.descendant)
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

func resolveAcceptedPermissions(items []*PathPermItem) types.Permission {
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

func pathPermissionLess(a, b *PathPermItem) bool {
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
