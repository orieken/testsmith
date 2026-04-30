package python_test

import (
	"strings"
	"testing"

	"github.com/orieken/testsmith/internal/drivers/python"
)

func TestPytestMockToUnittestMock_ReplacesAPIs(t *testing.T) {
	driver := &python.Driver{}
	var m interface{ MigrateFile(string) (string, error) }
	for _, mg := range driver.ListMigrators() {
		if mg.From() == "pytest-mock" && mg.To() == "unittest.mock" {
			m = mg
		}
	}
	if m == nil {
		t.Fatal("pytest-mock→unittest.mock migrator not found")
	}

	input := `import pytest

def test_payment(mocker):
    mock_fn = mocker.patch('services.charge')
    obj = mocker.MagicMock()
    spy = mocker.patch.object(PayService, 'run')
    mock_fn.return_value = True
`
	got, err := m.MigrateFile(input)
	if err != nil {
		t.Fatalf("MigrateFile: %v", err)
	}

	checks := []string{
		"patch('services.charge')",
		"MagicMock()",
		"patch.object(PayService",
		"from unittest.mock import",
	}
	for _, c := range checks {
		if !strings.Contains(got, c) {
			t.Errorf("expected %q in output:\n%s", c, got)
		}
	}
	if strings.Contains(got, "mocker.patch(") {
		t.Errorf("mocker.patch( should be replaced:\n%s", got)
	}
	if strings.Contains(got, "mocker.MagicMock(") {
		t.Errorf("mocker.MagicMock( should be replaced:\n%s", got)
	}
}

func TestPytestMockToUnittestMock_RemovesMockerParam(t *testing.T) {
	driver := &python.Driver{}
	var m interface{ MigrateFile(string) (string, error) }
	for _, mg := range driver.ListMigrators() {
		if mg.From() == "pytest-mock" {
			m = mg
		}
	}

	cases := []struct {
		input string
		want  string
	}{
		{`def test_foo(mocker):`, `def test_foo():`},
		{`def test_foo(mocker, db):`, `def test_foo(db):`},
		{`def test_foo(db, mocker):`, `def test_foo(db):`},
	}
	for _, tc := range cases {
		got, _ := m.MigrateFile(tc.input)
		if !strings.Contains(got, tc.want) {
			t.Errorf("input %q: expected %q in output, got:\n%s", tc.input, tc.want, got)
		}
	}
}
