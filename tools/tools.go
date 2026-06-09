//go:build tools

// Package tools tracks build-time tool dependencies so go mod tidy keeps them.
package tools

import (
	_ "github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs"
)
