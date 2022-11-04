package scheduled

import (
	"context"
	"go-drive/common/registry"
	"go-drive/common/types"
)

type JobWork func(context.Context, types.SM, *registry.ComponentsHolder) error

type JobDefinition struct {
	Name        string           `json:"name"`
	DisplayName string           `json:"displayName" i18n:""`
	Description string           `json:"description" i18n:""`
	ParamsForm  []types.FormItem `json:"paramsForm"`
	Do          JobWork          `json:"-"`
}
