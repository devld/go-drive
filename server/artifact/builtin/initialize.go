// Package builtin assembles the built-in artifact processors. The parent
// artifact package remains format-agnostic; this package is the composition
// root that loads thumbnail and archive handlers.
package builtin

import (
	"errors"
	"fmt"
	"go-drive/common"
	"go-drive/common/registry"
	"go-drive/common/task"
	"go-drive/server/artifact"
	_ "go-drive/server/artifact/archive"
	_ "go-drive/server/artifact/thumbnail"
	_ "go-drive/server/artifact/zip"
)

// Initialize creates the shared artifact service, constructs every registered
// handler, and installs their artifact types.
func Initialize(config common.Config, options artifact.OptionReader, runner task.Runner, components *registry.ComponentsHolder) (*artifact.Service, error) {
	if runner == nil {
		return nil, errors.New("artifact task runner is required")
	}
	artifacts, err := artifact.NewService(config, runner, options)
	if err != nil {
		return nil, fmt.Errorf("create artifact service: %w", err)
	}
	if components != nil {
		components.Add(registry.KeyArtifact, artifacts)
	}
	return artifacts, nil
}
