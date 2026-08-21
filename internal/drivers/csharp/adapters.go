package csharp

import (
	"bytes"
	"fmt"
	"text/template"

	"github.com/orieken/assay/internal/domain"
)

// ── registry ──────────────────────────────────────────────────────────────────

var registry = buildRegistry()

func buildRegistry() *domain.AdapterRegistry {
	r := domain.NewAdapterRegistry()
	r.SetDefault(&xunitMoqAdapter{})
	r.Register(&nunitMoqAdapter{})
	r.Register(&mstestMoqAdapter{})
	r.Register(&xunitNSubstituteAdapter{})
	r.Register(&nunitNSubstituteAdapter{})
	return r
}

func selectAdapter(ctx *domain.ProjectContext) domain.TestAdapter {
	return registry.SelectFromContext(ctx)
}

// ── shared render data ────────────────────────────────────────────────────────

type renderData struct {
	Namespace string
	Members   []domain.PublicMember
	LLMBodies map[string][]string
}

func (d renderData) PublicMethods(m domain.PublicMember) []domain.MethodInfo {
	var out []domain.MethodInfo
	for _, method := range m.Methods {
		if method.IsPublic {
			out = append(out, method)
		}
	}
	return out
}

func (d renderData) BodyFor(name string) []string {
	if d.LLMBodies == nil {
		return nil
	}
	return d.LLMBodies[name]
}

func newRenderData(analysis *domain.SourceAnalysis, opts domain.GenerateOpts) renderData {
	return renderData{
		Namespace: csNamespace(analysis.ModulePath),
		Members:   analysis.PublicAPI,
		LLMBodies: opts.LLMBodies,
	}
}

func renderTmpl(tmpl *template.Template, data renderData) (string, error) {
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("render csharp test: %w", err)
	}
	return normaliseBlankLines(buf.String()), nil
}

var sharedFuncs = template.FuncMap{
	"indent": indentLines,
}

// ── 1. xUnit + Moq (default) ─────────────────────────────────────────────────

type xunitMoqAdapter struct{}

func (a *xunitMoqAdapter) Framework() string   { return "xunit" }
func (a *xunitMoqAdapter) MockLibrary() string { return "moq" }
func (a *xunitMoqAdapter) FrameworkConfig() domain.TestFrameworkConfig {
	return domain.TestFrameworkConfig{
		Name:           "xunit",
		TestFileSuffix: "Tests.cs",
		TestFuncPrefix: "[Fact]",
	}
}

func (a *xunitMoqAdapter) GenerateTestFile(analysis *domain.SourceAnalysis, opts domain.GenerateOpts) (string, error) {
	return renderTmpl(xunitMoqTmpl, newRenderData(analysis, opts))
}

var xunitMoqTmpl = template.Must(template.New("xunit-moq").Funcs(sharedFuncs).Parse(`using Xunit;
using Moq;
using System;

namespace {{ .Namespace }}.Tests;

{{ range .Members }}
{{ if eq .Kind "class" -}}
public class {{ .Name }}Tests
{
    // private readonly Mock<IDependency> _mockDep = new();
    private readonly {{ .Name }} _sut;

    public {{ .Name }}Tests()
    {
        // _sut = new {{ .Name }}(_mockDep.Object);
        _sut = default!; // TODO: initialise
    }
{{ range ($.PublicMethods .) }}
{{ $body := $.BodyFor .Name }}
    [Fact]
    public void {{ .Name }}_ShouldWork()
    {
{{ if $body }}
{{ indent $body 2 }}
{{ else }}
        // Arrange
        // TODO: arrange

        // Act
        // TODO: act

        // Assert
        Assert.NotNull(_sut);
{{ end }}
    }
{{ end }}
}
{{ end -}}
{{ end }}`))

// ── 2. NUnit + Moq ────────────────────────────────────────────────────────────

type nunitMoqAdapter struct{}

func (a *nunitMoqAdapter) Framework() string   { return "nunit" }
func (a *nunitMoqAdapter) MockLibrary() string { return "moq" }
func (a *nunitMoqAdapter) FrameworkConfig() domain.TestFrameworkConfig {
	return domain.TestFrameworkConfig{
		Name:           "nunit",
		TestFileSuffix: "Tests.cs",
		TestFuncPrefix: "[Test]",
	}
}

func (a *nunitMoqAdapter) GenerateTestFile(analysis *domain.SourceAnalysis, opts domain.GenerateOpts) (string, error) {
	return renderTmpl(nunitMoqTmpl, newRenderData(analysis, opts))
}

var nunitMoqTmpl = template.Must(template.New("nunit-moq").Funcs(sharedFuncs).Parse(`using NUnit.Framework;
using Moq;
using System;

namespace {{ .Namespace }}.Tests;

{{ range .Members }}
{{ if eq .Kind "class" -}}
[TestFixture]
public class {{ .Name }}Tests
{
    // private Mock<IDependency> _mockDep;
    private {{ .Name }} _sut;

    [SetUp]
    public void SetUp()
    {
        // _mockDep = new Mock<IDependency>();
        // _sut = new {{ .Name }}(_mockDep.Object);
        _sut = default!; // TODO: initialise
    }
{{ range ($.PublicMethods .) }}
{{ $body := $.BodyFor .Name }}
    [Test]
    public void {{ .Name }}_ShouldWork()
    {
{{ if $body }}
{{ indent $body 2 }}
{{ else }}
        // Arrange
        // TODO: arrange

        // Act
        // TODO: act

        // Assert
        Assert.That(_sut, Is.Not.Null);
{{ end }}
    }
{{ end }}
}
{{ end -}}
{{ end }}`))

// ── 3. MSTest + Moq ───────────────────────────────────────────────────────────

type mstestMoqAdapter struct{}

func (a *mstestMoqAdapter) Framework() string   { return "mstest" }
func (a *mstestMoqAdapter) MockLibrary() string { return "moq" }
func (a *mstestMoqAdapter) FrameworkConfig() domain.TestFrameworkConfig {
	return domain.TestFrameworkConfig{
		Name:           "mstest",
		TestFileSuffix: "Tests.cs",
		TestFuncPrefix: "[TestMethod]",
	}
}

func (a *mstestMoqAdapter) GenerateTestFile(analysis *domain.SourceAnalysis, opts domain.GenerateOpts) (string, error) {
	return renderTmpl(mstestMoqTmpl, newRenderData(analysis, opts))
}

var mstestMoqTmpl = template.Must(template.New("mstest-moq").Funcs(sharedFuncs).Parse(`using Microsoft.VisualStudio.TestTools.UnitTesting;
using Moq;
using System;

namespace {{ .Namespace }}.Tests;

{{ range .Members }}
{{ if eq .Kind "class" -}}
[TestClass]
public class {{ .Name }}Tests
{
    // private Mock<IDependency> _mockDep;
    private {{ .Name }} _sut;

    [TestInitialize]
    public void TestInitialize()
    {
        // _mockDep = new Mock<IDependency>();
        // _sut = new {{ .Name }}(_mockDep.Object);
        _sut = default!; // TODO: initialise
    }
{{ range ($.PublicMethods .) }}
{{ $body := $.BodyFor .Name }}
    [TestMethod]
    public void {{ .Name }}_ShouldWork()
    {
{{ if $body }}
{{ indent $body 2 }}
{{ else }}
        // Arrange
        // TODO: arrange

        // Act
        // TODO: act

        // Assert
        Assert.IsNotNull(_sut);
{{ end }}
    }
{{ end }}
}
{{ end -}}
{{ end }}`))

// ── 4. xUnit + NSubstitute ────────────────────────────────────────────────────

type xunitNSubstituteAdapter struct{}

func (a *xunitNSubstituteAdapter) Framework() string   { return "xunit" }
func (a *xunitNSubstituteAdapter) MockLibrary() string { return "nsubstitute" }
func (a *xunitNSubstituteAdapter) FrameworkConfig() domain.TestFrameworkConfig {
	return domain.TestFrameworkConfig{
		Name:           "xunit",
		TestFileSuffix: "Tests.cs",
		TestFuncPrefix: "[Fact]",
	}
}

func (a *xunitNSubstituteAdapter) GenerateTestFile(analysis *domain.SourceAnalysis, opts domain.GenerateOpts) (string, error) {
	return renderTmpl(xunitNSubTmpl, newRenderData(analysis, opts))
}

var xunitNSubTmpl = template.Must(template.New("xunit-nsubstitute").Funcs(sharedFuncs).Parse(`using Xunit;
using NSubstitute;
using System;

namespace {{ .Namespace }}.Tests;

{{ range .Members }}
{{ if eq .Kind "class" -}}
public class {{ .Name }}Tests
{
    // private readonly IDependency _mockDep = Substitute.For<IDependency>();
    private readonly {{ .Name }} _sut;

    public {{ .Name }}Tests()
    {
        // _sut = new {{ .Name }}(_mockDep);
        _sut = default!; // TODO: initialise
    }
{{ range ($.PublicMethods .) }}
{{ $body := $.BodyFor .Name }}
    [Fact]
    public void {{ .Name }}_ShouldWork()
    {
{{ if $body }}
{{ indent $body 2 }}
{{ else }}
        // Arrange
        // TODO: arrange

        // Act
        // TODO: act

        // Assert
        Assert.NotNull(_sut);
{{ end }}
    }
{{ end }}
}
{{ end -}}
{{ end }}`))

// ── 5. NUnit + NSubstitute ────────────────────────────────────────────────────

type nunitNSubstituteAdapter struct{}

func (a *nunitNSubstituteAdapter) Framework() string   { return "nunit" }
func (a *nunitNSubstituteAdapter) MockLibrary() string { return "nsubstitute" }
func (a *nunitNSubstituteAdapter) FrameworkConfig() domain.TestFrameworkConfig {
	return domain.TestFrameworkConfig{
		Name:           "nunit",
		TestFileSuffix: "Tests.cs",
		TestFuncPrefix: "[Test]",
	}
}

func (a *nunitNSubstituteAdapter) GenerateTestFile(analysis *domain.SourceAnalysis, opts domain.GenerateOpts) (string, error) {
	return renderTmpl(nunitNSubTmpl, newRenderData(analysis, opts))
}

var nunitNSubTmpl = template.Must(template.New("nunit-nsubstitute").Funcs(sharedFuncs).Parse(`using NUnit.Framework;
using NSubstitute;
using System;

namespace {{ .Namespace }}.Tests;

{{ range .Members }}
{{ if eq .Kind "class" -}}
[TestFixture]
public class {{ .Name }}Tests
{
    // private IDependency _mockDep;
    private {{ .Name }} _sut;

    [SetUp]
    public void SetUp()
    {
        // _mockDep = Substitute.For<IDependency>();
        // _sut = new {{ .Name }}(_mockDep);
        _sut = default!; // TODO: initialise
    }
{{ range ($.PublicMethods .) }}
{{ $body := $.BodyFor .Name }}
    [Test]
    public void {{ .Name }}_ShouldWork()
    {
{{ if $body }}
{{ indent $body 2 }}
{{ else }}
        // Arrange
        // TODO: arrange

        // Act
        // TODO: act

        // Assert
        Assert.That(_sut, Is.Not.Null);
{{ end }}
    }
{{ end }}
}
{{ end -}}
{{ end }}`))

// csNamespace is defined in generator.go and available to all files in this package.

// ── LLMVocabulary ─────────────────────────────────────────────────────────────

func (a *xunitMoqAdapter) LLMVocabulary() map[string]string {
	return map[string]string{
		"framework":      "xUnit",
		"mock_library":   "Moq",
		"assert_style":   "Assert.Equal(expected, actual)",
		"mock_style":     "var mock = new Mock<IFoo>(); mock.Setup(x => x.Method()).Returns(value)",
		"test_attribute": "[Fact]",
	}
}

func (a *nunitMoqAdapter) LLMVocabulary() map[string]string {
	return map[string]string{
		"framework":      "NUnit",
		"mock_library":   "Moq",
		"assert_style":   "Assert.AreEqual(expected, actual)",
		"mock_style":     "var mock = new Mock<IFoo>(); mock.Setup(x => x.Method()).Returns(value)",
		"test_attribute": "[Test]",
	}
}

func (a *mstestMoqAdapter) LLMVocabulary() map[string]string {
	return map[string]string{
		"framework":      "MSTest",
		"mock_library":   "Moq",
		"assert_style":   "Assert.AreEqual(expected, actual)",
		"mock_style":     "var mock = new Mock<IFoo>(); mock.Setup(x => x.Method()).Returns(value)",
		"test_attribute": "[TestMethod]",
	}
}

func (a *xunitNSubstituteAdapter) LLMVocabulary() map[string]string {
	return map[string]string{
		"framework":      "xUnit",
		"mock_library":   "NSubstitute",
		"assert_style":   "Assert.Equal(expected, actual)",
		"mock_style":     "var sub = Substitute.For<IFoo>(); sub.Method().Returns(value)",
		"test_attribute": "[Fact]",
	}
}

func (a *nunitNSubstituteAdapter) LLMVocabulary() map[string]string {
	return map[string]string{
		"framework":      "NUnit",
		"mock_library":   "NSubstitute",
		"assert_style":   "Assert.AreEqual(expected, actual)",
		"mock_style":     "var sub = Substitute.For<IFoo>(); sub.Method().Returns(value)",
		"test_attribute": "[Test]",
	}
}
