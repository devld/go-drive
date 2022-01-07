package task

import "regexp"

type Option = func(*Task)

var groupRegexp = regexp.MustCompile("^([A-z0-9-_]+)(/([A-z0-9-_]+))*$")

func IsValidGroup(group string) bool {
	return groupRegexp.MatchString(group)
}

// WithName sets the description of the task.
func WithName(name string) Option {
	return func(t *Task) {
		t.Name = name
	}
}

// WithGroup sets the group of the task.
// group is a string in the format "group/subgroup/subsubgroup"
func WithGroup(group string) Option {
	if !IsValidGroup(group) {
		panic("invalid group name")
	}
	return func(t *Task) {
		t.Group = group
	}
}

// WithNameGroup sets the name and group of the task.
// see WithName and WithGroup
func WithNameGroup(name, group string) Option {
	if !IsValidGroup(group) {
		panic("invalid group name")
	}
	return func(t *Task) {
		t.Name = name
		t.Group = group
	}
}
