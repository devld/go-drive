package job

import (
	"context"
	"errors"
	"fmt"
	"go-drive/common/i18n"
	"go-drive/common/registry"
	"go-drive/common/types"
	"log"
)

// executes after all other jobs registered
func init() {
	t := i18n.TPrefix("jobs.flow.")

	commonFormItem := []types.FormItem{
		{Field: "_ignoreErr", Label: t("ignore_err"), Description: t("ignore_err_desc"), Type: "checkbox"},
	}

	actionDefs := GetActionDefs()
	flowForms := make([]types.FormItemForm, 0, len(actionDefs))
	for _, def := range actionDefs {
		form := make([]types.FormItem, len(def.ParamsForm)+len(commonFormItem))
		copy(form, def.ParamsForm)
		copy(form[len(def.ParamsForm):], commonFormItem)

		flowForms = append(flowForms, types.FormItemForm{
			Key:  def.Name,
			Name: def.DisplayName,
			Form: form,
		})
	}

	RegisterActionDef(JobActionDef{
		Name:        "flow",
		DisplayName: t("name"),
		Description: t("desc"),
		ParamsForm: []types.FormItem{
			{Field: "ops", Type: "form", Forms: &types.FormItemForms{
				AddText: t("add_text"),
				Forms:   flowForms,
			}, Required: true},
		},
		Do: func(ctx context.Context, params types.SM, ch *registry.ComponentsHolder, logFn func(string)) error {
			ops := params.GetMapList("ops")
			if len(ops) == 0 {
				return errors.New("empty ops")
			}
			for i, op := range ops {
				actionKey := op["$key"]
				ignoreError := op.GetBool("_ignoreErr")
				actionDef := GetActionDef(actionKey)
				delete(op, "$key")
				delete(op, "_ignoreErr")
				e := actionDef.Do(ctx, op, ch, logFn)
				if e != nil && !ignoreError {
					return fmt.Errorf("flow execution error at step %d: %s", i+1, e.Error())
				}
				if e != nil {
					log.Printf("ignored error at step %d: %s", i+1, e.Error())
				}
			}
			return nil
		},
	})
}
