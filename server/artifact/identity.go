package artifact

import (
	"fmt"
	"go-drive/common/driveutil"
	"go-drive/common/types"
)

// resolvedRequest is a handler Resolve result plus the store identity Service
// derives from the source entry. Handler fields are copied, not rewritten.
type resolvedRequest struct {
	ResolvedRequest
	fullKey         string
	fullFingerprint string
}

func bindSource(entry types.IEntry, handler ResolvedRequest) resolvedRequest {
	key, fingerprint := sourceIdentity(entry)
	return resolvedRequest{
		ResolvedRequest: handler,
		fullKey:         joinIdentity(key, handler.Key),
		fullFingerprint: joinIdentity(fingerprint, handler.Fingerprint),
	}
}

func sourceIdentity(entry types.IEntry) (key, fingerprint string) {
	key = entry.Path()
	if dispatcher, ok := driveutil.IEntryAs[types.IDispatcherEntry](entry); ok {
		key = dispatcher.GetRealPath()
	}
	fingerprint = fmt.Sprintf("path=%s|type=%v|size=%d|mod=%d",
		entry.Path(), entry.Type(), entry.Size(), entry.ModTime())
	return key, fingerprint
}

func joinIdentity(base, extra string) string {
	switch {
	case extra == "":
		return base
	case base == "":
		return extra
	default:
		return base + "|" + extra
	}
}
