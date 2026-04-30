package typescript

import (
	"github.com/orieken/testsmith/internal/domain"
	"github.com/orieken/testsmith/internal/migration"
)

var tsMigrators = []domain.Migrator{
	jestToVitest(),
	vitestToJest(),
}

func jestToVitest() *migration.TextMigrator {
	return migration.New("jest", "vitest").
		// Remove explicit jest-globals import.
		Add(`(?m)^import\s*\{[^}]*\}\s*from\s*['"]@jest/globals['"]\s*;?\n?`, "").
		// Replace jest.X → vi.X for all common APIs.
		Add(`\bjest\.(fn|mock|spyOn|clearAllMocks|resetAllMocks|restoreAllMocks|clearAllTimers|resetAllTimers|useFakeTimers|useRealTimers|runAllTimers|runOnlyPendingTimers|advanceTimersByTime|advanceTimersToNextTimer|setSystemTime|getRealSystemTime|createMockFromModule)\b`,
			`vi.$1`).
		Add(`\bjest\.mock\(`, `vi.mock(`).
		// Inject vitest import at the top.
		InjectImport(`import { describe, test, it, expect, beforeEach, afterEach, beforeAll, afterAll, vi } from 'vitest';`)
}

func vitestToJest() *migration.TextMigrator {
	return migration.New("vitest", "jest").
		// Remove vitest import.
		Add(`(?m)^import\s*\{[^}]*\}\s*from\s*['"]vitest['"]\s*;?\n?`, "").
		// Replace vi.X → jest.X.
		Add(`\bvi\.(fn|mock|spyOn|clearAllMocks|resetAllMocks|restoreAllMocks|clearAllTimers|resetAllTimers|useFakeTimers|useRealTimers|runAllTimers|runOnlyPendingTimers|advanceTimersByTime|advanceTimersToNextTimer|setSystemTime|getRealSystemTime|createMockFromModule)\b`,
			`jest.$1`).
		Add(`\bvi\.mock\(`, `jest.mock(`)
}
