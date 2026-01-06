# Integrating Regression-Free Coding with GitHub Spec-Kit

> **Goal**: Minimize drift, regressions, and bloat while maximizing automated validation quality—removing human validation as the bottleneck for agentic coding scale.

---

## The Specification Correctness Problem

A fundamental challenge in agentic AI is ensuring that instructions given to an agent accurately reflect the user's intent while remaining checkable by automated systems. This is the **specification correctness problem**.

Traditional prose specifications are ambiguous. AI agents can misinterpret vague requirements, leading to correct-but-wrong implementations. The solution: **structured specification languages** that bridge natural language and formal verification.

---

## Overview: Spec-Kit + Regression-Free Architecture

Spec-kit provides the structure: **Constitution → Specification → Plan → Tasks → Implementation**

Regression-free coding provides the verification: **Types → Contracts → Properties → Mutations → Architecture → Complexity → Monitoring**

The integration maps verification constraints to each spec-kit layer:

```
┌─────────────────────────────────────────────────────────────────────────┐
│                         CONSTITUTION LAYER                              │
│  Human-authored: Coding principles, architectural patterns, budgets    │
│  Becomes: ArchUnit rules, dependency-cruiser config, complexity gates  │
├─────────────────────────────────────────────────────────────────────────┤
│                         SPECIFICATION LAYER                             │
│  Human-authored: EARS requirements + acceptance criteria                │
│  Becomes: Property tests, behavioral contracts, type signatures        │
├─────────────────────────────────────────────────────────────────────────┤
│                            PLAN LAYER                                   │
│  AI-generated: Technical design verified by Formal-LLM constraints     │
│  Verified by: Type checking, contract definitions, architecture rules  │
├─────────────────────────────────────────────────────────────────────────┤
│                            TASK LAYER                                   │
│  AI-generated: Implementation tasks with acceptance criteria           │
│  Each task: Type check + Contracts + Properties + Mutations pass      │
├─────────────────────────────────────────────────────────────────────────┤
│                        IMPLEMENTATION LAYER                             │
│  AI-generated: Actual code                                             │
│  Full 8-layer verification stack + runtime monitoring before merge    │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## Layer 1: Constitution as Executable Constraints

The constitution defines project principles. Make them **machine-enforced**, not just prose.

### Traditional Constitution (Prose)

```markdown
## Code Quality Standards
- Follow Clean Architecture
- Keep functions small and focused
- Maintain high test coverage
- Use TypeScript strict mode
```

### Regression-Free Constitution (Executable)

Create `.speckit/constitution/constraints.yaml`:

```yaml
# Constitution: Machine-Enforced Constraints
version: 1.0

# Type System Requirements
types:
  typescript:
    strict: true
    noImplicitAny: true
    strictNullChecks: true
    noUncheckedIndexedAccess: true
  python:
    mypy_strict: true
    pyright_mode: strict

# Architecture Rules (→ dependency-cruiser / ArchUnit)
architecture:
  pattern: clean-architecture
  layers:
    - name: entities
      path: src/domain/entities
      allowed_imports: []  # No external imports
    - name: usecases
      path: src/domain/usecases
      allowed_imports: [entities]
    - name: adapters
      path: src/adapters
      allowed_imports: [entities, usecases]
    - name: frameworks
      path: src/frameworks
      allowed_imports: [entities, usecases, adapters]
  forbidden:
    - from: entities
      to: [usecases, adapters, frameworks]
    - from: usecases
      to: [adapters, frameworks]
    - from: adapters
      to: [frameworks]
  no_cycles: true

# Complexity Budgets (→ eslint, radon, gocyclo)
complexity:
  max_cyclomatic_per_function: 10
  max_lines_per_function: 50
  max_lines_per_file: 400
  max_nesting_depth: 4
  max_parameters: 5
  max_dependencies_per_module: 7

# Testing Requirements
testing:
  property_tests_required: true
  metamorphic_tests_required: false  # Enable for ML/search systems
  mutation_score_threshold: 0.8
  coverage_threshold: 0.85

# Contract Requirements
contracts:
  require_preconditions: true
  require_postconditions: true
  require_class_invariants: true

# Runtime Monitoring (AgentGuard)
monitoring:
  enabled: true
  alert_on_divergence: true
  max_state_drift: 0.15

# Adversarial Review
review:
  enabled: true
  focus: [security, complexity, duplication]
```

### Constitution → CI Configuration Generator

```python
#!/usr/bin/env python3
"""Generate CI configuration from constitution constraints."""

import yaml
from pathlib import Path

def load_constitution():
    return yaml.safe_load(Path('.speckit/constitution/constraints.yaml').read_text())

def generate_eslint_config(constitution):
    complexity = constitution['complexity']
    return {
        "rules": {
            "complexity": ["error", complexity['max_cyclomatic_per_function']],
            "max-lines-per-function": ["error", complexity['max_lines_per_function']],
            "max-depth": ["error", complexity['max_nesting_depth']],
            "max-params": ["error", complexity['max_parameters']],
            "max-lines": ["error", complexity['max_lines_per_file']]
        }
    }

def generate_dependency_cruiser_config(constitution):
    arch = constitution['architecture']
    forbidden = []

    for rule in arch['forbidden']:
        for target in rule['to']:
            forbidden.append({
                'name': f"no-{rule['from']}-to-{target}",
                'from': {'path': f"^src/{rule['from']}"},
                'to': {'path': f"^src/{target}"}
            })

    if arch['no_cycles']:
        forbidden.append({
            'name': 'no-circular',
            'from': {},
            'to': {'circular': True}
        })

    return {'forbidden': forbidden}

def generate_github_actions(constitution):
    testing = constitution['testing']
    return f"""
name: Regression-Free Verification
on: [push, pull_request]
jobs:
  verify:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Type Check
        run: npx tsc --noEmit

      - name: Lint & Complexity
        run: npx eslint src/ --max-warnings 0

      - name: Architecture
        run: npx depcruise src --config --output-type err

      - name: Property Tests
        run: npm run test:property

      - name: Mutation Testing
        if: github.event_name == 'pull_request'
        run: |
          npx stryker run
          SCORE=$(cat reports/mutation/mutation.json | jq '.mutationScore')
          if (( $(echo "$SCORE < {testing['mutation_score_threshold']}" | bc -l) )); then
            echo "Mutation score $SCORE below threshold"
            exit 1
          fi
"""

if __name__ == '__main__':
    constitution = load_constitution()
    # Generate all config files from constitution
    ...
```

---

## Layer 2: Specifications with EARS Templates

The `/speckit.specify` command creates feature specifications. Use **EARS (Easy Approach to Requirements Syntax)** for machine-parseable requirements.

### EARS: Constraining Natural Language for AI

EARS provides five sentence patterns that force explicit triggers, conditions, and states:

| Pattern | Template | AI Utility |
|---------|----------|------------|
| **Ubiquitous** | The [system] shall [action] | Global Invariant |
| **Event-Driven** | When [trigger], the [system] shall [action] | Function Trigger |
| **State-Driven** | While [state], the [system] shall [action] | State Guard |
| **Unwanted Behavior** | If [condition], then the [system] shall [action] | Exception Handler |
| **Optional** | Where [feature], the [system] shall [action] | Feature Flag |

### Traditional Specification (Prose)

```markdown
## Feature: Shopping Cart

### User Story
As a customer, I want to add items to my cart so that I can purchase them later.

### Acceptance Criteria
- Users can add items to cart
- Cart shows correct total
- Items persist across sessions
```

### Regression-Free Specification with EARS

`.speckit/specs/shopping-cart/spec.yaml`:

```yaml
feature: shopping-cart
version: 1.0

user_stories:
  - as: customer
    i_want: to add items to my cart
    so_that: I can purchase them later

# EARS-formatted requirements (machine-parseable)
requirements:
  # Ubiquitous: always true
  - id: REQ-001
    pattern: ubiquitous
    ears: "The shopping cart shall maintain a non-negative total."
    property: cart.total() >= 0
    test_type: invariant

  # Event-driven: triggered by action
  - id: REQ-002
    pattern: event-driven
    ears: "When the user adds an item, the system shall increase the cart count by one."
    trigger: user.adds_item(item)
    action: cart.count() == old_cart.count() + 1
    test_type: property

  - id: REQ-003
    pattern: event-driven
    ears: "When the user removes an item, the system shall decrease the cart count."
    trigger: user.removes_item(item)
    action: cart.count() == old_cart.count() - 1
    test_type: property

  # State-driven: while condition holds
  - id: REQ-004
    pattern: state-driven
    ears: "While the cart is empty, the system shall display 'Your cart is empty'."
    state: cart.is_empty()
    action: ui.displays("Your cart is empty")
    test_type: state_machine

  # Unwanted behavior: exception handling
  - id: REQ-005
    pattern: unwanted-behavior
    ears: "If the item quantity exceeds stock, then the system shall reject the addition."
    condition: item.quantity > inventory.stock(item.id)
    action: raise OutOfStockError
    test_type: exception

  # Optional: feature flag
  - id: REQ-006
    pattern: optional
    ears: "Where express checkout is enabled, the system shall show one-click purchase."
    feature_flag: express_checkout
    action: ui.shows_one_click_button()
    test_type: conditional

# Machine-verifiable properties (derived from EARS)
properties:
  # Property 1: Adding item increases cart size (from REQ-002)
  - name: add_increases_count
    description: Adding an item increases cart item count by 1
    ears_ref: REQ-002
    property: |
      forall item in valid_items:
        let old_count = cart.count()
        cart.add(item)
        assert cart.count() == old_count + 1

  # Property 2: Adding and removing is identity
  - name: add_remove_identity
    description: Adding then removing an item leaves cart unchanged
    property: |
      forall item in valid_items:
        let old_state = cart.snapshot()
        cart.add(item)
        cart.remove(item)
        assert cart.snapshot() == old_state

  # Property 3: Total is sum of item prices (from REQ-001)
  - name: total_is_sum
    description: Cart total equals sum of item prices times quantities
    ears_ref: REQ-001
    property: |
      assert cart.total() == sum(item.price * item.quantity for item in cart.items())

  # Property 4: Persistence roundtrip
  - name: persistence_roundtrip
    description: Saving and loading cart preserves all items
    property: |
      forall cart in valid_carts:
        let saved = persistence.save(cart)
        let loaded = persistence.load(saved)
        assert loaded.items() == cart.items()

# Metamorphic relations (for oracle-free testing)
metamorphic_relations:
  - name: order_independence
    description: Adding items in different order should produce same total
    relation: |
      cart1.add(item_a); cart1.add(item_b)
      cart2.add(item_b); cart2.add(item_a)
      assert cart1.total() == cart2.total()

  - name: quantity_additivity
    description: Adding item twice equals adding with quantity 2
    relation: |
      cart1.add(item, quantity=1); cart1.add(item, quantity=1)
      cart2.add(item, quantity=2)
      assert cart1.total() == cart2.total()

# Contracts for the Cart interface
contracts:
  Cart:
    invariants:
      - all(item.quantity > 0 for item in self.items)
      - self.total() >= 0

    methods:
      add:
        preconditions:
          - item is not None
          - item.quantity > 0
        postconditions:
          - item in self.items
          - self.count() == old.count() + 1

      remove:
        preconditions:
          - item in self.items
        postconditions:
          - item not in self.items or self.get(item).quantity == old.get(item).quantity - 1

# Type signatures (for type-level verification)
types:
  CartItem:
    id: string
    name: string
    price: number  # >= 0
    quantity: number  # > 0

  Cart:
    items: () -> List[CartItem]
    add: (item: CartItem) -> void
    remove: (item: CartItem) -> void
    total: () -> number
    count: () -> number
```

### EARS → Test Generator

```python
#!/usr/bin/env python3
"""Generate tests from EARS-formatted specifications."""

import yaml
from pathlib import Path

def generate_tests_from_ears(spec):
    """Generate appropriate tests based on EARS patterns."""
    tests = []

    for req in spec['requirements']:
        pattern = req['pattern']

        if pattern == 'ubiquitous':
            # Global invariant → class invariant test
            tests.append(generate_invariant_test(req))

        elif pattern == 'event-driven':
            # Trigger-action → property test
            tests.append(generate_property_test(req))

        elif pattern == 'state-driven':
            # State guard → state machine test
            tests.append(generate_state_machine_test(req))

        elif pattern == 'unwanted-behavior':
            # Exception handling → exception test
            tests.append(generate_exception_test(req))

        elif pattern == 'optional':
            # Feature flag → conditional test
            tests.append(generate_conditional_test(req))

    return tests

def generate_property_test(req):
    """Generate Hypothesis property test from event-driven EARS."""
    return f'''
@given(valid_items())
def test_{req['id'].lower().replace('-', '_')}(item):
    """EARS: {req['ears']}"""
    cart = Cart()
    old_count = cart.count()
    cart.add(item)
    assert cart.count() == old_count + 1
'''

def generate_exception_test(req):
    """Generate exception test from unwanted-behavior EARS."""
    return f'''
def test_{req['id'].lower().replace('-', '_')}():
    """EARS: {req['ears']}"""
    cart = Cart()
    item = create_item(quantity=100)
    mock_inventory(stock=5)

    with pytest.raises(OutOfStockError):
        cart.add(item)
'''

def generate_metamorphic_tests(spec):
    """Generate metamorphic relation tests."""
    tests = []

    for relation in spec.get('metamorphic_relations', []):
        tests.append(f'''
def test_metamorphic_{relation['name']}():
    """Metamorphic: {relation['description']}"""
    {relation['relation']}
''')

    return tests
```

---

## Layer 3: Plans as Verified Architecture with Formal-LLM

The `/speckit.plan` command creates technical plans. Use **Formal-LLM constraints** to ensure plans are executable.

### Formal-LLM: Constrained Plan Generation

For multi-step agent plans, Formal-LLM enforces hard constraints via Context-Free Grammars:

```yaml
# .speckit/plans/shopping-cart/plan_grammar.yaml
grammar: |
  plan: component_design api_design data_design
  component_design: layer_assignment+
  layer_assignment: component ":" layer dependencies
  layer: "entities" | "usecases" | "adapters" | "frameworks"
  dependencies: "[" component ("," component)* "]" | "[]"

  api_design: endpoint+
  endpoint: method path request response
  method: "GET" | "POST" | "PUT" | "DELETE"

  data_design: model+
  model: name fields invariants
```

### Plan Verification Checklist

```yaml
# .speckit/plans/shopping-cart/plan.yaml
feature: shopping-cart
technical_plan:

  architecture:
    # Explicit layer assignments (verified against constitution)
    components:
      - name: CartEntity
        layer: entities
        dependencies: []  # Verified: entities have no deps

      - name: AddToCartUseCase
        layer: usecases
        dependencies: [CartEntity]  # Verified: only entity deps

      - name: CartRepository
        layer: adapters
        dependencies: [CartEntity, AddToCartUseCase]  # Verified

      - name: CartController
        layer: frameworks
        dependencies: [AddToCartUseCase, CartRepository]

  api_contracts:
    # Type-safe API definitions
    endpoints:
      - path: POST /cart/items
        request:
          body:
            type: CartItem
            validation:
              - field: quantity
                constraint: "> 0"
              - field: price
                constraint: ">= 0"
        response:
          success:
            status: 201
            body: { cart: Cart }
          errors:
            - status: 400
              when: "invalid item data"
            - status: 404
              when: "item not found in catalog"

  data_models:
    # Invariants for data structures
    Cart:
      fields:
        - name: id
          type: UUID
          constraints: [not_null]
        - name: user_id
          type: UUID
          constraints: [not_null]
        - name: items
          type: List[CartItem]
          invariants:
            - "all items have quantity > 0"
            - "no duplicate item IDs"
        - name: created_at
          type: DateTime
          constraints: [not_null, immutable]
```

### Plan Verification Script

```python
#!/usr/bin/env python3
"""Verify plan conforms to constitution and grammar."""

import yaml
from pathlib import Path

def load_constitution():
    return yaml.safe_load(
        Path('.speckit/constitution/constraints.yaml').read_text()
    )

def load_plan(feature):
    return yaml.safe_load(
        Path(f'.speckit/plans/{feature}/plan.yaml').read_text()
    )

def verify_layer_dependencies(plan, constitution):
    """Verify no forbidden dependencies in plan."""
    arch = constitution['architecture']
    forbidden = {(r['from'], t) for r in arch['forbidden'] for t in r['to']}

    errors = []
    for component in plan['architecture']['components']:
        component_layer = component['layer']
        for dep in component.get('dependencies', []):
            dep_layer = get_component_layer(plan, dep)
            if (component_layer, dep_layer) in forbidden:
                errors.append(
                    f"{component['name']} ({component_layer}) cannot depend on "
                    f"{dep} ({dep_layer})"
                )

    return errors

def verify_ears_coverage(plan, spec):
    """Ensure all EARS requirements are addressed in plan."""
    ears_ids = {r['id'] for r in spec['requirements']}
    addressed = set()

    for component in plan['architecture']['components']:
        for req_id in component.get('addresses_requirements', []):
            addressed.add(req_id)

    missing = ears_ids - addressed
    if missing:
        return [f"EARS requirement {rid} not addressed in plan" for rid in missing]
    return []

if __name__ == '__main__':
    constitution = load_constitution()
    plan = load_plan('shopping-cart')

    errors = verify_layer_dependencies(plan, constitution)
    if errors:
        print("Plan violates architecture constraints:")
        for e in errors:
            print(f"  - {e}")
        exit(1)
```

---

## Layer 4: Tasks with Machine-Verifiable Criteria

The `/speckit.tasks` command breaks plans into tasks. Each task must have **machine-verifiable completion criteria**.

### Task Definition Template

```yaml
# .speckit/tasks/shopping-cart/001-cart-entity.yaml
task:
  id: 001
  name: Implement CartEntity
  parallel: false

  description: |
    Create the Cart entity with proper invariants and value semantics.

  # Link to EARS requirements
  addresses_requirements: [REQ-001, REQ-002, REQ-003]

  files:
    - src/domain/entities/Cart.ts
    - src/domain/entities/CartItem.ts
    - tests/domain/entities/Cart.property.test.ts

  # Machine-verifiable completion criteria
  verification:
    types:
      - file: src/domain/entities/Cart.ts
        check: tsc --noEmit

    contracts:
      - class: Cart
        invariants:
          - "items.every(i => i.quantity > 0)"
          - "total >= 0"

    properties:
      - name: add_increases_count
        ears_ref: REQ-002
        test_file: tests/domain/entities/Cart.property.test.ts
        must_pass: true
      - name: add_remove_identity
        test_file: tests/domain/entities/Cart.property.test.ts
        must_pass: true

    metamorphic:
      - name: order_independence
        test_file: tests/domain/entities/Cart.metamorphic.test.ts
        must_pass: true

    mutations:
      files: [src/domain/entities/Cart.ts]
      minimum_score: 0.85

    architecture:
      file: src/domain/entities/Cart.ts
      layer: entities
      allowed_imports: []  # No external imports

    complexity:
      file: src/domain/entities/Cart.ts
      max_cyclomatic: 10
      max_lines: 200

  # Automated acceptance
  acceptance:
    all_of:
      - types.check == pass
      - all(contracts.invariants).verified
      - all(properties).must_pass
      - all(metamorphic).must_pass
      - mutations.score >= mutations.minimum_score
      - architecture.violations == 0
      - complexity.actual <= complexity.max
```

### Task Runner with Verification

```python
#!/usr/bin/env python3
"""Execute task and verify completion criteria."""

import subprocess
import yaml
from pathlib import Path

def run_task_verification(task_file):
    task = yaml.safe_load(Path(task_file).read_text())
    verification = task['task']['verification']
    results = {}

    # Layer 1: Type check
    print("Verifying types...")
    for type_check in verification.get('types', []):
        result = subprocess.run(
            type_check['check'].split(),
            capture_output=True
        )
        results['types'] = result.returncode == 0

    # Layer 2: Contract verification (runtime)
    # Contracts verified during property tests

    # Layer 3: Property tests
    print("Running property tests...")
    result = subprocess.run(
        ['pytest', '-m', 'property', '--tb=short'],
        capture_output=True
    )
    results['properties'] = result.returncode == 0

    # Layer 3b: Metamorphic tests
    if verification.get('metamorphic'):
        print("Running metamorphic tests...")
        result = subprocess.run(
            ['pytest', '-m', 'metamorphic', '--tb=short'],
            capture_output=True
        )
        results['metamorphic'] = result.returncode == 0

    # Layer 4: Mutation testing
    print("Running mutation tests...")
    for mut_check in verification.get('mutations', []):
        result = subprocess.run(
            ['npx', 'stryker', 'run', '--files'] + mut_check['files'],
            capture_output=True
        )
        score = parse_mutation_score(result.stdout)
        results['mutations'] = score >= mut_check['minimum_score']

    # Layer 5: Architecture
    print("Checking architecture...")
    result = subprocess.run(
        ['npx', 'depcruise', '--config', '--output-type', 'err'] +
        [v['file'] for v in verification.get('architecture', [])],
        capture_output=True
    )
    results['architecture'] = result.returncode == 0

    # Layer 6: Complexity
    print("Checking complexity...")
    for comp_check in verification.get('complexity', []):
        result = subprocess.run(
            ['npx', 'eslint', comp_check['file'], '--format', 'json'],
            capture_output=True
        )
        results['complexity'] = result.returncode == 0

    # Evaluate acceptance criteria
    acceptance = task['task']['acceptance']
    if 'all_of' in acceptance:
        passed = all(results.values())

    return passed, results

if __name__ == '__main__':
    import sys
    passed, results = run_task_verification(sys.argv[1])

    print("\nVerification Results:")
    for check, result in results.items():
        status = "✓" if result else "✗"
        print(f"  {status} {check}")

    if not passed:
        print("\nTask verification FAILED")
        sys.exit(1)
    print("\nTask verification PASSED")
```

---

## Layer 5: Implementation with Full Verification + Runtime Monitoring

When AI executes `/speckit.implement`, it generates code that must pass the complete verification stack plus runtime monitoring via AgentGuard.

### Implementation Loop with AgentGuard

```
┌─────────────────────────────────────────────────────────────────┐
│                      AI IMPLEMENTATION LOOP                     │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  1. AI reads task definition and verification criteria          │
│                        ↓                                        │
│  2. AI generates code                                           │
│                        ↓                                        │
│  3. Run verification stack:                                     │
│     • Type check        → FAIL? → Feedback to AI → retry        │
│     • Contract check    → FAIL? → Feedback to AI → retry        │
│     • Property tests    → FAIL? → Feedback to AI → retry        │
│     • Metamorphic tests → FAIL? → Feedback to AI → retry        │
│     • Mutation tests    → FAIL? → Feedback to AI → retry        │
│     • Architecture      → FAIL? → Feedback to AI → retry        │
│     • Complexity        → FAIL? → Feedback to AI → retry        │
│                        ↓                                        │
│  4. ALL PASS? → Deploy to staging with AgentGuard monitoring    │
│                        ↓                                        │
│  5. AgentGuard: Monitor runtime behavior against learned MDP    │
│     • Behavioral drift? → Alert and rollback                    │
│     • State divergence? → Alert and rollback                    │
│                        ↓                                        │
│  6. Stable? → Task complete → Next task                         │
│                                                                 │
│  7. Max retries exceeded? → Escalate to human (spec issue?)     │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### AgentGuard Integration

```python
from agentguard import RuntimeMonitor, MDPVerifier

class SpecKitAgentGuard:
    def __init__(self, learned_model_path: str):
        self.monitor = RuntimeMonitor()
        self.verifier = MDPVerifier.load(learned_model_path)

    def wrap_agent_execution(self, agent_fn):
        """Wrap agent execution with runtime monitoring."""

        def monitored_fn(*args, **kwargs):
            # Record initial state
            self.monitor.record_state("START")

            try:
                result = agent_fn(*args, **kwargs)

                # Record transitions
                self.monitor.record_transition(
                    from_state="START",
                    to_state="COMPLETE",
                    action=agent_fn.__name__
                )

                # Verify against learned model
                divergence = self.verifier.check_divergence(
                    self.monitor.get_trace()
                )

                if divergence > 0.15:  # Threshold from constitution
                    raise BehaviorDivergenceError(
                        f"Agent behavior diverged {divergence:.1%} from model"
                    )

                return result

            except Exception as e:
                self.monitor.record_state("ERROR")
                raise

        return monitored_fn
```

### AI Feedback Format

When verification fails, provide structured feedback including EARS traceability:

```json
{
  "verification_failed": true,
  "layer": "properties",
  "test": "test_add_increases_count",
  "ears_requirement": "REQ-002",
  "ears_text": "When the user adds an item, the system shall increase the cart count by one.",
  "error": {
    "type": "AssertionError",
    "message": "assert 2 == 1",
    "shrunk_example": {
      "item": {"id": "x", "quantity": 1, "price": 0}
    },
    "location": "tests/Cart.property.test.ts:42"
  },
  "suggestion": "The add() method may not be incrementing count correctly. Check that the item is being appended to the items array.",
  "relevant_contract": "postcondition: self.count() == old.count() + 1",
  "retry_count": 2,
  "max_retries": 5
}
```

---

## Complete Workflow Integration

### Directory Structure

```
project/
├── .speckit/
│   ├── constitution/
│   │   ├── constraints.yaml      # Machine-enforceable rules
│   │   └── README.md             # Human-readable principles
│   │
│   ├── specs/
│   │   └── shopping-cart/
│   │       ├── spec.yaml         # EARS requirements, properties, contracts
│   │       └── README.md         # User stories, context
│   │
│   ├── plans/
│   │   └── shopping-cart/
│   │       ├── plan.yaml         # Technical plan with verification
│   │       ├── plan_grammar.yaml # Formal-LLM grammar constraints
│   │       └── README.md         # Architecture decisions
│   │
│   ├── tasks/
│   │   └── shopping-cart/
│   │       ├── 001-cart-entity.yaml
│   │       ├── 002-add-usecase.yaml
│   │       └── 003-cart-controller.yaml
│   │
│   └── models/
│       └── agent_mdp.json        # Learned behavioral model for AgentGuard
│
├── src/                          # Generated implementation
├── tests/
│   ├── property/                 # Generated property tests
│   ├── metamorphic/              # Generated metamorphic tests
│   └── contracts/                # Contract test harnesses
│
├── .dependency-cruiser.js        # Generated from constitution
├── .eslintrc.js                  # Generated from constitution
├── stryker.conf.js               # Generated from constitution
└── .github/workflows/verify.yml  # Generated from constitution
```

### Workflow Commands

```bash
# 1. Initialize project with regression-free constraints
speckit init myproject --regression-free

# 2. Define constitution (creates constraints.yaml template)
speckit constitution

# 3. Generate verification configs from constitution
speckit generate-configs

# 4. Create feature specification with EARS templates
speckit specify "shopping cart feature" --ears
# → Creates spec.yaml with EARS requirement templates
# → Generates property test skeletons

# 5. Create technical plan with Formal-LLM validation
speckit plan shopping-cart --validate-grammar
# → Verifies plan against constitution before saving
# → Checks plan grammar via Formal-LLM

# 6. Generate tasks with verification criteria
speckit tasks shopping-cart
# → Each task includes machine-verifiable acceptance
# → Links tasks to EARS requirements

# 7. Implement with full verification loop + monitoring
speckit implement shopping-cart --verify --monitor
# → Runs verification stack after each task
# → Auto-retries on failure with feedback
# → Deploys to staging with AgentGuard
# → Escalates to human only for spec issues

# 8. Verify entire feature
speckit verify shopping-cart
# → Runs full 8-layer verification
# → Generates verification report with EARS traceability
```

---

## Example: Complete Feature Flow

### Step 1: Constitution

```yaml
# .speckit/constitution/constraints.yaml
architecture:
  pattern: clean-architecture
  layers: [entities, usecases, adapters, frameworks]

complexity:
  max_cyclomatic_per_function: 10
  max_lines_per_function: 50

testing:
  property_tests_required: true
  metamorphic_tests_required: true
  mutation_score_threshold: 0.8

monitoring:
  enabled: true
  max_state_drift: 0.15
```

### Step 2: Specification with EARS

```yaml
# .speckit/specs/user-auth/spec.yaml
feature: user-authentication

requirements:
  - id: REQ-AUTH-001
    pattern: ubiquitous
    ears: "The system shall never store plaintext passwords."

  - id: REQ-AUTH-002
    pattern: event-driven
    ears: "When the user provides valid credentials, the system shall return an authentication token."

  - id: REQ-AUTH-003
    pattern: unwanted-behavior
    ears: "If the user provides invalid credentials, then the system shall reject authentication."

  - id: REQ-AUTH-004
    pattern: unwanted-behavior
    ears: "If the user exceeds max login attempts, then the system shall lock the account."

properties:
  - name: password_hash_not_reversible
    ears_ref: REQ-AUTH-001
    property: |
      forall password in valid_passwords:
        hash1 = hash(password)
        hash2 = hash(password)
        assert hash1 != password
        assert hash1 == hash2  # deterministic

  - name: valid_token_authenticates
    ears_ref: REQ-AUTH-002
    property: |
      forall user in valid_users:
        token = authenticate(user.email, user.password)
        assert verify_token(token) == user

  - name: invalid_password_rejected
    ears_ref: REQ-AUTH-003
    property: |
      forall user in valid_users, wrong_pass in passwords:
        assume wrong_pass != user.password
        result = authenticate(user.email, wrong_pass)
        assert result is AuthError

metamorphic_relations:
  - name: timing_independence
    description: Authentication time should not reveal password length
    relation: |
      t1 = time(authenticate(email, "short"))
      t2 = time(authenticate(email, "verylongpassword"))
      assert abs(t1 - t2) < threshold  # Constant-time comparison

contracts:
  AuthService:
    invariants:
      - "failed_attempts[user] <= max_attempts"
    methods:
      authenticate:
        preconditions:
          - email is valid_email
          - password is not empty
        postconditions:
          - result is Token or result is AuthError
```

### Step 3: Plan (Auto-verified)

```yaml
# .speckit/plans/user-auth/plan.yaml
components:
  - name: User
    layer: entities
    addresses_requirements: [REQ-AUTH-001]

  - name: AuthenticateUseCase
    layer: usecases
    dependencies: [User]
    addresses_requirements: [REQ-AUTH-002, REQ-AUTH-003, REQ-AUTH-004]

  - name: TokenService
    layer: adapters
    dependencies: [User]
```

### Step 4: Tasks (Auto-verified on completion)

```yaml
# .speckit/tasks/user-auth/001-user-entity.yaml
task:
  id: 001
  name: User Entity
  addresses_requirements: [REQ-AUTH-001]
  verification:
    properties: [password_hash_not_reversible]
    metamorphic: [timing_independence]
    mutations:
      minimum_score: 0.85
```

### Step 5: Implementation (Auto-verified, auto-retry, monitored)

```bash
$ speckit implement user-auth --verify --monitor

Implementing task 001-user-entity...
  Attempt 1: Generating code...
  Verifying...
    ✓ Types
    ✓ Contracts
    ✗ Properties: password_hash_not_reversible failed
      EARS: REQ-AUTH-001 - The system shall never store plaintext passwords.
      Shrunk example: password=""
      Feedback: Empty password edge case not handled

  Attempt 2: Regenerating with feedback...
  Verifying...
    ✓ Types
    ✓ Contracts
    ✓ Properties
    ✓ Metamorphic (timing_independence)
    ✓ Mutations (score: 0.89)
    ✓ Architecture
    ✓ Complexity

  Deploying to staging with AgentGuard...
    ✓ Behavioral model stable (drift: 0.03)
    ✓ No state divergence detected

  ✓ Task 001 complete

Implementing task 002-authenticate-usecase...
...
```

---

## Key Benefits

| Aspect | Traditional Spec-Kit | Regression-Free Spec-Kit |
|--------|---------------------|--------------------------|
| Requirements | Prose descriptions | EARS-formatted, machine-parseable |
| Acceptance Criteria | Human interpretation | Machine-verifiable properties |
| Code Review | Human required | Automated (8-layer stack) |
| Runtime Safety | None | AgentGuard monitoring |
| Failure Handling | Human debugging | Auto-retry with EARS-traced feedback |
| Scaling | Linear in human reviewers | Unlimited AI agents |
| Confidence | Depends on review quality | Mathematically constrained |
| Traceability | Manual | Automated EARS → test → code |

---

## Summary

Integrating regression-free coding with spec-kit transforms the development process:

1. **Constitution** becomes executable constraints, not just principles
2. **Specifications** use EARS templates for unambiguous, machine-parseable requirements
3. **Plans** are validated against grammar constraints via Formal-LLM
4. **Tasks** have machine-verifiable completion criteria with EARS traceability
5. **Implementation** runs in a verification loop with auto-retry and runtime monitoring

### The Scaling Unlock

```
Traditional:     Throughput = min(Agent Capacity, Human Review Bandwidth)
Regression-Free: Throughput = Agent Capacity × Automated Validation Coverage
```

By minimizing failure modes (drift, regressions, bloat) and maximizing automated validation quality, human involvement shifts from per-change bottleneck to upfront specification—enabling true agentic parallelization.

---

## References

- EARS: Easy Approach to Requirements Syntax (Rolls-Royce, Alistair Mavin)
- AgentGuard: Runtime Verification of AI Agents (arXiv 2509.23864)
- Formal-LLM: Integrating Formal Language for Controllable Agents (arXiv 2402.00798)
- Effect/Schema: TypeScript validation and type inference
- Metamorphic Testing: A Review of Challenges and Opportunities (HKU)
- Martin Kleppmann: AI will make formal verification go mainstream
