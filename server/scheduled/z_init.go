package scheduled

import (
	"context"
	"errors"
	"fmt"
	"go-drive/common/i18n"
	"go-drive/common/registry"
	"go-drive/common/types"
)

// executes after all other jobs registered
func init() {
	t := i18n.TPrefix("jobs.flow.")

	commonFormItem := []types.FormItem{
		{Field: "_ignoreErr", Label: t("ignore_err"), Description: t("ignore_err_desc"), Type: "checkbox"},
	}

	jobs := GetJobs()
	flowForms := make([]types.FormItemForm, 0, len(jobs))
	for _, job := range jobs {
		form := make([]types.FormItem, len(job.ParamsForm)+len(commonFormItem))
		copy(form, job.ParamsForm)
		copy(form[len(job.ParamsForm):], commonFormItem)

		flowForms = append(flowForms, types.FormItemForm{
			Key:  job.Name,
			Name: job.DisplayName,
			Form: form,
		})
	}

	RegisterJob(JobDefinition{
		Name:        "flow",
		DisplayName: t("name"),
		Description: t("desc"),
		ParamsForm: []types.FormItem{
			{Field: "ops", Type: "form", Forms: &types.FormItemForms{
				AddText: t("add_text"),
				Forms:   flowForms,
			}, Required: true},
		},
		Do: func(ctx context.Context, params types.SM, ch *registry.ComponentsHolder, log func(string)) error {
			ops := params.GetMapList("ops")
			if len(ops) == 0 {
				return errors.New("empty ops")
			}
			for i, op := range ops {
				jobKey := op["$key"]
				ignoreError := op.GetBool("_ignoreErr")
				job := GetJob(jobKey)
				delete(op, "$key")
				delete(op, "_ignoreErr")
				e := job.Do(ctx, op, ch, log)
				if e != nil && !ignoreError {
					return fmt.Errorf("flow execution error at step %d: %s", i+1, e.Error())
				}
				if e != nil {
					fmt.Printf("ignored error at step %d: %s", i+1, e.Error())
				}
			}
			return nil
		},
	})
}
