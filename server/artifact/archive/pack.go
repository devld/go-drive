package archive

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"go-drive/common/driveutil"
	"go-drive/common/task"
	"go-drive/common/types"
	"go-drive/server/artifact"
	pathpkg "path"
	"sort"
	"strings"
	"time"
)

// parsePackMembers reads a pack:<json array> args value. The returned key is a
// hash of the canonical member list, so the cache slot does not store that list.
func parsePackMembers(args string) (members []string, key string, err error) {
	if !strings.HasPrefix(args, argsPackPrefix) {
		return nil, "", notFound("invalid archive artifact args")
	}
	var raw []string
	if jsonErr := json.Unmarshal([]byte(strings.TrimPrefix(args, argsPackPrefix)), &raw); jsonErr != nil {
		return nil, "", notFound("invalid archive artifact args")
	}
	selected, err := selectedMemberSet(raw)
	if err != nil {
		return nil, "", err
	}
	members = make([]string, 0, len(selected))
	for member := range selected {
		members = append(members, member)
	}
	sort.Strings(members)
	canonical, err := json.Marshal(members)
	if err != nil {
		return nil, "", err
	}
	sum := sha256.Sum256(canonical)
	return members, hex.EncodeToString(sum[:]), nil
}

func (s *Previewer) producePack(ctx types.TaskCtx, entry types.IEntry, members []string, out artifact.Writer) error {
	selected, err := selectedMemberSet(members)
	if err != nil {
		return err
	}
	srcCtx := task.NewTaskCtxWrapper(ctx, false, false)
	opened, format, err := s.openSourceAndFormat(srcCtx, entry)
	if err != nil {
		return err
	}
	defer func() { _ = opened.Close() }()

	archiveFS := s.newArchiveFS(ctx, opened, format)
	items, err := s.collectExtractItems(archiveFS, selected)
	if err != nil {
		return err
	}
	if err := out.WriteMeta(artifact.Meta{
		Name:     packDownloadName(entry.Name()),
		MimeType: "application/zip",
		ModTime:  time.UnixMilli(entry.ModTime()),
	}); err != nil {
		return err
	}
	var total int64
	for _, item := range items {
		if !item.isDir && item.size > 0 {
			total += item.size
		}
	}
	if total > 0 {
		ctx.Total(total, true)
	}
	copyCtx := task.NewTaskCtxWrapper(ctx, true, false)
	zw := zip.NewWriter(out)
	for _, item := range items {
		if err := ctx.Err(); err != nil {
			_ = zw.Close()
			return err
		}
		name := item.path
		if item.isDir {
			name += "/"
		}
		writer, err := zw.Create(name)
		if err != nil {
			_ = zw.Close()
			return err
		}
		if item.isDir {
			continue
		}
		file, err := archiveFS.Open(item.path)
		if err != nil {
			_ = zw.Close()
			return invalidArchiveError(err)
		}
		_, err = driveutil.Copy(copyCtx, writer, file)
		_ = file.Close()
		if err != nil {
			_ = zw.Close()
			return err
		}
	}
	return zw.Close()
}

func packDownloadName(name string) string {
	base := strings.TrimSuffix(name, pathpkg.Ext(name))
	if base == "" {
		base = "archive"
	}
	return base + ".zip"
}
