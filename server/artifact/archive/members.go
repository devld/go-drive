package archive

import (
	"fmt"
	apierr "go-drive/common/errors"
	"io/fs"
	pathpkg "path"
	"strings"
)

type extractItem struct {
	path  string
	isDir bool
	size  int64
}

func selectedMemberSet(members []string) (map[string]struct{}, error) {
	if len(members) == 0 {
		return nil, apiBadRequest("")
	}
	selected := make(map[string]struct{}, len(members))
	for _, member := range members {
		normalized, err := normalizeMember(member)
		if err != nil {
			return nil, err
		}
		selected[normalized] = struct{}{}
	}
	return selected, nil
}

func (s *Previewer) collectExtractItems(archiveFS fs.FS, selected map[string]struct{}) ([]extractItem, error) {
	items := make([]extractItem, 0)
	hit := make(map[string]struct{}, len(selected))
	err := fs.WalkDir(archiveFS, ".", func(name string, item fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if name == "." {
			return nil
		}
		if !validArchivePath(name) {
			return fmt.Errorf("unsafe archive path %q", name)
		}
		if !memberMatches(name, selected) {
			return nil
		}
		markSelectedHits(name, selected, hit)
		if len(items) >= s.maxEntries {
			return notFound(msgArchiveTooManyEntries)
		}
		info, e := item.Info()
		if e != nil {
			return e
		}
		items = append(items, extractItem{path: name, isDir: info.IsDir(), size: info.Size()})
		return nil
	})
	if err != nil {
		return nil, invalidArchiveError(err)
	}
	if len(hit) != len(selected) {
		return nil, notFound(msgMemberNotFound)
	}
	if len(items) == 0 {
		return nil, notFound(msgMemberNotFound)
	}
	return items, nil
}

func memberMatches(path string, selected map[string]struct{}) bool {
	if _, ok := selected[path]; ok {
		return true
	}
	for prefix := range selected {
		if strings.HasPrefix(path, prefix+"/") {
			return true
		}
	}
	return false
}

func markSelectedHits(path string, selected, hit map[string]struct{}) {
	if _, ok := selected[path]; ok {
		hit[path] = struct{}{}
	}
	for prefix := range selected {
		if path == prefix || strings.HasPrefix(path, prefix+"/") {
			hit[prefix] = struct{}{}
		}
	}
}

func normalizeMember(name string) (string, error) {
	if strings.ContainsRune(name, '\x00') || strings.ContainsRune(name, '\\') {
		return "", notFound("invalid archive path")
	}
	name = pathpkg.Clean(strings.TrimPrefix(name, "/"))
	if name == "." || !validArchivePath(name) {
		return "", notFound("invalid archive path")
	}
	return name, nil
}

func validArchivePath(name string) bool {
	return name != "." && name != "" &&
		!pathpkg.IsAbs(name) && name != ".." &&
		!strings.HasPrefix(name, "../") &&
		!strings.ContainsRune(name, '\x00') &&
		!strings.ContainsRune(name, '\\')
}

func apiBadRequest(msg string) error {
	return apierr.NewBadRequestError(msg)
}
