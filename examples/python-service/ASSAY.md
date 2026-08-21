# Assay Project Knowledge — python-service

## Framework
- Language: Python 3.11+
- Test framework: pytest 8 with pytest-asyncio
- Mock style: `unittest.mock.patch` / `MagicMock`

## Conventions
- Test files live in `tests/` mirroring the `src/` tree
  (`tests/orders/test_models.py` for `src/orders/models.py`)
- Fixtures in `conftest.py` — use `@pytest.fixture` for shared objects
- Use `pytest.raises(ExceptionClass, match="pattern")` for error assertions
- Parametrize with `@pytest.mark.parametrize` for data-driven cases
- No `self` — prefer plain functions over test classes

## Example fixture
```python
@pytest.fixture
def order_service():
    return OrderService()

@pytest.fixture
def pending_order(order_service):
    return order_service.create_order("ord-1", "cust-42")
```
