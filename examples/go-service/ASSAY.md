# Assay Project Knowledge — go-service

## Framework
- Language: Go 1.22
- Test framework: `testing` (standard library) + `testify` assertions
- Mock style: manual stubs; no mock generator

## Conventions
- Test files live alongside source (`payment_test.go` next to `payment.go`)
- Use `t.Run("description", func(t *testing.T) {...})` sub-tests
- Table-driven tests for functions with multiple input/output combinations
- Error assertions via `require.ErrorIs(t, err, ErrXxx)` (sentinel comparison)
- Name happy-path cases "success" and error cases after the sentinel they trigger

## Import style
```go
import (
    "testing"

    "github.com/stretchr/testify/require"
    "github.com/orieken/assay-example-go/internal/payment"
)
```
