package types

import "time"

type EntrySearchItem struct {
	Path    string    `json:"path"`
	Name    string    `json:"name"`
	Ext     string    `json:"ext"`
	Type    EntryType `json:"type"`
	Size    int64     `json:"size"`
	ModTime time.Time `json:"modTime"`
}

type EntrySearchResultItem struct {
	Entry      EntrySearchItem     `json:"entry"`
	Highlights map[string][]string `json:"highlights"`
}

type EntrySearchResult struct {
	Items []EntrySearchResultItem `json:"items"`
	Next  int                     `json:"next"`
}
