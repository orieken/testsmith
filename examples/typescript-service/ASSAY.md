# Assay Project Knowledge — typescript-service

## Framework
- Language: TypeScript 5 (strict)
- Test runner: Vitest 1.x
- Assertions: Vitest built-ins (`expect`, `it`, `describe`, `beforeEach`)
- Mocks: `vi.fn()` / `vi.spyOn()` from Vitest

## Conventions
- Test files: `src/**/*.test.ts` co-located with source
- Group related tests in `describe` blocks named after the class or function
- Use `beforeEach` to reset state between tests
- Prefer `expect(fn).toThrow(ErrorClass)` for exception assertions
- No `any` — unknown inputs use `unknown` with type guards

## Example test structure
```typescript
import { describe, it, expect, beforeEach } from "vitest";
import { Catalogue } from "./catalogue.js";

describe("Catalogue", () => {
  let catalogue: Catalogue;
  beforeEach(() => { catalogue = new Catalogue(); });

  it("throws ProductNotFoundError for unknown SKU", () => {
    expect(() => catalogue.getProduct("MISSING")).toThrow("Product not found");
  });
});
```
