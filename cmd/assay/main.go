package main

import (
	"github.com/orieken/assay/internal/drivers/csharp"
	"github.com/orieken/assay/internal/drivers/golang"
	"github.com/orieken/assay/internal/drivers/java"
	"github.com/orieken/assay/internal/drivers/kotlin"
	"github.com/orieken/assay/internal/drivers/python"
	"github.com/orieken/assay/internal/drivers/rust"
	"github.com/orieken/assay/internal/drivers/typescript"
	"github.com/orieken/assay/internal/registry"
)

// Version is injected at build time via -ldflags.
var Version = "v2.0.1-dev"

// reg is the global DriverRegistry populated at startup.
var reg *registry.Registry

func main() {
	initRegistry()
	execute()
}

func initRegistry() {
	reg = registry.New()
	reg.Register(python.New())
	reg.Register(typescript.New())
	reg.Register(golang.New())
	reg.Register(java.New())
	reg.Register(csharp.New())
	reg.Register(rust.New())
	reg.Register(kotlin.New())
}
