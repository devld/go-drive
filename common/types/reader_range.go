package types

import "fmt"

// ReaderRange selects bytes of an entry for IContentReader.GetReader.
// A negative Start or Size, or both zero, is a full-content request.
// Callers that want the entire content pass FullReaderRange (-1, -1).
type ReaderRange struct {
	Start int64
	Size  int64
}

// FullReaderRange requests the entire content.
func FullReaderRange() ReaderRange {
	return ReaderRange{Start: -1, Size: -1}
}

// IsFullRequest reports whether r requests the entire content.
// A negative Start, a negative Size, or both zero selects the full content.
func (r ReaderRange) IsFullRequest() bool {
	return r.Start < 0 || r.Size < 0 || (r.Start == 0 && r.Size == 0)
}

// BuildHTTPRangeHeader returns the HTTP Range header value for r,
// such as "bytes=10-14" or "bytes=10-". It returns an empty string for
// a full-content request.
func (r ReaderRange) BuildHTTPRangeHeader() string {
	if r.IsFullRequest() {
		return ""
	}
	header := fmt.Sprintf("bytes=%d-", r.Start)
	if r.Size > 0 {
		header += fmt.Sprintf("%d", r.Start+r.Size-1)
	}
	return header
}
