package java

import (
	"github.com/orieken/testsmith/internal/domain"
	"github.com/orieken/testsmith/internal/migration"
)

var javaMigrators = []domain.Migrator{
	junit4ToJunit5(),
	junit5ToJunit4(),
}

func junit4ToJunit5() *migration.TextMigrator {
	return migration.New("junit4", "junit5").
		// Imports — order matters: more-specific patterns first.
		Add(`import org\.junit\.runner\.RunWith;`, `import org.junit.jupiter.api.extension.ExtendWith;`).
		Add(`import org\.mockito\.junit\.MockitoJUnitRunner;`, `import org.mockito.junit.jupiter.MockitoExtension;`).
		Add(`import org\.junit\.BeforeClass;`, `import org.junit.jupiter.api.BeforeAll;`).
		Add(`import org\.junit\.AfterClass;`, `import org.junit.jupiter.api.AfterAll;`).
		Add(`import org\.junit\.Before;`, `import org.junit.jupiter.api.BeforeEach;`).
		Add(`import org\.junit\.After;`, `import org.junit.jupiter.api.AfterEach;`).
		Add(`import org\.junit\.Ignore;`, `import org.junit.jupiter.api.Disabled;`).
		Add(`import org\.junit\.Assert;`, `import org.junit.jupiter.api.Assertions;`).
		Add(`import org\.junit\.Test;`, `import org.junit.jupiter.api.Test;`).
		// Runner annotation.
		Add(`@RunWith\(MockitoJUnitRunner\.Silent\.class\)`, `@ExtendWith(MockitoExtension.class)`).
		Add(`@RunWith\(MockitoJUnitRunner\.class\)`, `@ExtendWith(MockitoExtension.class)`).
		Add(`@RunWith\(\w+\.class\)`, `@ExtendWith($0)`). // generic fallback — leaves unknown runners
		// Lifecycle annotations.
		Add(`@BeforeClass\b`, `@BeforeAll`).
		Add(`@AfterClass\b`, `@AfterAll`).
		Add(`@Before\b`, `@BeforeEach`).
		Add(`@After\b`, `@AfterEach`).
		Add(`@Ignore\b`, `@Disabled`).
		// Assertions.
		Add(`Assert\.assertEquals\(`, `Assertions.assertEquals(`).
		Add(`Assert\.assertNotEquals\(`, `Assertions.assertNotEquals(`).
		Add(`Assert\.assertTrue\(`, `Assertions.assertTrue(`).
		Add(`Assert\.assertFalse\(`, `Assertions.assertFalse(`).
		Add(`Assert\.assertNull\(`, `Assertions.assertNull(`).
		Add(`Assert\.assertNotNull\(`, `Assertions.assertNotNull(`).
		Add(`Assert\.assertSame\(`, `Assertions.assertSame(`).
		Add(`Assert\.assertNotSame\(`, `Assertions.assertNotSame(`).
		Add(`Assert\.assertArrayEquals\(`, `Assertions.assertArrayEquals(`).
		Add(`Assert\.assertThrows\(`, `Assertions.assertThrows(`).
		Add(`Assert\.fail\(`, `Assertions.fail(`)
}

func junit5ToJunit4() *migration.TextMigrator {
	return migration.New("junit5", "junit4").
		// Imports.
		Add(`import org\.junit\.jupiter\.api\.extension\.ExtendWith;`, `import org.junit.runner.RunWith;`).
		Add(`import org\.mockito\.junit\.jupiter\.MockitoExtension;`, `import org.mockito.junit.MockitoJUnitRunner;`).
		Add(`import org\.junit\.jupiter\.api\.BeforeAll;`, `import org.junit.BeforeClass;`).
		Add(`import org\.junit\.jupiter\.api\.AfterAll;`, `import org.junit.AfterClass;`).
		Add(`import org\.junit\.jupiter\.api\.BeforeEach;`, `import org.junit.Before;`).
		Add(`import org\.junit\.jupiter\.api\.AfterEach;`, `import org.junit.After;`).
		Add(`import org\.junit\.jupiter\.api\.Disabled;`, `import org.junit.Ignore;`).
		Add(`import org\.junit\.jupiter\.api\.Assertions;`, `import org.junit.Assert;`).
		Add(`import org\.junit\.jupiter\.api\.Test;`, `import org.junit.Test;`).
		// Runner annotation.
		Add(`@ExtendWith\(MockitoExtension\.class\)`, `@RunWith(MockitoJUnitRunner.class)`).
		// Lifecycle annotations.
		Add(`@BeforeAll\b`, `@BeforeClass`).
		Add(`@AfterAll\b`, `@AfterClass`).
		Add(`@BeforeEach\b`, `@Before`).
		Add(`@AfterEach\b`, `@After`).
		Add(`@Disabled\b`, `@Ignore`).
		// Assertions.
		Add(`Assertions\.assertEquals\(`, `Assert.assertEquals(`).
		Add(`Assertions\.assertNotEquals\(`, `Assert.assertNotEquals(`).
		Add(`Assertions\.assertTrue\(`, `Assert.assertTrue(`).
		Add(`Assertions\.assertFalse\(`, `Assert.assertFalse(`).
		Add(`Assertions\.assertNull\(`, `Assert.assertNull(`).
		Add(`Assertions\.assertNotNull\(`, `Assert.assertNotNull(`).
		Add(`Assertions\.assertSame\(`, `Assert.assertSame(`).
		Add(`Assertions\.assertNotSame\(`, `Assert.assertNotSame(`).
		Add(`Assertions\.assertArrayEquals\(`, `Assert.assertArrayEquals(`).
		Add(`Assertions\.assertThrows\(`, `Assert.assertThrows(`).
		Add(`Assertions\.fail\(`, `Assert.fail(`)
}
