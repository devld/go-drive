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

// maxSizeOption is the operator setting for the largest zip this handler packs.
const maxSizeOption = "zip.maxSize"

func (h *Handler) selectionFrom(request artifact.Request) ([]string, string, error) {
	if !request.Source.Type().IsDir() {
		return nil, "", apierr.NewBadRequestError("")
	}
	paths, err := parseArgs(request.Args)
	if err != nil {
		return nil, "", err
	}
	return paths, utils.CleanPath(request.Source.Path()), nil
}

func (h *Handler) sizeLimit() (int64, string) {
	if h.options == nil {
		return -1, ""
	}
	value := h.options.GetValue(maxSizeOption)
	return value.DataSize(-1), string(value)
}

type Handler struct {
	ttl     time.Duration
	options artifact.OptionReader
}

func init() {
	artifact.RegisterHandler(HandlerName, NewHandler)
}

func NewHandler(ctx artifact.HandlerContext) (artifact.Handler, error) {
	ttl := utils.PositiveOr(ctx.Config.Archive.PackTTL, common.DefaultArchivePackTTL)
	return &Handler{ttl: ttl, options: ctx.Options}, nil
}

func (h *Handler) Spec() artifact.Spec {
	return artifact.Spec{
		Caches: []artifact.CacheSpec{{Policy: artifact.Policy{TTL: h.ttl}}},
	}
}

func (h *Handler) Resolve(request artifact.Request) (artifact.ResolvedRequest, error) {
	paths, _, err := h.selectionFrom(request)
	if err != nil {
		return artifact.ResolvedRequest{}, err
	}
	sum := sha256.Sum256([]byte(strings.Join(paths, "\n")))
	return artifact.ResolvedRequest{
		Key:         hex.EncodeToString(sum[:]),
		Fingerprint: "zip-v1",
	}, nil
}

func (h *Handler) Produce(ctx types.TaskCtx, request artifact.Request, out artifact.Writer) error {
	paths, prefix, parseErr := h.selectionFrom(request)
	if parseErr != nil {
		return parseErr
	}
	drive := request.Source.Drive()
	if drive == nil {
		return apierr.NewNotFoundMessageError("zip source has no drive")
	}
	entries := make([]driveutil.EntryTreeNode, 0, len(paths))
	maxBytes, maxLabel := h.sizeLimit()
	for _, member := range paths {
		if err := ctx.Err(); err != nil {
			return err
		}
		entry, e := drive.Get(ctx, path.Join(prefix, member))
		if e != nil {
			return e
		}
		node, e := driveutil.BuildEntriesTree(ctx, entry, true)
		if e != nil {
			return e
		}
		if maxBytes > 0 && ctx.GetTotal() > maxBytes {
			return apierr.NewNotAllowedMessageError(i18n.T("api.zip.size_exceed", maxLabel))
		}
		entries = append(entries, node)
	}
	if err := out.WriteMeta(artifact.Meta{
		Name:     fmt.Sprintf("packaged_%d.zip", len(paths)),
		MimeType: "application/zip",
		ModTime:  time.Now(),
	}); err != nil {
		return err
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

func normalize(paths []string) ([]string, error) {
	cleaned := make([]string, 0, len(paths))
	seen := map[string]struct{}{}
	for _, path := range paths {
		path = utils.CleanPath(path)
		if path == "" {
			continue
		}
		if _, ok := seen[path]; ok {
			continue
		}
		seen[path] = struct{}{}
		cleaned = append(cleaned, path)
	}
	if len(cleaned) == 0 {
		return nil, apierr.NewBadRequestError("")
	}
	sort.Strings(cleaned)
	return cleaned, nil
}

func parseArgs(args string) ([]string, error) {
	var paths []string
	if err := json.Unmarshal([]byte(args), &paths); err != nil {
		return nil, apierr.NewBadRequestError("")
	}
	return normalize(paths)
}
