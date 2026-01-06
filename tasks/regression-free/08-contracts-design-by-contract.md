# Task: Contracts and Design-by-Contract

> Part of: [Regression-Free Integration](./00-overview.md)
> Reference: `.dev/tasks/SPECKIT_REGRESSION_FREE_INTEGRATION.md` (Layer 2: Specifications - Contracts)

## Summary

Extend spec.yaml schema to support Design-by-Contract (DbC) definitions: class invariants, method preconditions, and postconditions. These contracts become machine-verifiable assertions that bridge specification and implementation.

## Configuration

### Config Toggle

```yaml
# .autospec/config.yml
verification:
  contracts: true  # Enable/disable this feature
```

- **Config key**: `verification.contracts`
- **Default**: `false`
- **Level presets**: Enabled at `enhanced` and `full` levels
- **Override**: Can be explicitly set regardless of level

## Motivation

Traditional acceptance criteria describe what should happen in prose. Contracts formalize these as executable assertions:

- **Preconditions**: What must be true before a method runs
- **Postconditions**: What must be true after a method completes  
- **Invariants**: What must always be true for a class/module

When contracts are explicit in the spec, AI agents know exactly what to implement, and verification can check contract violations automatically.

## Design

### Spec Schema Extension

New optional `contracts` block in spec.yaml:

```yaml
# specs/001-shopping-cart/spec.yaml
feature: shopping-cart

contracts:
  Cart:
    description: "Shopping cart entity with item management"
    
    invariants:
      - id: "INV-001"
        description: "All items have positive quantity"
        assertion: "all(item.quantity > 0 for item in self.items)"
      
      - id: "INV-002"
        description: "Total is never negative"
        assertion: "self.total() >= 0"
      
      - id: "INV-003"
        description: "No duplicate item IDs"
        assertion: "len(self.items) == len(set(item.id for item in self.items))"
    
    methods:
      add:
        description: "Add an item to the cart"
        preconditions:
          - id: "PRE-001"
            description: "Item must not be nil"
            assertion: "item is not None"
          - id: "PRE-002"
            description: "Item quantity must be positive"
            assertion: "item.quantity > 0"
        postconditions:
          - id: "POST-001"
            description: "Item exists in cart"
            assertion: "item in self.items"
          - id: "POST-002"
            description: "Cart count increased"
            assertion: "self.count() == old.count() + 1"
            uses_old: true
      
      remove:
        description: "Remove an item from the cart"
        preconditions:
          - id: "PRE-003"
            description: "Item must exist in cart"
            assertion: "item in self.items"
        postconditions:
          - id: "POST-003"
            description: "Item removed or quantity decreased"
            assertion: "item not in self.items or self.get(item).quantity == old.get(item).quantity - 1"
            uses_old: true
      
      total:
        description: "Calculate cart total"
        preconditions: []  # None required
        postconditions:
          - id: "POST-004"
            description: "Total equals sum of item prices"
            assertion: "result == sum(item.price * item.quantity for item in self.items)"

  CartItem:
    description: "Item in a shopping cart"
    
    invariants:
      - id: "INV-004"
        description: "Price is non-negative"
        assertion: "self.price >= 0"
      - id: "INV-005"
        description: "Quantity is positive"
        assertion: "self.quantity > 0"
```

### Contract Assertion Language

Use a simplified, language-agnostic assertion syntax:

| Construct | Meaning | Example |
|-----------|---------|---------|
| `self` | Current instance | `self.count()` |
| `old` | State before method call | `old.count()` |
| `result` | Method return value | `result >= 0` |
| `all(...)` | Universal quantifier | `all(x > 0 for x in items)` |
| `any(...)` | Existential quantifier | `any(x.id == id for x in items)` |
| `implies(a, b)` | Logical implication | `implies(empty, total == 0)` |

### Contract-to-Code Generation Hints

Contracts inform code generation. The `/autospec.implement` command uses contracts to:

1. Generate assertion statements in implementation
2. Add validation at method entry (preconditions)
3. Add validation at method exit (postconditions)
4. Generate invariant check methods

Example generated Go code:

```go
func (c *Cart) Add(item CartItem) error {
    // Preconditions (from PRE-001, PRE-002)
    if item == (CartItem{}) {
        return ErrNilItem
    }
    if item.Quantity <= 0 {
        return ErrInvalidQuantity
    }
    
    oldCount := c.Count()
    
    // Implementation
    c.items = append(c.items, item)
    
    // Postconditions (from POST-001, POST-002)
    assert(c.contains(item), "POST-001: item must be in cart")
    assert(c.Count() == oldCount+1, "POST-002: count must increase by 1")
    
    // Invariants checked on exit
    c.checkInvariants()
    
    return nil
}
```

### Validation Rules

When `verification.level` is `enhanced` or `full`:

1. Contract IDs must be unique within the spec
2. Assertions must use valid syntax
3. `uses_old: true` required when assertion references `old`
4. Cross-references to contracts from tasks must be valid

### Integration with Tasks

Tasks can reference contracts they implement:

```yaml
# tasks.yaml
tasks:
  - id: "TASK-001"
    title: "Implement Cart.Add method"
    
    implements_contracts:
      - class: Cart
        method: add
        contracts: [PRE-001, PRE-002, POST-001, POST-002]
    
    verification:
      contracts:
        must_satisfy: [PRE-001, PRE-002, POST-001, POST-002]
```

### Contract Verification

The `autospec verify` command checks contracts:

```bash
# Verify contracts are satisfied
autospec verify cart-feature --contracts

# Output
Verifying contracts: cart-feature

Cart.add
  ✓ PRE-001: item is not None
  ✓ PRE-002: item.quantity > 0
  ✓ POST-001: item in self.items
  ✗ POST-002: self.count() == old.count() + 1
    FAIL: count increased by 2 (duplicate add?)

Cart invariants
  ✓ INV-001: all items have positive quantity
  ✓ INV-002: total is non-negative
  ✗ INV-003: no duplicate item IDs
    FAIL: found duplicate ID "item-123"
```

## Implementation Notes

### Schema Package

Extend `internal/validation/` with:

- Contract struct with invariants and methods
- MethodContract struct with pre/postconditions
- Assertion parser (validate syntax)
- Cross-reference validation

### Contract Parser

New package `internal/contracts/`:

- Parse assertion expressions
- Validate `old` usage matches `uses_old` flag
- Generate verification code hints

### Integration Points

- Spec validation checks contract syntax
- Task generation references contracts
- Verify command includes contract checking
- Implement command uses contracts for code generation hints

## Acceptance Criteria

1. Existing specs without contracts block parse correctly
2. Contracts validate assertion syntax
3. Invalid contract references produce helpful errors
4. Contracts are linked to tasks via `implements_contracts`
5. `autospec verify --contracts` checks contract satisfaction
6. Contract IDs traceable through implementation
7. Documentation includes contract examples

## Language Support

Initial focus on language-agnostic assertions that map to:

| Language | Assertion Mechanism |
|----------|---------------------|
| Go | Custom assert package or explicit checks |
| TypeScript | Runtime assertions or Effect/Schema |
| Python | assert statements or contracts library |

Future: Generate language-specific contract libraries.

## Dependencies

- `01-verification-config.md` (uses verification level)
- `04-verify-command.md` (extends verify with contract checking)

## Estimated Scope

Medium-Large. Schema extension, assertion parser, verification integration, code generation hints.
