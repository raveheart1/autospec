# Task: Metamorphic Testing

> Part of: [Regression-Free Integration](./00-overview.md)
> Reference: `.dev/tasks/SPECKIT_REGRESSION_FREE_INTEGRATION.md` (Layer 2: Specifications - Metamorphic Relations)

## Summary

Extend spec.yaml schema to support metamorphic relations—relationships between inputs and outputs that must hold even when the exact output is unknown. This enables testing of systems where oracle values are difficult or impossible to compute (ML models, search, optimization).

## Configuration

### Config Toggle

```yaml
# .autospec/config.yml
verification:
  metamorphic_tests: true  # Enable/disable this feature
```

- **Config key**: `verification.metamorphic_tests`
- **Default**: `false`
- **Level presets**: Enabled at `full` level only
- **Override**: Can be explicitly set regardless of level

## Motivation

Property-based tests require knowing the expected output. But for many systems, the correct answer is hard to compute independently:

- **Search engines**: What's the "correct" ranking?
- **ML models**: What's the "correct" prediction?
- **Optimization**: What's the "correct" optimal solution?
- **Numerical algorithms**: Floating-point makes exact comparison hard

Metamorphic testing sidesteps the oracle problem by testing **relationships** between outputs rather than exact values.

## The Oracle Problem

Traditional testing: `assert f(input) == expected_output`

But what if you can't compute `expected_output`?

Metamorphic testing: `assert relation(f(input1), f(input2))` where the relation is known even when individual outputs aren't.

## Design

### Spec Schema Extension

New optional `metamorphic_relations` block in spec.yaml:

```yaml
# specs/001-shopping-cart/spec.yaml
feature: shopping-cart

metamorphic_relations:
  # Order independence: adding items in different order produces same total
  - id: "MR-001"
    name: order_independence
    description: "Adding items in different order should produce same total"
    
    relation: |
      # Setup: two carts, same items, different order
      cart1 = Cart()
      cart2 = Cart()
      
      cart1.add(item_a)
      cart1.add(item_b)
      
      cart2.add(item_b)
      cart2.add(item_a)
      
      # Metamorphic relation: totals must be equal
      assert cart1.total() == cart2.total()
    
    inputs:
      item_a: valid_cart_item
      item_b: valid_cart_item
  
  # Quantity additivity: adding twice equals adding with double quantity
  - id: "MR-002"
    name: quantity_additivity
    description: "Adding item twice equals adding with quantity 2"
    
    relation: |
      cart1 = Cart()
      cart2 = Cart()
      
      cart1.add(item, quantity=1)
      cart1.add(item, quantity=1)
      
      cart2.add(item, quantity=2)
      
      assert cart1.total() == cart2.total()
    
    inputs:
      item: valid_cart_item
  
  # Scaling: doubling all prices doubles total
  - id: "MR-003"
    name: price_scaling
    description: "Doubling all item prices should double the total"
    
    relation: |
      cart1 = Cart()
      cart2 = Cart()
      
      for item in items:
        cart1.add(item)
        scaled_item = item.with_price(item.price * 2)
        cart2.add(scaled_item)
      
      assert cart2.total() == cart1.total() * 2
    
    inputs:
      items: list_of(valid_cart_item, min=1, max=10)

# Example for search/ML systems
  - id: "MR-004"
    name: subset_relevance
    description: "Results for subset query should be subset of superset query results"
    
    relation: |
      results_abc = search("a b c")
      results_ab = search("a b")
      
      # Metamorphic relation: more specific query yields subset
      assert set(results_abc).issubset(set(results_ab))
    
    inputs:
      # Query terms generated
```

### Common Metamorphic Relation Patterns

| Pattern | Description | Example |
|---------|-------------|---------|
| **Permutation** | Reordering inputs shouldn't change result | `sort([3,1,2]) == sort([1,2,3])` |
| **Scaling** | Scaling input scales output proportionally | `2 * f(x) == f(2x)` |
| **Addition** | Adding to input adds to output | `f(x) + c == f(x + c)` |
| **Subset** | Subset input yields subset output | `f(A) ⊆ f(A ∪ B)` |
| **Negation** | Negating input negates/inverts output | `f(-x) == -f(x)` |
| **Idempotence** | Applying twice equals applying once | `f(f(x)) == f(x)` |
| **Composition** | Operations can be composed | `f(g(x)) == h(x)` |

### Metamorphic Test Generation

```python
# Generated: tests/metamorphic/test_cart_metamorphic.py
import hypothesis
from hypothesis import given, strategies as st

from generators import valid_cart_item
from domain.cart import Cart

class TestCartMetamorphic:
    """Metamorphic relation tests generated from spec.yaml"""
    
    @given(
        item_a=valid_cart_item(),
        item_b=valid_cart_item()
    )
    def test_order_independence(self, item_a, item_b):
        """MR-001: Adding items in different order should produce same total"""
        cart1 = Cart()
        cart2 = Cart()
        
        cart1.add(item_a)
        cart1.add(item_b)
        
        cart2.add(item_b)
        cart2.add(item_a)
        
        assert cart1.total() == cart2.total()
    
    @given(item=valid_cart_item())
    def test_quantity_additivity(self, item):
        """MR-002: Adding item twice equals adding with quantity 2"""
        cart1 = Cart()
        cart2 = Cart()
        
        cart1.add(item, quantity=1)
        cart1.add(item, quantity=1)
        
        cart2.add(item, quantity=2)
        
        assert cart1.total() == cart2.total()
    
    @given(items=st.lists(valid_cart_item(), min_size=1, max_size=10))
    def test_price_scaling(self, items):
        """MR-003: Doubling all item prices should double the total"""
        cart1 = Cart()
        cart2 = Cart()
        
        for item in items:
            cart1.add(item)
            scaled_item = item.with_price(item.price * 2)
            cart2.add(scaled_item)
        
        assert abs(cart2.total() - cart1.total() * 2) < 0.01  # Float tolerance
```

### Integration with Verify Command

```bash
# Run metamorphic tests
autospec verify cart-feature --metamorphic

# Output
Verifying metamorphic relations: cart-feature

  ✓ MR-001: order_independence (500 examples)
  ✓ MR-002: quantity_additivity (500 examples)
  ✗ MR-003: price_scaling
    FAIL after 23 examples
    Shrunk to: items=[CartItem(price=0.01, quantity=1)]
    Expected: cart2.total() == 0.02
    Got: cart2.total() == 0.019999999999999997
    
    Note: Floating-point precision issue detected
    Suggestion: Use decimal arithmetic or tolerance comparison

Metamorphic relations: 2/3 passed
```

### Difference from Property Testing

| Aspect | Property Testing | Metamorphic Testing |
|--------|------------------|---------------------|
| **Oracle** | Required (know expected output) | Not required |
| **What's tested** | Single input → expected output | Relationship between outputs |
| **Use case** | Deterministic, computable functions | Search, ML, optimization |
| **Example** | `sort([3,1,2]) == [1,2,3]` | `sort(A) == sort(permute(A))` |

### Structured Feedback

```json
{
  "metamorphic_id": "MR-003",
  "status": "failed",
  "examples_run": 23,
  "counterexample": {
    "shrunk": true,
    "inputs": {
      "items": [{"price": 0.01, "quantity": 1}]
    },
    "outputs": {
      "cart1_total": 0.01,
      "cart2_total": 0.019999999999999997
    }
  },
  "relation_violated": "cart2.total() == cart1.total() * 2",
  "suggestion": "Floating-point precision issue. Consider using decimal.Decimal or math.isclose() for comparison."
}
```

## Implementation Notes

### Schema Package

Extend `internal/validation/`:

- MetamorphicRelation struct with relation code, inputs
- Relation syntax validation
- Input generator references

### Test Generator

Extend `internal/proptest/` (or new package):

- Parse metamorphic relation definitions
- Generate paired test executions
- Handle tolerance for float comparisons

### Floating-Point Handling

Metamorphic tests often expose floating-point issues. Options:

1. Auto-detect numeric comparisons and add tolerance
2. Require explicit tolerance in relation definition
3. Recommend decimal types in feedback

## Acceptance Criteria

1. Existing specs without metamorphic_relations block parse correctly
2. Relations validate syntax and input references
3. Test stubs generated with paired execution pattern
4. `autospec verify --metamorphic` runs metamorphic tests
5. Floating-point comparisons handled gracefully
6. Counterexamples show both outputs being compared
7. Clear distinction from property tests in output

## Use Cases Beyond Shopping Cart

```yaml
# Search system
metamorphic_relations:
  - id: "MR-SEARCH-001"
    name: query_subset
    description: "More specific query yields subset of results"
    relation: |
      broad = search("python")
      narrow = search("python tutorial")
      assert set(narrow).issubset(set(broad))

# ML model
metamorphic_relations:
  - id: "MR-ML-001"
    name: input_perturbation
    description: "Small input change shouldn't drastically change output"
    relation: |
      pred1 = model.predict(image)
      pred2 = model.predict(add_noise(image, epsilon=0.01))
      assert cosine_similarity(pred1, pred2) > 0.95

# Optimization
metamorphic_relations:
  - id: "MR-OPT-001"
    name: constraint_relaxation
    description: "Relaxing constraints shouldn't worsen optimal value"
    relation: |
      opt1 = optimize(problem_with_constraint)
      opt2 = optimize(problem_without_constraint)
      assert opt2.value >= opt1.value
```

## Dependencies

- `01-verification-config.md` (uses verification level)
- ~~`04-verify-command.md`~~ **SKIPPED** - verify command not implemented
- `09-property-based-testing.md` (shares generator infrastructure)

## Estimated Scope

Medium. Schema extension, test generation (builds on property test infra), relation validation.

## References

- Chen et al., "Metamorphic Testing: A Review of Challenges and Opportunities"
- Segura et al., "A Survey on Metamorphic Testing"
- HKU Metamorphic Testing Research Group
