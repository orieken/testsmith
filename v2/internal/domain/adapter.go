package domain

import "sort"

// TestAdapter implements test scaffold generation for one framework+mock combination.
//
// To add support for a new framework, implement this interface and register it
// with the driver's AdapterRegistry using RegisterAdapter().
//
// Minimal implementation:
//
//	type myAdapter struct{}
//
//	func (a *myAdapter) Framework() string        { return "myframework" }
//	func (a *myAdapter) MockLibrary() string      { return "mymocklib" }
//	func (a *myAdapter) FrameworkConfig() TestFrameworkConfig { return TestFrameworkConfig{...} }
//	func (a *myAdapter) GenerateTestFile(analysis *SourceAnalysis, opts GenerateOpts) (string, error) {
//	    // render your template here
//	}
type TestAdapter interface {
	// Framework returns the test framework identifier (lowercase, no spaces).
	// Examples: "pytest", "jest", "vitest", "junit5", "junit4", "testng",
	//           "springboot", "xunit", "nunit", "mstest", "testing", "testify".
	Framework() string

	// MockLibrary returns the mock/stub library identifier (lowercase, no spaces).
	// Use "interfaces" for languages where interface-based fakes are the convention (Go stdlib).
	// Examples: "pytest-mock", "unittest.mock", "mockito", "moq", "nsubstitute",
	//           "sinon", "testify", "gomock", "interfaces".
	MockLibrary() string

	// FrameworkConfig returns file-naming conventions used by the analysis pipeline
	// (test file suffix/prefix, fixture dir, bootstrap file).
	FrameworkConfig() TestFrameworkConfig

	// GenerateTestFile produces the full content of a test scaffold file.
	// Implementations must NOT perform any I/O.
	GenerateTestFile(analysis *SourceAnalysis, opts GenerateOpts) (string, error)
}

// AdapterRegistry maps (framework, mockLibrary) → TestAdapter.
// It is the primary extension point for adding new framework support.
type AdapterRegistry struct {
	adapters map[string]TestAdapter // key: "framework/mockLibrary"
	def      TestAdapter            // fallback when no match is found
}

// NewAdapterRegistry returns an empty registry.
func NewAdapterRegistry() *AdapterRegistry {
	return &AdapterRegistry{adapters: make(map[string]TestAdapter)}
}

// Register adds an adapter. The key is derived from adapter.Framework()+"/"+adapter.MockLibrary().
func (r *AdapterRegistry) Register(a TestAdapter) {
	r.adapters[adapterKey(a.Framework(), a.MockLibrary())] = a
}

// SetDefault sets the adapter returned when no (framework, mockLibrary) match is found.
func (r *AdapterRegistry) SetDefault(a TestAdapter) {
	r.def = a
	r.Register(a) // also reachable by explicit key
}

// All returns every registered adapter. The default adapter is always first;
// remaining adapters are sorted by framework+mockLibrary key for stable output.
func (r *AdapterRegistry) All() []TestAdapter {
	var out []TestAdapter
	var defKey string

	if r.def != nil {
		defKey = adapterKey(r.def.Framework(), r.def.MockLibrary())
		out = append(out, r.def)
	}

	// Collect keys for deterministic sort.
	keys := make([]string, 0, len(r.adapters))
	for k := range r.adapters {
		if k != defKey {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)

	for _, k := range keys {
		out = append(out, r.adapters[k])
	}
	return out
}

// Select returns the adapter for the given framework and mock library.
// Falls back to the default adapter when no exact match exists.
func (r *AdapterRegistry) Select(framework, mockLibrary string) TestAdapter {
	if a, ok := r.adapters[adapterKey(framework, mockLibrary)]; ok {
		return a
	}
	// Try framework-only match (first registered mock for that framework).
	if framework != "" {
		for key, a := range r.adapters {
			if a.Framework() == framework {
				_ = key
				return a
			}
		}
	}
	return r.def
}

// SelectFromContext extracts Framework and MockLibrary from ctx.Metadata and selects
// the matching adapter. This is the primary call site inside drivers.
func (r *AdapterRegistry) SelectFromContext(ctx *ProjectContext) TestAdapter {
	if ctx == nil {
		return r.def
	}
	fw, _ := ctx.Metadata["framework"].(string)
	ml, _ := ctx.Metadata["mock_library"].(string)
	return r.Select(fw, ml)
}

func adapterKey(framework, mockLibrary string) string {
	return framework + "/" + mockLibrary
}
