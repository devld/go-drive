package artifact

import (
	"fmt"
	"go-drive/common/driveutil"
	"go-drive/common/types"
)

// bindSource prepends the source entry's stable slot and changing identity
// onto a handler Resolve result. Handlers only describe their own key
// fragment and generator settings; they do not encode entry metadata.
func bindSource(entry types.IEntry, resolved ResolvedRequest) ResolvedRequest {
	if entry == nil {
		return resolved
	}
	key, fingerprint := sourceIdentity(entry)
	resolved.Key = joinIdentity(key, resolved.Key)
	resolved.Fingerprint = joinIdentity(fingerprint, resolved.Fingerprint)
	return resolved
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
