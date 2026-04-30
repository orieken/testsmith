package python

import (
	"github.com/orieken/testsmith/internal/domain"
	"github.com/orieken/testsmith/internal/migration"
)

var pyMigrators = []domain.Migrator{
	pytestMockToUnittestMock(),
}

func pytestMockToUnittestMock() *migration.TextMigrator {
	return migration.New("pytest-mock", "unittest.mock").
		// Finer-grained patch variants first.
		Add(`mocker\.patch\.object\(`, `patch.object(`).
		Add(`mocker\.patch\.multiple\(`, `patch.multiple(`).
		Add(`mocker\.patch\(`, `patch(`).
		// Mock constructors.
		Add(`mocker\.MagicMock\(`, `MagicMock(`).
		Add(`mocker\.Mock\(`, `Mock(`).
		Add(`mocker\.AsyncMock\(`, `AsyncMock(`).
		Add(`mocker\.NonCallableMock\(`, `NonCallableMock(`).
		// Other helpers.
		Add(`mocker\.sentinel\b`, `sentinel`).
		Add(`mocker\.call\b`, `call`).
		Add(`mocker\.ANY\b`, `ANY`).
		// Strip `mocker` from test function parameter lists (common patterns).
		Add(`(?m)(def test_\w+\()mocker,\s*`, `$1`).
		Add(`(?m)(def test_\w+\([^)]+),\s*mocker(\))`, `$1$2`).
		Add(`(?m)(def test_\w+\([^)]+),\s*mocker,\s*`, `$1, `).
		Add(`(?m)(def test_\w+\()mocker(\))`, `$1$2`).
		// Inject unittest.mock import.
		InjectImport(`from unittest.mock import patch, MagicMock, Mock, AsyncMock, NonCallableMock, call, sentinel, ANY`)
}
