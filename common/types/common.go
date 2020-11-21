package types

import "context"

type M map[string]interface{}
type SM map[string]string

type TaskCtx interface {
	context.Context
	Progress(loaded int64, abs bool)
	Total(total int64, abs bool)
	Canceled() bool
}

type IDisposable interface {
	Dispose() error
}

type IStatistics interface {
	// Status returns the name, status of this component
	Status() (string, SM, error)
}
type FormItemOption struct {
	Name     string `json:"name"`
	Title    string `json:"title"`
	Value    string `json:"value"`
	Disabled bool   `json:"disabled"`
}

type FormItem struct {
	Label       string           `json:"label"`
	Type        string           `json:"type"`
	Field       string           `json:"field"`
	Required    bool             `json:"required"`
	Description string           `json:"description"`
	Disabled    bool             `json:"disabled"`
	Options     []FormItemOption `json:"options"`
}
