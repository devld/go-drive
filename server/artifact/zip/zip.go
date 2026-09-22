// Package zip stores zip downloads of drive entries through the shared
// artifact store. The request path is the directory being packed. The args
// document lists selected paths already relative to that directory.
package zip

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"go-drive/common"
	"go-drive/common/driveutil"
	apierr "go-drive/common/errors"
	"go-drive/common/i18n"
	"go-drive/common/task"
	"go-drive/common/types"
	"go-drive/common/utils"
	"go-drive/server/artifact"
	"path"
	"sort"
	"strings"
	"time"
)

// HandlerName is the /artifact handler that packs selected drive entries.
const HandlerName = "zip"

// Selection is one drive zip download. MaxBytes is stamped by the server.
// Paths are already relative to the request source, the directory being packed.
type Selection struct {
	Paths    []string `json:"paths"`
	MaxBytes int64    `json:"maxBytes,omitempty"`
	MaxLabel string   `json:"maxLabel,omitempty"`
}

// EncodeArgs canonicalizes a selection into the artifact args document.
func EncodeArgs(selection Selection) (string, error) {
	parsed, err := normalize(selection)
	if err != nil {
		return "", err
	}
	raw, err := json.Marshal(parsed)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

// StampLimit rewrites args so the size limit comes from the server.
func StampLimit(args string, maxBytes int64, maxLabel string) (string, error) {
	var selection Selection
	if err := json.Unmarshal([]byte(args), &selection); err != nil {
		return "", apierr.NewBadRequestError("")
	}
	selection.MaxBytes = maxBytes
	selection.MaxLabel = maxLabel
	return EncodeArgs(selection)
}

func normalize(selection Selection) (Selection, error) {
	paths := make([]string, 0, len(selection.Paths))
	seen := map[string]struct{}{}
	for _, path := range selection.Paths {
		path = utils.CleanPath(path)
		if path == "" {
			continue
		}
		if _, ok := seen[path]; ok {
			continue
		}
		seen[path] = struct{}{}
		paths = append(paths, path)
	}
	if len(paths) == 0 {
		return Selection{}, apierr.NewBadRequestError("")
	}
	sort.Strings(paths)
	selection.Paths = paths
	return selection, nil
}

func parseArgs(args string) (Selection, error) {
	var selection Selection
	if err := json.Unmarshal([]byte(args), &selection); err != nil {
		return Selection{}, apierr.NewBadRequestError("")
	}
	return normalize(selection)
}

func selectionFrom(request artifact.Request) (Selection, string, error) {
	if request.Source == nil || !request.Source.Type().IsDir() {
		return Selection{}, "", apierr.NewBadRequestError("")
	}
	selection, err := parseArgs(request.Args)
	if err != nil {
		return Selection{}, "", err
	}
	return selection, utils.CleanPath(request.Source.Path()), nil
}

type Handler struct {
	ttl time.Duration
}

func init() {
	artifact.RegisterHandler(HandlerName, NewHandler)
}

func NewHandler(ctx artifact.HandlerContext) (artifact.Handler, error) {
	ttl := utils.PositiveOr(ctx.Config.Archive.PackTTL, common.DefaultArchivePackTTL)
	return &Handler{ttl: ttl}, nil
}

func (h *Handler) Spec() artifact.Spec {
	return artifact.Spec{
		Caches: []artifact.CacheSpec{{Policy: artifact.Policy{TTL: h.ttl}}},
	}
}

func (h *Handler) Resolve(request artifact.Request) (artifact.ResolvedRequest, error) {
	selection, _, err := selectionFrom(request)
	if err != nil {
		return artifact.ResolvedRequest{}, err
	}
	sum := sha256.Sum256([]byte(strings.Join(selection.Paths, "\n")))
	return artifact.ResolvedRequest{
		Key:         hex.EncodeToString(sum[:]),
		Fingerprint: "zip-v1",
	}, nil
}

func (h *Handler) Produce(ctx types.TaskCtx, request artifact.Request, out artifact.Writer) error {
	selection, prefix, parseErr := selectionFrom(request)
	if parseErr != nil {
		return parseErr
	}
	drive := request.Source.Drive()
	if drive == nil {
		return apierr.NewNotFoundMessageError("zip source has no drive")
	}
	scratch := task.NewTaskContext(ctx)
	entries := make([]driveutil.EntryTreeNode, 0, len(selection.Paths))
	for _, member := range selection.Paths {
		if err := ctx.Err(); err != nil {
			return err
		}
		entry, e := drive.Get(ctx, path.Join(prefix, member))
		if e != nil {
			return e
		}
		node, e := driveutil.BuildEntriesTree(scratch, entry, true)
		if e != nil {
			return e
		}
		entries = append(entries, node)
	}
	total := scratch.GetTotal()
	if selection.MaxBytes > 0 && total > selection.MaxBytes {
		return apierr.NewNotAllowedMessageError(i18n.T("api.zip.size_exceed", selection.MaxLabel))
	}
	if err := out.WriteMeta(artifact.Meta{
		Name:     fmt.Sprintf("packaged_%d.zip", len(selection.Paths)),
		MimeType: "application/zip",
		ModTime:  time.Now(),
	}); err != nil {
		return err
	}
	if total > 0 {
		ctx.Total(total, true)
	}
	copyCtx := task.NewTaskCtxWrapper(ctx, true, false)
	zw := zip.NewWriter(out)
	for _, node := range entries {
		if e := driveutil.VisitEntriesTree(node, func(entry types.IEntry) error {
			if err := ctx.Err(); err != nil {
				return err
			}
			name := entry.Path()
			if entry.Type().IsDir() {
				name += "/"
			}
			if prefix != "" && strings.HasPrefix(name, prefix+"/") {
				name = strings.TrimPrefix(name, prefix+"/")
			}
			file, e := zw.Create(name)
			if e != nil {
				return e
			}
			if entry.Type().IsFile() {
				return driveutil.CopyIContent(copyCtx, entry, file)
			}
			return nil
		}); e != nil {
			_ = zw.Close()
			return e
		}
	}
	return zw.Close()
}
