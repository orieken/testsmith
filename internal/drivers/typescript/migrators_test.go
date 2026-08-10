package typescript_test

import (
	"strings"
	"testing"

	"github.com/orieken/assay/internal/drivers/typescript"
)

func newDriver() *typescript.Driver {
	return &typescript.Driver{}
}

func TestJestToVitest_ReplacesJestGlobals(t *testing.T) {
	driver := newDriver()
	ms := driver.ListMigrators()

	var m interface{ MigrateFile(string) (string, error) }
	for _, mg := range ms {
		if mg.From() == "jest" && mg.To() == "vitest" {
			m = mg
		}
	}
	if m == nil {
		t.Fatal("jest→vitest migrator not found")
	}

	input := `jest.fn()
jest.mock('./foo')
jest.spyOn(obj, 'method')
jest.clearAllMocks()
jest.useFakeTimers()
jest.advanceTimersByTime(1000)
`
	got, err := m.MigrateFile(input)
	if err != nil {
		t.Fatalf("MigrateFile: %v", err)
	}

	checks := []string{"vi.fn()", "vi.mock(", "vi.spyOn(", "vi.clearAllMocks()", "vi.useFakeTimers()", "vi.advanceTimersByTime("}
	for _, c := range checks {
		if !strings.Contains(got, c) {
			t.Errorf("expected %q in output; got:\n%s", c, got)
		}
	}
	if strings.Contains(got, "jest.") {
		t.Errorf("output still contains jest. references:\n%s", got)
	}
}

func TestJestToVitest_RemovesJestGlobalsImport(t *testing.T) {
	driver := newDriver()
	var m interface{ MigrateFile(string) (string, error) }
	for _, mg := range driver.ListMigrators() {
		if mg.From() == "jest" {
			m = mg
		}
	}

	input := `import { describe, it, expect } from '@jest/globals';

describe('foo', () => { it('works', () => { expect(1).toBe(1); }); });
`
	got, _ := m.MigrateFile(input)
	if strings.Contains(got, "@jest/globals") {
		t.Errorf("jest/globals import should be removed:\n%s", got)
	}
	if !strings.Contains(got, "vitest") {
		t.Errorf("vitest import should be injected:\n%s", got)
	}
}

func TestVitestToJest_ReplacesViGlobals(t *testing.T) {
	driver := newDriver()
	var m interface{ MigrateFile(string) (string, error) }
	for _, mg := range driver.ListMigrators() {
		if mg.From() == "vitest" && mg.To() == "jest" {
			m = mg
		}
	}
	if m == nil {
		t.Fatal("vitest→jest migrator not found")
	}

	input := `vi.fn()
vi.mock('./foo')
vi.spyOn(obj, 'method')
vi.clearAllMocks()
`
	got, _ := m.MigrateFile(input)
	for _, c := range []string{"jest.fn()", "jest.mock(", "jest.spyOn(", "jest.clearAllMocks()"} {
		if !strings.Contains(got, c) {
			t.Errorf("expected %q:\n%s", c, got)
		}
	}
}
