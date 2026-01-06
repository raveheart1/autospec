# Task: Property-Based Testing

> Part of: [Regression-Free Integration](./00-overview.md)
> Reference: `.dev/tasks/SPECKIT_REGRESSION_FREE_INTEGRATION.md` (Layer 2: Specifications - Properties)

## Summary

Extend spec.yaml schema to support property definitions that can be automatically translated into property-based tests. Properties describe invariants that must hold across many randomly generated inputs, providing stronger guarantees than example-based tests.

## Configuration

### Config Toggle

```yaml
# .autospec/config.yml
verification:
  property_tests: true  # Enable/disable this feature
```

- **Config key**: `verification.property_tests`
- **Default**: `false`
- **Level presets**: Enabled at `full` level only
- **Override**: Can be explicitly set regardless of level

## Motivation

Example-based tests verify specific cases. Property-based tests verify invariants across thousands of generated inputs, finding edge cases humans miss. When properties are defined in the spec alongside EARS requirements, they:

1. Force precise thinking about behavior
2. Enable automatic test generation
3. Catch edge cases through shrinking
4. Link directly to requirements for traceability

## Design

### Spec Schema Extension

New optional `properties` block in spec.yaml:

```yaml
# specs/001-shopping-cart/spec.yaml
feature: shopping-cart

properties:
  # Property linked to EARS requirement
  - id: "PROP-001"
    name: add_increases_count
    description: "Adding an item increases cart item count by 1"
    ears_ref: "EARS-002"  # Links to EARS requirement
    
    # Property definition in pseudo-code
    property: |
      forall item in valid_items:
        let old_count = cart.count()
        cart.add(item)
        assert cart.count() == old_count + 1
    
    # Generator hints for test framework
    generators:
      item: valid_cart_item  # References generator definition
    
    # Expected test framework
    test_framework: hypothesis  # hypothesis | fast-check | gopter | etc.
  
  # Roundtrip property
  - id: "PROP-002"
    name: add_remove_identity
    description: "Adding then removing an item leaves cart unchanged"
    
    property: |
      forall item in valid_items:
        let old_state = cart.snapshot()
        cart.add(item)
        cart.remove(item)
        assert cart.snapshot() == old_state
    
    generators:
      item: valid_cart_item
  
  # Algebraic property
  - id: "PROP-003"
    name: total_is_sum
    description: "Cart total equals sum of item prices times quantities"
    ears_ref: "EARS-001"
    
    property: |
      forall cart in valid_carts:
        assert cart.total() == sum(item.price * item.quantity for item in cart.items())
    
    generators:
      cart: valid_cart_with_items

# Generator definitions (reusable across properties)
generators:
  valid_cart_item:
    type: CartItem
    constraints:
      id: non_empty_string
      name: non_empty_string
      price: float_range(0.01, 10000.00)
      quantity: int_range(1, 100)
  
  valid_cart_with_items:
    type: Cart
    constraints:
      items: list_of(valid_cart_item, min=0, max=50)
```

### Property Types

| Type | Pattern | Use Case |
|------|---------|----------|
| **Invariant** | `forall x: P(x)` | Must always hold |
| **Roundtrip** | `decode(encode(x)) == x` | Serialization, transformations |
| **Idempotent** | `f(f(x)) == f(x)` | Formatting, normalization |
| **Commutative** | `f(a,b) == f(b,a)` | Order-independent operations |
| **Algebraic** | `f(g(x)) == h(x)` | Mathematical relationships |

### Test Generation

When `verification.level` is `enhanced` or `full`, the `/autospec.tasks` command generates test stubs:

```python
# Generated: tests/property/test_cart_properties.py
import hypothesis
from hypothesis import given, strategies as st

from generators import valid_cart_item, valid_cart_with_items
from domain.cart import Cart

class TestCartProperties:
    """Property tests generated from spec.yaml"""
    
    @given(item=valid_cart_item())
    def test_add_increases_count(self, item):
        """PROP-001: Adding an item increases cart item count by 1
        
        EARS-002: When the user adds an item, the system shall 
        increase the cart count by one.
        """
        cart = Cart()
        old_count = cart.count()
        cart.add(item)
        assert cart.count() == old_count + 1
    
    @given(item=valid_cart_item())
    def test_add_remove_identity(self, item):
        """PROP-002: Adding then removing an item leaves cart unchanged"""
        cart = Cart()
        old_state = cart.snapshot()
        cart.add(item)
        cart.remove(item)
        assert cart.snapshot() == old_state
    
    @given(cart=valid_cart_with_items())
    def test_total_is_sum(self, cart):
        """PROP-003: Cart total equals sum of item prices times quantities
        
        EARS-001: The shopping cart shall maintain a non-negative total.
        """
        expected = sum(item.price * item.quantity for item in cart.items())
        assert cart.total() == expected
```

### Generator Definition Language

Generators define how to create valid test inputs:

```yaml
generators:
  # Primitive constraints
  non_empty_string:
    type: string
    constraints:
      min_length: 1
      max_length: 100
  
  # Numeric ranges
  positive_int:
    type: int
    constraints:
      min: 1
      max: 2147483647
  
  # Composite types
  valid_user:
    type: User
    constraints:
      id: uuid
      email: email_format
      age: int_range(18, 120)
  
  # Collections
  user_list:
    type: list
    element: valid_user
    constraints:
      min_length: 0
      max_length: 100
  
  # Conditional/filtered
  premium_user:
    base: valid_user
    filter: "user.subscription == 'premium'"
```

### Integration with Verify Command

```bash
# Run property tests
autospec verify cart-feature --properties

# Output with shrunk counterexample
Verifying properties: cart-feature

  ✓ PROP-001: add_increases_count (1000 examples)
  ✗ PROP-002: add_remove_identity
    FAIL after 47 examples
    Shrunk to: item=CartItem(id="", quantity=1, price=0.0)
    Error: KeyError - item not found in cart
    
    Linked requirement: EARS-002
    
  ✓ PROP-003: total_is_sum (1000 examples)

Properties: 2/3 passed
```

### Shrinking and Counterexamples

When a property fails, the test framework shrinks the input to the minimal failing case. This is captured in structured feedback:

```json
{
  "property_id": "PROP-002",
  "status": "failed",
  "examples_run": 47,
  "counterexample": {
    "shrunk": true,
    "inputs": {
      "item": {"id": "", "quantity": 1, "price": 0.0}
    }
  },
  "error": {
    "type": "KeyError",
    "message": "item not found in cart"
  },
  "ears_ref": "EARS-002",
  "suggestion": "The remove method may not handle items with empty IDs correctly"
}
```

## Implementation Notes

### Schema Package

Extend `internal/validation/`:

- Property struct with generators, assertions, refs
- Generator struct with type constraints
- Property syntax validation
- EARS reference validation

### Test Generator Package

New package `internal/proptest/`:

- Parse property definitions
- Generate test code for target framework
- Generator-to-strategy translation
- Language-specific output (Python/Go/TS)

### Framework Support

| Language | Framework | Generator Style |
|----------|-----------|-----------------|
| Python | Hypothesis | `@given(st.integers())` |
| TypeScript | fast-check | `fc.integer()` |
| Go | gopter | `gen.Int()` |

### Integration Points

- Spec validation includes property syntax checking
- Task generation includes property test references
- Verify command runs property tests
- Structured feedback includes shrunk counterexamples

## Acceptance Criteria

1. Existing specs without properties block parse correctly
2. Property definitions validate syntax and generator references
3. EARS references must exist in spec
4. Test stubs generated for configured framework
5. `autospec verify --properties` runs property tests
6. Counterexamples include shrunk minimal inputs
7. Properties linked to tasks via verification block

## Dependencies

- `01-verification-config.md` (uses verification level)
- `02-ears-spec-schema.md` (for EARS reference linking)
- ~~`04-verify-command.md`~~ **SKIPPED** - verify command not implemented

## Estimated Scope

Medium-Large. Schema extension, generator DSL, test code generation, framework adapters.
