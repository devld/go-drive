package types

import "testing"

func TestReaderRangeIsFullRequestAndHeader(t *testing.T) {
	cases := []struct {
		name   string
		r      ReaderRange
		full   bool
		header string
	}{
		{name: "full", r: FullReaderRange(), full: true, header: ""},
		{name: "negative start", r: ReaderRange{Start: -1, Size: 0}, full: true, header: ""},
		{name: "negative size", r: ReaderRange{Start: 0, Size: -1}, full: true, header: ""},
		{name: "negative size at offset", r: ReaderRange{Start: 10, Size: -1}, full: true, header: ""},
		{name: "both zero", r: ReaderRange{}, full: true, header: ""},
		{name: "open from offset", r: ReaderRange{Start: 10, Size: 0}, full: false, header: "bytes=10-"},
		{name: "bounded", r: ReaderRange{Start: 10, Size: 5}, full: false, header: "bytes=10-14"},
		{name: "positive size without start", r: ReaderRange{Start: -1, Size: 5}, full: true, header: ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.r.IsFullRequest(); got != tc.full {
				t.Fatalf("IsFullRequest = %v, want %v", got, tc.full)
			}
			if got := tc.r.BuildHTTPRangeHeader(); got != tc.header {
				t.Fatalf("header = %q, want %q", got, tc.header)
			}
		})
	}
}
