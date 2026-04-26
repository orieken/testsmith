package main

import (
	"github.com/orieken/testsmith/internal/drivers/csharp"
	"github.com/orieken/testsmith/internal/drivers/golang"
	"github.com/orieken/testsmith/internal/drivers/java"
	"github.com/orieken/testsmith/internal/drivers/python"
	"github.com/orieken/testsmith/internal/drivers/typescript"
	"github.com/orieken/testsmith/internal/registry"
)

// Version is injected at build time via -ldflags.
var Version = "v2.0.0-dev"

// reg is the global DriverRegistry populated at startup.
var reg *registry.Registry

func main() {
	// Composition root: register all language drivers.
	reg = registry.New()
	reg.Register(python.New())
	reg.Register(typescript.New())
	reg.Register(golang.New())
	reg.Register(java.New())
	reg.Register(csharp.New())

	execute()
}
