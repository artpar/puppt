# Puppt SWE Skill Binding Layer

This document is binding engineering doctrine for Puppt. The sections below apply to every implementation, test, fixture, CLI contract, dependency decision, validation rule, and release decision.

Puppt-specific interpretation:

1. Puppt is a production-grade agent-first `.pptx` inspection, editing, creation, validation, and review tool.
2. Puppt v1 MUST be implemented in Go.
3. Go code MUST be written as maintainable production software, not as a script-like prototype.
4. `.pptx` files are structured Open XML packages. Puppt MUST treat them as structured documents, not visual canvases.
5. Correctness means preserving editability, preserving untargeted content, reporting uncertainty, and validating output.
6. Speed, convenience, broad best-effort behavior, or attractive demos MUST NOT override preservation, explicit errors, and traceable changes.
7. The command-line and JSON result shapes are interfaces. They require compatibility discipline.
8. Test fixtures and golden outputs are part of the system because they prove behavior across real package structures.
9. Unsupported content MUST be preserved where possible and reported when relevant. Unknown content MUST NOT be silently dropped.
10. Reliable third-party Go libraries SHOULD be used wherever they reduce non-core infrastructure risk.
11. Puppt MUST own the authoritative `.pptx` package reader/writer and mutation path because that preservation layer is the product's core USP.
12. Any dependency that reads, writes, renders, mutates, validates, or interprets `.pptx` content is a controlled dependency and requires justification.

## Go Engineering Rules for Puppt

These rules specialize the doctrine below for Go implementation.

1. Every package MUST have a narrow responsibility and tests.
2. CLI code in `cmd/puppt` MUST stay thin. Business logic belongs in internal packages.
3. Use `context.Context` for operations that may perform I/O or later become cancellable.
4. Return explicit errors. Do not use `panic` for ordinary user input, malformed decks, unsupported operations, ambiguity, no-match cases, or validation failures.
5. Wrap errors with operation and file/part context while avoiding secrets or excessive document content in logs and errors.
6. Prefer typed or classified errors for unsupported file type, malformed package, unsupported feature, ambiguous target, no match, validation failure, and internal failure.
7. Use deterministic output ordering for JSON, inspection results, plans, warnings, changes, and tests.
8. Prefer reliable, maintained Go libraries over bespoke code for non-core infrastructure when they make Puppt safer or more complete; isolate them behind Puppt-owned internal interfaces.
9. Use Go standard library ZIP, XML, JSON, filesystem, and testing facilities directly for the core `.pptx` reader/writer unless a future decision explicitly proves a helper library does not control or obscure Puppt's preservation logic.
10. Do not shell out to office software as part of the core product path.
11. Do not use reflection-heavy or global-state-heavy designs where plain structs and interfaces are sufficient.
12. Do not hide PowerPoint package complexity behind an abstraction that prevents precise preservation or validation.
13. Keep public structs additive and versioned once they appear in CLI/API JSON.
14. Fixture builders MUST produce deterministic `.pptx` files or deterministic normalized expectations.
15. Golden tests MUST normalize values that are legitimately unstable, and only those values.
16. Round-trip tests MUST prove both requested changes and preservation of unrelated content.

## Puppt Definition of Done

A Puppt change is not done until:

1. The relevant source documents have been checked.
2. The current code path has been inspected before editing.
3. The implementation is the smallest coherent step toward the v1 goal.
4. Tests or fixtures cover the behavior at the right level of risk.
5. Validation commands have been run and recorded.
6. JSON and human-facing output remain honest and stable.
7. Unsupported behavior is either preserved, skipped with explanation, or rejected before mutation.
8. The progress record states changed files, verification results, remaining risks, and the next checkpoint.

The rest of this file supplies the general engineering law that governs those rules.

---

# 0. Authority, Scope, and Normative Language

## 0.1 Purpose

This doctrine exists to govern the creation, operation, maintenance, and retirement of reliable software systems that may live for decades, outlast teams, grow by orders of magnitude, and operate in hostile production environments.

It optimizes for:

1. correctness
2. reliability
3. maintainability
4. debuggability
5. predictability
6. operability
7. long-term evolution
8. clarity
9. recoverability
10. minimizing hidden complexity

It explicitly does **not** optimize for novelty, trendiness, developer ego, unchecked velocity, framework fashion, or short-term delivery at the expense of future reliability.

## 0.2 Normative Keywords

The following words have binding meaning:

| Term          | Meaning                                                                                      |
| ------------- | -------------------------------------------------------------------------------------------- |
| **MUST**      | Required. Violation blocks merge, release, or operation unless an approved exception exists. |
| **MUST NOT**  | Forbidden. Violation requires correction or formal risk acceptance.                          |
| **SHOULD**    | Required by default. Exception requires documented rationale.                                |
| **MAY**       | Permitted, but still subject to safety, security, compatibility, and ownership rules.        |
| **EXCEPTION** | A documented, time-bounded deviation approved by the accountable authority.                  |
| **WAIVER**    | A temporary exception with an owner, expiry date, risk statement, and remediation plan.      |
| **OWNER**     | A named person or team accountable for correctness, operation, and lifecycle of an artifact. |

## 0.3 Universal Rule Accountability Format

Every normative rule in this doctrine is interpreted through the following accountability format, even when abbreviated:

| Field            | Required Meaning                                              |
| ---------------- | ------------------------------------------------------------- |
| **Rule**         | What must or must not be done.                                |
| **Why**          | The engineering reason the rule exists.                       |
| **Prevents**     | The failure mode the rule blocks or reduces.                  |
| **Tradeoff**     | The cost introduced by the rule.                              |
| **Exception**    | When the rule may be bypassed.                                |
| **Verification** | How compliance is proven before merge, release, or operation. |
| **Detection**    | How violations are found after the fact.                      |
| **Correction**   | How violations are repaired.                                  |

A rule without a verification method is not considered enforceable. An unenforceable rule MUST be rewritten or removed.

---

# 1. Terminology

## 1.1 System

A **system** is a set of software, data, configuration, infrastructure, people, procedures, dependencies, and operational practices that together deliver behavior.

A system is not only code.

## 1.2 Component

A **component** is a deployable or non-deployable unit with a defined responsibility, owner, interface, and lifecycle.

Examples: service, library, database schema, queue topic, CLI tool, batch job, frontend module, infrastructure module.

## 1.3 Module

A **module** is a code-level unit that hides implementation details behind a stable interface.

A directory is not automatically a module. A module exists only if it has:

1. explicit responsibility
2. controlled dependencies
3. public interface
4. tests
5. owner

## 1.4 Service

A **service** is an independently deployable runtime component that communicates with other components over a process or network boundary.

A service MUST have:

1. owner
2. SLOs where user-facing or dependency-facing
3. runbook
4. telemetry
5. deployment procedure
6. rollback procedure
7. data ownership declaration
8. security boundary declaration

## 1.5 Interface

An **interface** is any boundary through which behavior is consumed.

Interfaces include APIs, events, database schemas, files, CLIs, SDKs, configuration formats, logs consumed by automation, and operational procedures.

## 1.6 State

**State** is any information retained across time that influences future behavior.

State includes database rows, caches, queues, files, object storage, configuration, sessions, feature flags, locks, checkpoints, indexes, metrics used for automation, and external provider state.

## 1.7 Invariant

An **invariant** is a condition that must always remain true.

Example: “Every payment capture MUST reference exactly one authorized payment intent.”

Every critical invariant MUST be documented, tested, monitored where feasible, and protected by code or storage constraints.

## 1.8 Failure, Fault, Defect, Incident

| Term         | Meaning                                                                |
| ------------ | ---------------------------------------------------------------------- |
| **Defect**   | A flaw in design, code, configuration, process, or documentation.      |
| **Fault**    | A runtime condition that may cause incorrect behavior.                 |
| **Failure**  | Externally observable incorrect behavior.                              |
| **Incident** | A failure or credible risk requiring coordinated operational response. |

## 1.9 Release and Deployment

A **release** makes a version eligible for production.

A **deployment** moves a version into an environment.

A released version may not be deployed. A deployed version may be rolled back. Release and deployment MUST be separately tracked.

## 1.10 Compatibility

**Backward compatibility** means new producers or providers continue to support old consumers.

**Forward compatibility** means old consumers tolerate new data or behavior without failure.

## 1.11 SLI, SLO, SLA, Error Budget

| Term             | Meaning                                                   |
| ---------------- | --------------------------------------------------------- |
| **SLI**          | Service Level Indicator: measured reliability signal.     |
| **SLO**          | Service Level Objective: internal reliability target.     |
| **SLA**          | Service Level Agreement: external contractual commitment. |
| **Error Budget** | Permitted unreliability over a defined window.            |

SLOs govern engineering decisions. SLAs govern external obligations. SLAs MUST NOT be stricter than the system’s demonstrated operational capability.

---

# 2. Classification System

The doctrine applies to all production software. Rigor increases with criticality and scale.

## 2.1 Criticality Classes

| Class                        | Consequence of Failure                                                                         | Examples                                            | Minimum Controls                                                                 |
| ---------------------------- | ---------------------------------------------------------------------------------------------- | --------------------------------------------------- | -------------------------------------------------------------------------------- |
| **C0 Experimental**          | No production users, no production data, no durable dependency                                 | throwaway prototype                                 | isolated environment, no production credentials                                  |
| **C1 Low Impact**            | Minor internal inconvenience                                                                   | internal dashboard                                  | tests, owner, backup plan                                                        |
| **C2 Business Critical**     | Customer-visible or business-impacting failure                                                 | billing UI, customer API                            | SLO, runbook, staged deploy, observability                                       |
| **C3 High Criticality**      | Major financial, safety, legal, privacy, or availability impact                                | payments, identity, medical workflow, control plane | formal design review, incident drills, DR, security review                       |
| **C4 Civil/Safety Critical** | Loss may threaten life, civil infrastructure, large-scale public trust, or irreversible damage | aviation, medical devices, national infrastructure  | independent assurance, formal verification where feasible, strict change control |

## 2.2 Scale Classes

| Class  | Operational Scale                                   |
| ------ | --------------------------------------------------- |
| **S0** | local-only or single-user                           |
| **S1** | small team or internal use                          |
| **S2** | production application with external users          |
| **S3** | distributed multi-service system                    |
| **S4** | multi-region, high-volume, high-availability system |
| **S5** | civilization-scale or infrastructure-scale system   |

## 2.3 Control Escalation

Rules apply as follows:

| System Class | Required Discipline                                                                    |
| ------------ | -------------------------------------------------------------------------------------- |
| C0/S0        | Isolation, no production coupling, deletion plan                                       |
| C1/S1        | Basic tests, code review, owner, documentation                                         |
| C2/S2        | Full doctrine baseline                                                                 |
| C3/S3+       | Formal architecture review, SLOs, incident drills, threat modeling, DR tests           |
| C4/S4+       | Independent verification, change boards, formal hazard analysis, stricter auditability |

No component may be downgraded in classification to avoid controls.

---

# 3. Core Axioms

These axioms override convenience, preference, and trend.

## Axiom 1: Software is a socio-technical system

Code, people, tools, processes, dependencies, and operations are one system. Failure in any part can become software failure.

## Axiom 2: Production is adversarial

Production contains bad inputs, partial failures, clock drift, exhausted resources, compromised credentials, hostile users, broken dependencies, slow networks, corrupt data, and human mistakes.

## Axiom 3: Failure is normal

The doctrine assumes failure will occur. Systems must be designed for detection, isolation, diagnosis, recovery, and repair.

## Axiom 4: Correctness precedes speed

Fast incorrect software is a liability. Velocity that creates hidden defects is negative progress.

## Axiom 5: Simplicity is a reliability feature

Unnecessary abstraction, distribution, configuration, dependency, and state are reliability risks.

## Axiom 6: State is dangerous

State must have ownership, invariants, migration rules, backup strategy, corruption detection, and repair procedures.

## Axiom 7: Interfaces outlive implementations

Internal code may change quickly. Interfaces require compatibility discipline because consumers accumulate over time.

## Axiom 8: Observability is mandatory

A system that cannot explain its behavior is not production-ready.

## Axiom 9: Security is part of correctness

A system that behaves correctly only for honest actors is not correct.

## Axiom 10: Maintainability is a survival requirement

Teams change. Memory decays. A system that only current authors understand is already failing.

---

# 4. Non-Negotiable Rules

The following rules are mandatory for all C1+ production systems.

## NN-1: Every production component MUST have an owner

**Why:** Unowned software decays.
**Prevents:** orphaned systems, unresolved incidents, unpatched vulnerabilities.
**Tradeoff:** creates explicit responsibility.
**Exception:** none for production.
**Verification:** ownership registry.
**Detection:** monthly ownership audit.
**Correction:** assign owner or retire component.

## NN-2: No production change may bypass review and traceability

**Why:** Changes must be attributable and reversible.
**Prevents:** mystery regressions, unauthorized changes, audit failure.
**Tradeoff:** slower emergency action.
**Exception:** emergency fix with retroactive review within one business day.
**Verification:** change record linked to commit, build, deploy, and approver.
**Detection:** audit log comparison.
**Correction:** freeze pipeline until traceability is restored.

## NN-3: No service may exist without telemetry

**Why:** Operations require evidence.
**Prevents:** blind debugging, false confidence, slow incident response.
**Tradeoff:** instrumentation effort and telemetry cost.
**Exception:** isolated C0 prototypes only.
**Verification:** logs, metrics, traces, health checks, dashboard, alert rules.
**Detection:** production readiness review.
**Correction:** block deployment or add instrumentation.

## NN-4: No critical state may exist without backup, restore, and corruption strategy

**Why:** Data loss and corruption are existential failures.
**Prevents:** unrecoverable loss, silent corruption, destructive migrations.
**Tradeoff:** storage and operational complexity.
**Exception:** explicitly disposable state labeled as such.
**Verification:** restore drill evidence.
**Detection:** backup audit and corruption checks.
**Correction:** halt risky releases until recovery plan exists.

## NN-5: No external dependency may be introduced without approval

**Why:** Dependencies import risk, maintenance obligations, licenses, and attack surface.
**Prevents:** supply-chain compromise, abandonment, version lock, legal exposure.
**Tradeoff:** slower adoption.
**Exception:** emergency security patch with post-approval within five business days.
**Verification:** dependency registry and approval record.
**Detection:** dependency scanner.
**Correction:** approve, replace, vendor, or remove.

## NN-6: No breaking interface change may be released without migration path

**Why:** Consumers accumulate and cannot all move instantly.
**Prevents:** cascading outages, client failures, data incompatibility.
**Tradeoff:** temporary support of old and new behavior.
**Exception:** active exploitation or legal requirement; requires incident-level communication.
**Verification:** compatibility tests and deprecation record.
**Detection:** contract test failures and consumer monitoring.
**Correction:** rollback, compatibility shim, or emergency migration.

## NN-7: Secrets MUST NOT appear in source code, logs, tickets, documentation, or telemetry

**Why:** Secret exposure converts bugs into compromise.
**Prevents:** credential theft, privilege escalation, persistence.
**Tradeoff:** requires secret-management tooling.
**Exception:** none.
**Verification:** secret scanning.
**Detection:** repository, log, and artifact scans.
**Correction:** revoke, rotate, investigate, and purge.

## NN-8: Every retry MUST be bounded, delayed, and safe

**Why:** Retries amplify failure.
**Prevents:** retry storms, duplicate side effects, overload collapse.
**Tradeoff:** some requests fail sooner.
**Exception:** none for network or external calls.
**Verification:** code review and policy linting.
**Detection:** metrics showing retry rates and amplification.
**Correction:** add timeout, backoff, jitter, idempotency, or remove retry.

## NN-9: Every production release MUST be rollback-capable or explicitly roll-forward-only

**Why:** Bad releases are inevitable.
**Prevents:** extended outages.
**Tradeoff:** constrains schema and deployment design.
**Exception:** irreversible migrations require formal roll-forward plan and approval.
**Verification:** release checklist.
**Detection:** deployment audit.
**Correction:** create rollback artifact, compatibility bridge, or roll-forward script.

## NN-10: Documentation required for operation is part of the system

**Why:** Undocumented systems fail during staff turnover and incidents.
**Prevents:** tribal knowledge, slow recovery, unsafe changes.
**Tradeoff:** writing effort.
**Exception:** C0 only.
**Verification:** runbook and architecture-doc review.
**Detection:** incident review and onboarding audit.
**Correction:** block release until required docs exist.

---

# 5. Tradeoff Hierarchy

When rules conflict, decisions MUST follow this priority order:

1. human safety
2. legal and regulatory obligation
3. security of users and systems
4. data integrity
5. correctness
6. recoverability
7. availability
8. operability and debuggability
9. maintainability
10. compatibility
11. performance
12. delivery speed
13. cost reduction
14. developer convenience
15. novelty or preference

## Conflict Rule

A lower-priority goal may not override a higher-priority goal unless:

1. the higher-priority risk is explicitly identified
2. impact is bounded
3. accountable owner approves
4. compensating control exists
5. expiry or review date is set

Example: availability may not override data integrity. If a payment system is uncertain whether a charge succeeded, it MUST preserve ambiguity and reconcile later rather than invent success or failure.

---

# 6. Decision Framework

All significant engineering decisions MUST use the same decision frame.

## 6.1 Required Decision Record

Every architecture, dependency, major refactor, migration, public API, persistence, security, or reliability decision MUST have a decision record containing:

1. decision title
2. status: proposed, accepted, rejected, superseded
3. owner
4. date
5. context
6. constraints
7. alternatives considered
8. decision
9. expected benefits
10. failure modes
11. tradeoffs
12. observability impact
13. security impact
14. data impact
15. rollback or exit strategy
16. review date

## 6.2 Decision Tree: New Capability

```text
Is the capability required by a stated product, operational, legal, or safety need?
  No -> Do not build.
  Yes ->
    Can existing maintained code satisfy it without hidden coupling?
      Yes -> Reuse.
      No ->
        Can it be added inside an existing module without weakening boundaries?
          Yes -> Add inside module.
          No ->
            Is a new module sufficient?
              Yes -> Create module.
              No ->
                Is independent deployment, scaling, ownership, or failure isolation required now?
                  Yes -> Consider service extraction.
                  No -> Use modular monolith boundary.
```

## 6.3 Decision Tree: Add External Dependency

```text
Is the dependency necessary?
  No -> Reject.
  Yes ->
    Is equivalent functionality small enough to implement safely in-house?
      Yes -> Build in-house if lower lifecycle risk.
      No ->
        Is dependency actively maintained, licensed, secure, and compatible?
          No -> Reject.
          Yes ->
            Can it be isolated behind an internal interface?
              No -> Reject or create adapter.
              Yes ->
                Does it pass security, license, operational, and upgrade review?
                  Yes -> Approve with owner.
                  No -> Reject.
```

## 6.4 Decision Tree: Release

```text
Does build come from approved source and reproducible pipeline?
  No -> Block.
Do tests pass without quarantined critical failures?
  No -> Block.
Are migrations backward-compatible?
  No -> Block unless approved irreversible release.
Are dashboards, alerts, and rollback ready?
  No -> Block.
Is error budget healthy for impacted services?
  No -> Reliability owner approval required.
Can release be staged or canaried?
  Yes -> Stage.
  No -> Formal risk acceptance required.
```

---

# 7. Foundational Philosophy

## 7.1 Engineering Values

The organization MUST value:

1. truth over optimism
2. evidence over opinion
3. explicitness over cleverness
4. boring reliability over fashionable novelty
5. repairability over perfection theater
6. ownership over heroics
7. interfaces over implementations
8. invariants over incidental behavior
9. operational reality over design intent
10. long-term stewardship over short-term output

## 7.2 Reliability Philosophy

Reliability is not added after development. Reliability is designed into:

1. requirements
2. architecture
3. interfaces
4. state models
5. code
6. tests
7. telemetry
8. deployment
9. operations
10. maintenance

A feature is incomplete until it can be operated, observed, debugged, recovered, and retired.

## 7.3 Quality Philosophy

Quality means the system reliably does the correct thing under expected, unexpected, degraded, and adversarial conditions.

Quality is not measured by code beauty alone.

Required quality evidence includes:

1. passing tests
2. clear invariants
3. bounded failure modes
4. observability
5. review history
6. operational readiness
7. compatibility proof
8. security review where applicable
9. performance evidence where applicable

## 7.4 Complexity Philosophy

Complexity is a liability unless it buys a documented benefit.

Complexity has forms:

1. code complexity
2. state complexity
3. dependency complexity
4. operational complexity
5. cognitive complexity
6. organizational complexity
7. temporal complexity
8. distributed-systems complexity

The doctrine prefers complexity that is local, visible, testable, and reversible over complexity that is distributed, implicit, stateful, or irreversible.

## 7.5 Engineering Ethics

Engineers MUST NOT knowingly ship systems that:

1. hide material risk from users or operators
2. mishandle private data
3. lack recovery for critical state
4. misrepresent reliability
5. create unsafe operational burdens
6. depend on undocumented heroics
7. conceal known defects that affect safety, security, legality, or integrity

## 7.6 Acceptable Tradeoffs

The following tradeoffs are acceptable when documented:

| Tradeoff                                         | Acceptable When                             |
| ------------------------------------------------ | ------------------------------------------- |
| Slower release for safer deployment              | Any C2+ system                              |
| Higher cost for better recovery                  | State is critical                           |
| Reduced feature scope for correctness            | Deadline threatens quality                  |
| Lower availability to preserve integrity         | Conflicting writes or corruption risk exist |
| Simpler architecture over peak scalability       | Current scale does not justify distribution |
| Manual process before automation                 | Automation would encode unstable behavior   |
| Temporary duplication over premature abstraction | Domain has not stabilized                   |

---

# 8. Failure Philosophy

## 8.1 Failure Assumptions

Every production design MUST assume:

1. dependencies fail
2. networks partition
3. clocks drift
4. deployments break
5. humans make mistakes
6. credentials leak
7. data corrupts
8. traffic spikes
9. queues backlog
10. retries amplify load
11. monitoring has gaps
12. documentation becomes stale
13. requirements change
14. teams turn over

## 8.2 Failure Handling Priorities

When failure occurs:

1. preserve human safety
2. prevent further corruption
3. contain blast radius
4. restore critical service
5. preserve forensic evidence
6. communicate status
7. repair root causes
8. update doctrine, tests, docs, and automation

## 8.3 Failure Mode Register

Every C2+ component MUST maintain a failure mode register containing:

1. failure mode
2. triggering conditions
3. expected symptoms
4. telemetry signals
5. blast radius
6. immediate mitigation
7. permanent fix class
8. owner
9. last validation date

Example:

| Failure Mode             | Signal                          | Mitigation                   | Permanent Fix                               |
| ------------------------ | ------------------------------- | ---------------------------- | ------------------------------------------- |
| Payment provider timeout | provider latency p95 > budget   | stop retries, queue captures | provider circuit breaker and reconciliation |
| Cache stampede           | DB QPS spike after cache expiry | enable request coalescing    | jittered TTL and single-flight locking      |
| Schema migration lock    | DB lock wait rising             | abort migration              | online migration tooling                    |

---

# 9. Software Entropy Management

## 9.1 Entropy Definition

Software entropy is the gradual increase of disorder, coupling, ambiguity, stale assumptions, obsolete dependencies, undocumented behavior, and operational fragility.

Entropy is inevitable. Unmanaged entropy becomes failure.

## 9.2 Entropy Budget

Each team MUST reserve capacity for entropy reduction.

Minimum allocation:

| System Class | Required Maintenance Capacity           |
| ------------ | --------------------------------------- |
| C1           | 5% of engineering time                  |
| C2           | 10%                                     |
| C3           | 15%                                     |
| C4           | 20% or more, set by assurance authority |

This capacity cannot be reallocated to feature work without owner approval and risk record.

## 9.3 Entropy Signals

Teams MUST track:

1. stale dependencies
2. failing or flaky tests
3. unowned code
4. rising incident count
5. rising change failure rate
6. slow builds
7. code complexity hotspots
8. outdated docs
9. long-lived feature flags
10. unsupported APIs
11. repeated manual operational actions
12. recurring postmortem themes

## 9.4 Entropy Correction

Entropy must be corrected by:

1. deletion
2. simplification
3. boundary repair
4. dependency reduction
5. documentation update
6. test improvement
7. automation
8. migration
9. retirement

Adding abstraction is not the default entropy fix. Abstraction is permitted only when it reduces total lifecycle complexity.

---

# 10. Human Factors and Organizational Decay

## 10.1 Human Error Model

Humans are fallible. Processes MUST reduce the probability and impact of human mistakes.

Required controls:

1. checklists for risky operations
2. peer review for production changes
3. automation for repeated operations
4. clear ownership
5. incident drills
6. readable code
7. operational runbooks
8. safe defaults
9. least privilege
10. fatigue-aware on-call

## 10.2 Organizational Decay Signals

Organizations decay when:

1. ownership becomes unclear
2. exceptions become permanent
3. reviews become ceremonial
4. postmortems stop producing changes
5. documentation is known to be stale
6. on-call becomes heroic
7. architecture decisions are made by trend
8. teams optimize local goals over system reliability
9. leaders reward speed while ignoring failure cost
10. hidden dependencies accumulate

## 10.3 Required Organizational Countermeasures

Every quarter, engineering leadership MUST review:

1. incidents and recurring causes
2. exception register
3. ownership gaps
4. critical dependency health
5. SLO performance
6. security posture
7. technical debt inventory
8. staffing risks
9. documentation freshness
10. operational toil

---

# 11. System Design Principles

## 11.1 Coupling Rules

### Rule SD-1: Minimize directional coupling

A component MUST depend only on components whose stability and ownership are compatible with its lifecycle.

**Why:** Unstable dependencies destabilize consumers.
**Prevents:** cascading change, brittle architecture.
**Tradeoff:** requires adapter layers.
**Exception:** temporary migration bridge with expiry.
**Verification:** dependency graph review.
**Detection:** build graph and runtime dependency analysis.
**Correction:** introduce boundary, adapter, or dependency inversion.

### Binding Rules

1. High-level policy MUST NOT depend on low-level implementation details.
2. Shared libraries MUST NOT call application services.
3. Domain logic MUST NOT depend on transport, database, UI, or framework code.
4. Infrastructure code MAY depend on domain interfaces, not domain internals.
5. Cross-team dependencies MUST use documented interfaces.

## 11.2 Cohesion Rules

### Rule SD-2: A module MUST have one primary reason to change

**Why:** Cohesion makes change local.
**Prevents:** shotgun changes, accidental regressions.
**Tradeoff:** may create more modules.
**Exception:** small C1 utilities may combine related concerns until growth triggers split.
**Verification:** module responsibility statement.
**Detection:** repeated multi-module changes or unrelated edits in same module.
**Correction:** split by responsibility.

## 11.3 Module Boundary Rules

A module boundary is valid only when it defines:

1. responsibility
2. public interface
3. hidden internals
4. ownership
5. dependency rules
6. test strategy
7. error contract
8. versioning expectation if externally consumed

A boundary is invalid if callers must know internal storage, timing, locking, retry, or implementation details.

## 11.4 State Management Rules

### Rule SD-3: Every durable state store MUST have one owning component

**Why:** Shared ownership causes conflicting invariants.
**Prevents:** data races, schema chaos, corruption.
**Tradeoff:** may require APIs instead of direct access.
**Exception:** read-only analytical replicas with explicit contract.
**Verification:** data ownership registry.
**Detection:** unauthorized direct writes.
**Correction:** revoke access, route writes through owner.

State rules:

1. Writes MUST go through the state owner.
2. Readers MUST tolerate compatible schema evolution.
3. State invariants MUST be enforced as close to storage as feasible.
4. Cached state MUST declare freshness, invalidation, and fallback behavior.
5. Replicated state MUST declare conflict strategy.
6. Derived state MUST declare source of truth and rebuild procedure.
7. Temporary state MUST have expiry.
8. Orphaned state MUST be deleted or assigned an owner.

## 11.5 Distributed System Principles

Distributed systems MUST assume:

1. messages may be delayed, duplicated, reordered, or lost
2. nodes may fail independently
3. network partitions occur
4. clocks are unreliable for ordering
5. retries may duplicate side effects
6. consensus is expensive
7. global transactions are fragile
8. observability must cross boundaries

Rules:

1. Distributed writes MUST be idempotent or protected by transaction semantics.
2. Cross-service workflows MUST have compensation or reconciliation.
3. Services MUST NOT rely on exactly-once delivery unless the mechanism is formally specified and tested.
4. Message consumers MUST tolerate duplicates.
5. Event ordering MUST NOT be assumed unless guaranteed by partition key and documented.
6. Time-based ordering MUST use monotonic or logical ordering where correctness depends on order.

## 11.6 Failure Isolation

### Rule SD-4: Failure domains MUST be explicit

**Why:** Unknown blast radius converts small faults into outages.
**Prevents:** cascading failure.
**Tradeoff:** isolation can increase cost and complexity.
**Exception:** C1 systems may use shared infrastructure if documented.
**Verification:** architecture review.
**Detection:** incident blast-radius analysis.
**Correction:** add bulkheads, queues, rate limits, circuit breakers, or partitioning.

## 11.7 Deterministic Behavior

Code MUST be deterministic where correctness, testing, replay, or audit requires determinism.

Rules:

1. Inject clocks, randomness, and external services into testable logic.
2. Pure domain calculations MUST NOT read wall-clock time directly.
3. Serialization MUST use stable field ordering where used for signatures, hashes, or snapshots.
4. Tests MUST control time, random seeds, and external responses.

## 11.8 Dependency Direction

Allowed dependency direction:

```text
UI / Transport / CLI
        ↓
Application / Use Cases
        ↓
Domain Logic
        ↓
Domain Interfaces
        ↑
Infrastructure Implementations
```

Infrastructure implementations satisfy domain interfaces. Domain logic does not import infrastructure implementations.

## 11.9 Synchronization Rules

1. Shared mutable state MUST be minimized.
2. Locks MUST have bounded scope.
3. Lock acquisition order MUST be documented for multi-lock code.
4. Blocking calls MUST NOT occur while holding locks unless proven safe.
5. Deadlock-sensitive code MUST have tests or analysis.
6. Distributed locks MUST have lease expiry, fencing tokens, and failure behavior.

## 11.10 Eventing Rules

Events MUST represent facts that happened, not commands disguised as facts.

Good event: `InvoicePaid`
Bad event: `SendInvoiceEmailNow`

Event rules:

1. Event names MUST be past tense for facts.
2. Event schema MUST be versioned.
3. Event consumers MUST ignore unknown fields.
4. Event producers MUST NOT remove fields without deprecation.
5. Event replay behavior MUST be documented.
6. Event retention MUST match recovery and audit requirements.
7. Event handlers MUST be idempotent.
8. Event-driven workflows MUST include reconciliation.

## 11.11 Concurrency Models

Each component MUST declare its concurrency model:

1. single-threaded event loop
2. thread pool
3. actor model
4. async tasks
5. worker queue
6. process-per-request
7. distributed workers
8. hybrid

Undeclared concurrency is forbidden for C2+ systems.

---

# 12. Architecture Doctrine

## 12.1 Default Architecture

The default architecture for new systems is:

```text
modular monolith first,
service extraction only when justified,
microservices only when operational maturity exists.
```

## 12.2 Monolith vs Modular Monolith vs Microservices

| Architecture     | Use When                                                                                                   | Avoid When                                            |
| ---------------- | ---------------------------------------------------------------------------------------------------------- | ----------------------------------------------------- |
| Monolith         | small team, low domain complexity, single deployable acceptable                                            | boundaries are unclear and growth is expected         |
| Modular monolith | most C1-C3 systems, domain boundaries exist, operational simplicity matters                                | independent scaling or deployment is mandatory now    |
| Microservices    | independent ownership, scaling, deployment, or fault isolation is required and operational maturity exists | used for fashion, team politics, or premature scaling |

### Rule AR-1: Microservices require evidence

A new service MUST NOT be created unless at least one is true:

1. independent scaling is required now
2. independent deployment is required now
3. failure isolation is required now
4. data ownership boundary is strong and stable
5. separate team ownership is necessary and sustainable
6. compliance or security boundary requires separation

And all are true:

1. owner exists
2. on-call exists
3. telemetry exists
4. runbook exists
5. SLO exists where relevant
6. deployment and rollback exist
7. API contract exists
8. data ownership exists

## 12.3 Service Boundary Criteria

A valid service boundary has:

1. cohesive business capability
2. independent state ownership
3. stable API
4. clear failure behavior
5. clear owner
6. operational readiness
7. security boundary
8. compatibility plan

Invalid service boundaries include:

1. one service per database table
2. one service per developer preference
3. service created only to use a framework
4. service whose consumers require internal knowledge
5. service with shared writable database
6. service without on-call owner

## 12.4 API Evolution

### Rule AR-2: APIs MUST evolve additively first

**Why:** Consumers cannot all migrate instantly.
**Prevents:** breaking changes and outages.
**Tradeoff:** old fields and behavior persist temporarily.
**Exception:** security emergency or legal mandate.
**Verification:** contract tests.
**Detection:** consumer errors after deploy.
**Correction:** restore compatibility or rollback.

API rules:

1. Add fields before requiring fields.
2. Add endpoints before removing endpoints.
3. New fields MUST be optional until consumers migrate.
4. Unknown fields MUST be ignored by consumers.
5. Field meaning MUST NOT change silently.
6. Error codes MUST remain stable.
7. Pagination, filtering, and sorting semantics MUST be explicit.
8. Idempotency keys MUST be supported for externally retried mutation APIs.
9. APIs MUST specify timeout expectations.
10. APIs MUST document authentication, authorization, rate limits, and error behavior.

## 12.5 Schema Evolution

Schema changes MUST follow expand-migrate-contract:

```text
1. Expand: add new schema elements compatibly.
2. Migrate: move data and code to new shape.
3. Verify: prove all consumers use new shape.
4. Contract: remove old schema only after deprecation window.
```

Rules:

1. Destructive schema changes MUST NOT be deployed with code that still depends on old schema.
2. Renames MUST be treated as add-copy-deprecate-delete.
3. Backfills MUST be resumable and rate-limited.
4. Online migrations MUST avoid long exclusive locks.
5. Migration scripts MUST be reviewed and tested on production-like data.
6. Every migration MUST define rollback or roll-forward behavior.
7. Irreversible migrations require formal approval.

## 12.6 Versioning Rules

1. Internal modules MAY use repository versioning.
2. Public APIs MUST have explicit versioning strategy.
3. Semantic versioning MAY be used for libraries only when compatibility rules are enforced.
4. Date-based versions MAY be used for APIs when deprecation windows are explicit.
5. Version numbers MUST NOT be used to hide breaking changes.
6. Multiple active versions MUST have owners and retirement dates.

## 12.7 Reliability-First Architecture

Architectures MUST include:

1. failure domains
2. timeout policy
3. retry policy
4. load-shedding policy
5. queue behavior
6. data recovery
7. observability
8. incident procedures
9. deployment safety
10. dependency degradation

## 12.8 Recovery-First Design

Every C2+ system MUST answer:

1. How is a bad release rolled back?
2. How is corrupted data detected?
3. How is corrupted data repaired?
4. How is lost state restored?
5. How are failed workflows reconciled?
6. How is user impact measured?
7. How is dependency failure survived?
8. How is partial completion handled?

If these questions are unanswered, design is incomplete.

---

# 13. Code-Level Standards

## 13.1 Code Readability Doctrine

Code is written for future maintainers first and machines second.

Readable code has:

1. explicit names
2. simple control flow
3. local reasoning
4. small functions
5. clear error handling
6. limited side effects
7. visible invariants
8. minimal cleverness
9. tests that document behavior

## 13.2 Naming Conventions

### Universal Semantic Naming Rules

1. Names MUST describe purpose, not type alone.
2. Avoid abbreviations except approved domain terms.
3. Boolean names MUST start with `is`, `has`, `can`, `should`, or `was`.
4. Collection names MUST be plural.
5. Units MUST be encoded in names where ambiguity exists.
6. Time values MUST include unit or semantic form.
7. Currency values MUST include unit and precision.
8. Identifiers MUST include domain entity.
9. Temporary variables MAY be short only in tiny local scopes.
10. Names MUST NOT encode obsolete implementation details.

Examples:

| Bad         | Good                         |
| ----------- | ---------------------------- |
| `data`      | `invoiceRecords`             |
| `amount`    | `amountCents`                |
| `timeout`   | `timeoutMillis`              |
| `flag`      | `isPaymentRetryEnabled`      |
| `list`      | `activeUserIds`              |
| `process()` | `captureAuthorizedPayment()` |

### Casing Rules

| Artifact                | Default Convention                                      |
| ----------------------- | ------------------------------------------------------- |
| Types/classes           | `PascalCase`                                            |
| Functions/methods       | language standard; otherwise `lowerCamelCase`           |
| Variables               | language standard; otherwise `lowerCamelCase`           |
| Constants               | `UPPER_SNAKE_CASE`                                      |
| Database tables/columns | `snake_case`                                            |
| File names              | `kebab-case` unless language requires otherwise         |
| Environment variables   | `UPPER_SNAKE_CASE`                                      |
| Event names             | `PascalCase` past tense                                 |
| API JSON fields         | `camelCase` unless external standard requires otherwise |

Go projects MUST use Go's conventional casing, package naming, file naming, formatting, and documentation style. `gofmt` is mandatory.

## 13.3 File Organization

Each repository MUST define:

1. source directory
2. test directory
3. documentation directory
4. build configuration
5. dependency manifest
6. operational files
7. migration files
8. generated-code location
9. scripts location
10. ownership metadata

Default structure:

```text
repo/
  docs/
    architecture/
    decisions/
    runbooks/
    api/
  src/
    <domain-or-component>/
  tests/
    unit/
    integration/
    contract/
    e2e/
  migrations/
  scripts/
  config/
  deploy/
  observability/
  SECURITY.md
  README.md
  OWNERS
```

Generated code MUST be isolated and marked. Manual edits to generated code are forbidden.

## 13.4 Function Sizing

### Rule CS-1: Functions MUST be small enough to reason about locally

Default limits:

| Measure               | Limit                                            |
| --------------------- | ------------------------------------------------ |
| Logical lines         | ≤ 40 preferred, > 60 requires justification      |
| Parameters            | ≤ 5 preferred, > 7 requires parameter object     |
| Cyclomatic complexity | ≤ 10 preferred, > 15 requires refactor or waiver |
| Nested control depth  | ≤ 3 preferred                                    |
| Side-effect domains   | ≤ 1 per function unless orchestration function   |

Exceptions:

1. generated code
2. simple declarative mapping
3. performance-critical code with comments and tests
4. unavoidable protocol implementation

Violation correction:

1. extract pure logic
2. split orchestration from calculation
3. introduce named intermediate concepts
4. reduce nesting with guard clauses
5. replace mode flags with separate functions

## 13.5 Class and Module Sizing

| Artifact                 | Limit                                                              |
| ------------------------ | ------------------------------------------------------------------ |
| Class                    | ≤ 300 logical lines preferred                                      |
| Public methods per class | ≤ 12 preferred                                                     |
| Module file              | ≤ 500 logical lines preferred                                      |
| Package/module           | clear responsibility; no fixed line limit if internally structured |

Large classes MUST be reviewed for responsibility mixing.

## 13.6 Abstraction Rules

### Rule CS-2: Abstractions MUST pay rent

An abstraction is allowed only when at least one is true:

1. it protects a stable domain concept
2. it isolates external dependency
3. it hides unavoidable complexity
4. it enables testing across a boundary
5. it removes repeated behavior with stable semantics
6. it enforces invariant

An abstraction is forbidden when:

1. created for hypothetical future need
2. merely renames one function
3. hides simple code behind indirection
4. makes debugging harder without compensating value
5. couples unrelated concepts
6. requires readers to jump through many files for simple behavior

## 13.7 Inheritance and Composition

Rules:

1. Prefer composition over inheritance.
2. Inheritance MAY be used for stable type hierarchies with substitutability.
3. Inheritance MUST NOT be used for code sharing alone.
4. Base classes MUST NOT depend on subclass implementation details.
5. Deep inheritance chains greater than two levels require design review.
6. Mixins/traits MUST be stateless or explicitly documented.

## 13.8 Side-Effect Rules

Functions MUST be classified as:

1. pure calculation
2. state read
3. state write
4. external call
5. orchestration

A function that writes state or calls external systems MUST reveal that through name, type, or module location.

Example:

```text
calculateInvoiceTotal()          -> pure
loadInvoiceById()                -> state read
saveInvoiceStatus()              -> state write
sendInvoiceEmail()               -> external side effect
capturePaymentWorkflow()         -> orchestration
```

## 13.9 Immutability Rules

1. Prefer immutable values for domain data.
2. Mutable shared state MUST be protected by synchronization or confinement.
3. Public APIs MUST NOT expose mutable internal collections.
4. Configuration MUST be immutable after startup unless dynamic reload is explicitly designed.
5. Events MUST be immutable facts.

## 13.10 Error Handling Rules

### Rule CS-3: Errors MUST be explicit, typed, and actionable

Error rules:

1. Do not swallow errors silently.
2. Do not convert all errors into generic strings.
3. Preserve causal chain.
4. Distinguish validation, authorization, dependency, timeout, conflict, corruption, and internal errors.
5. Include stable error codes for public APIs.
6. Log errors once at ownership boundary.
7. Do not log secrets or excessive personal data.
8. Do not use exceptions for normal control flow in hot paths.
9. Retryable errors MUST be marked retryable by policy, not guesswork.
10. Fatal errors MUST fail fast with diagnostic context.

Bad:

```text
try:
    chargeCard()
except:
    return true
```

Good:

```text
try:
    captureAuthorizedPayment(paymentIntentId, idempotencyKey)
except ProviderTimeout as error:
    recordPaymentAttemptPending(paymentIntentId, error.correlationId)
    raise RetryableDependencyError("payment_provider_timeout", cause=error)
except ProviderDeclined as error:
    recordPaymentDeclined(paymentIntentId, error.reasonCode)
    raise NonRetryableBusinessError("payment_declined", cause=error)
```

## 13.11 Logging Rules

1. Logs MUST be structured.
2. Logs MUST include correlation or trace ID where request-scoped.
3. Logs MUST include component, environment, version, and operation.
4. Logs MUST NOT include secrets.
5. Personal data MUST be minimized and classified.
6. Log levels MUST be consistent.
7. Errors MUST include cause, operation, and impact.
8. High-cardinality fields MUST be controlled.
9. Logs consumed by automation are interfaces and require compatibility.
10. Debug logging in production MUST be time-bounded or dynamically controlled.

Log levels:

| Level    | Meaning                                                       |
| -------- | ------------------------------------------------------------- |
| DEBUG    | diagnostic detail, normally disabled                          |
| INFO     | meaningful lifecycle or business event                        |
| WARN     | unexpected but handled condition                              |
| ERROR    | failed operation requiring attention or correlated incident   |
| CRITICAL | system integrity, security, or availability at immediate risk |

## 13.12 Serialization Rules

1. Serialized formats are interfaces.
2. Schemas MUST be documented.
3. Unknown fields MUST be tolerated where compatibility requires.
4. Required fields MUST be stable.
5. Field meanings MUST NOT change without versioning.
6. Time MUST use explicit timezone or epoch format.
7. Numeric precision MUST be explicit.
8. Binary formats MUST have schema evolution strategy.
9. Deserialization MUST validate untrusted input.
10. Serialization used for signatures MUST be canonical.

## 13.13 Null Handling

1. Nullability MUST be explicit in types or documentation.
2. Public APIs MUST distinguish absent, unknown, empty, and zero.
3. Null MUST NOT be used as a generic error signal.
4. Optional values MUST be checked at boundaries.
5. Collections SHOULD be empty rather than null.
6. Database nullable columns MUST have rationale.

## 13.14 Defensive Programming

Defensive programming is required at trust boundaries and invariant boundaries.

Rules:

1. Validate external inputs.
2. Assert internal invariants.
3. Fail fast on impossible states.
4. Use allowlists for constrained values.
5. Bound memory, time, and input size.
6. Treat configuration as untrusted input.
7. Treat dependency responses as untrusted input.
8. Validate migrations before applying them.

---

# 14. Testing Doctrine

## 14.1 Testing Philosophy

Testing is evidence, not ritual.

A test suite MUST prove:

1. intended behavior works
2. invalid behavior is rejected
3. invariants hold
4. interfaces remain compatible
5. failures are handled
6. security controls work
7. performance budgets are not violated
8. migrations are safe
9. observability exists for critical paths

## 14.2 Test Ownership

Every test has the same owner as the behavior it protects.

Rules:

1. Tests MUST be maintained with production code.
2. Failing tests MUST block merge unless quarantined by policy.
3. Test deletion requires justification.
4. Flaky tests are defects.
5. Test-only behavior MUST NOT alter production semantics.

## 14.3 Unit Testing

Unit tests MUST cover:

1. pure domain logic
2. boundary validation
3. error classification
4. invariants
5. edge cases
6. deterministic calculations
7. state transitions

Unit tests MUST NOT depend on real networks, real external services, wall-clock time, random seeds, or test order.

## 14.4 Integration Testing

Integration tests MUST cover:

1. database interactions
2. message queues
3. external provider adapters using test doubles or sandboxes
4. serialization and deserialization
5. migrations
6. configuration loading
7. authentication and authorization integration
8. observability emission for critical flows

## 14.5 Contract Testing

Contract tests are mandatory for C2+ public APIs and cross-team service interfaces.

They MUST prove:

1. provider behavior matches published contract
2. consumers tolerate compatible changes
3. error responses remain stable
4. required fields are preserved
5. deprecated fields remain until retirement date

## 14.6 Property Testing

Property testing SHOULD be used for:

1. parsers
2. serializers
3. validators
4. financial calculations
5. state machines
6. permission logic
7. retry/backoff calculations
8. data transformations
9. reconciliation algorithms

Example property:

```text
For every valid invoice:
  deserialize(serialize(invoice)) == invoice
```

## 14.7 Snapshot Testing

Snapshot tests are allowed only when:

1. output is intentionally stable
2. snapshot review is meaningful
3. snapshot size is manageable
4. updates require human review
5. behavior is not better tested semantically

Snapshot tests MUST NOT be used to approve large unread diffs blindly.

## 14.8 End-to-End Testing

E2E tests MUST cover critical user journeys but MUST NOT attempt to cover every branch.

Required E2E coverage for C2+:

1. authentication path
2. primary success path
3. primary failure path
4. permission denial
5. critical integration path
6. rollback or recovery path where feasible

E2E tests must be few, stable, and high-value.

## 14.9 Reliability Testing

C3+ systems MUST test:

1. dependency outage
2. slow dependency
3. partial dependency failure
4. queue backlog
5. retry storm prevention
6. failover
7. rollback
8. backup restore
9. data repair workflow
10. degraded-mode behavior

## 14.10 Chaos Testing

Chaos testing is allowed only when:

1. blast radius is bounded
2. abort mechanism exists
3. hypothesis is written
4. telemetry is ready
5. owner is present
6. customer impact is acceptable
7. rollback exists

C3/S3+ systems SHOULD run controlled failure-injection exercises at least quarterly.

## 14.11 Fuzz Testing

Fuzz testing SHOULD be used for:

1. parsers
2. protocol handlers
3. file processors
4. compression/decompression
5. cryptographic input handling
6. public API input validation
7. unsafe memory-language components

## 14.12 Mutation Testing

Mutation testing SHOULD be used periodically for critical domain logic to detect ineffective tests.

Minimum cadence:

| Criticality | Cadence                         |
| ----------- | ------------------------------- |
| C1          | optional                        |
| C2          | semiannual for critical modules |
| C3          | quarterly for critical modules  |
| C4          | required by assurance plan      |

## 14.13 Performance Testing

Performance tests MUST exist for code with latency, throughput, memory, or storage budgets.

They MUST define:

1. workload model
2. data size
3. concurrency
4. environment
5. pass/fail budget
6. warm-up behavior
7. regression threshold
8. owner

## 14.14 Coverage Requirements

Coverage is a floor, not proof of quality.

| System Class | Required Coverage                                                                     |
| ------------ | ------------------------------------------------------------------------------------- |
| C1           | meaningful tests for changed behavior                                                 |
| C2           | ≥ 80% business logic line coverage or approved risk-based alternative                 |
| C3           | ≥ 90% critical-path logic coverage plus contract and failure tests                    |
| C4           | assurance-specific structural coverage, hazard coverage, and independent verification |

Coverage may not be increased using meaningless tests.

## 14.15 Flaky Test Handling

A flaky test MUST be handled as follows:

```text
Detect -> classify -> quarantine if blocking unrelated work -> assign owner -> repair -> restore gate
```

Rules:

1. Critical-path flaky tests MUST be fixed immediately.
2. Quarantined tests MUST have expiry.
3. Quarantine beyond 14 days requires engineering leadership approval.
4. Repeated flakes trigger root-cause review.

## 14.16 CI Reliability Rules

1. CI MUST be deterministic.
2. CI MUST fail closed for security, compatibility, and critical tests.
3. CI configuration is production-relevant code and requires review.
4. CI failures MUST be visible.
5. Bypasses MUST be audited.
6. Build artifacts MUST be traceable to source.
7. Test environments MUST be reproducible.

## 14.17 Forbidden Testing Practices

Forbidden:

1. tests that pass by sleeping arbitrary durations without condition checks
2. tests dependent on execution order
3. tests that require real production credentials
4. tests that ignore assertions
5. tests that assert implementation details unnecessarily
6. mass snapshot updates without review
7. deleting failing tests to pass CI
8. mocking the system under test so heavily that behavior is not tested
9. relying only on manual testing for C2+ critical behavior

---

# 15. Debugging and Observability

## 15.1 Observability Doctrine

A production system MUST be able to answer:

1. Is it healthy?
2. What changed?
3. Who is affected?
4. Which dependency is failing?
5. Which version is running?
6. What is the error rate?
7. What is the latency distribution?
8. What is saturated?
9. Which requests are failing?
10. What state changed?
11. Can we correlate logs, traces, metrics, and deploys?

## 15.2 Required Telemetry

Every C2+ service MUST emit:

1. structured logs
2. request metrics
3. dependency metrics
4. saturation metrics
5. error metrics
6. traces for cross-boundary calls
7. health checks
8. build and version metadata
9. audit events for sensitive actions
10. business-critical counters where relevant

## 15.3 Metrics

Required metric categories:

| Category          | Examples                                               |
| ----------------- | ------------------------------------------------------ |
| Request           | rate, errors, duration                                 |
| Dependency        | calls, errors, latency, timeout count                  |
| Saturation        | CPU, memory, disk, queue depth, connection pool        |
| Correctness       | reconciliation mismatch, invariant violation           |
| Business critical | payment captures, failed logins, delayed jobs          |
| Deployment        | version, canary cohort, rollback count                 |
| Security          | auth failures, permission denials, suspicious activity |

Metric cardinality MUST be bounded. User IDs, request IDs, and unbounded strings MUST NOT be metric labels.

## 15.4 Tracing

Tracing rules:

1. Incoming requests MUST create or propagate trace context.
2. Outbound calls MUST propagate trace context.
3. Spans MUST identify operation, dependency, status, and latency.
4. Spans MUST NOT include secrets.
5. Sampling MUST preserve error traces.
6. Critical workflows SHOULD have full tracing during incident windows.

## 15.5 Structured Logging

Required log fields:

1. timestamp
2. service/component
3. environment
4. version
5. operation
6. severity
7. trace ID or correlation ID
8. actor type where applicable
9. resource identifier where safe
10. error code where applicable
11. dependency where applicable

## 15.6 Profiling

C2+ systems MUST support safe profiling in non-production.

C3+ systems SHOULD support controlled production profiling with:

1. access control
2. rate limits
3. time bounds
4. privacy controls
5. audit logs
6. rollback switch

## 15.7 Crash Analysis

Every crash MUST produce:

1. component name
2. version
3. environment
4. stack trace or equivalent
5. recent relevant events
6. resource state
7. correlation ID where available
8. core dump only when privacy controls allow

Crash loops MUST trigger automatic containment.

## 15.8 Reproducibility

Bug reports for C2+ systems MUST include:

1. observed behavior
2. expected behavior
3. version
4. environment
5. input or triggering event
6. relevant logs/traces
7. data state classification
8. reproduction steps where possible
9. impact
10. owner

## 15.9 Replay Systems

Replay is required for C3+ event-driven or financial systems where correctness depends on historical events.

Replay rules:

1. events must be immutable
2. replay must be idempotent
3. replay must be isolated from production side effects unless explicitly approved
4. replay must record version used
5. replay must detect divergence
6. replay must not leak sensitive data

## 15.10 Production Debugging

Production debugging rules:

1. Prefer read-only inspection.
2. No manual data mutation without approved repair procedure.
3. No ad hoc scripts against production without review.
4. All access must be logged.
5. Debug sessions must have owner and purpose.
6. Temporary diagnostic changes must expire.
7. Customer data access must be minimized and authorized.

## 15.11 Failure Forensics

After C2+ incidents, teams MUST preserve:

1. deploy history
2. logs
3. traces
4. metrics
5. alerts
6. configuration changes
7. dependency status
8. operator actions
9. relevant data samples where lawful
10. timeline

---

# 16. Reliability Engineering

## 16.1 SLO Doctrine

Every C2+ user-facing service MUST define SLIs and SLOs.

Required SLO fields:

1. user journey or capability
2. SLI definition
3. measurement window
4. target
5. exclusions
6. data source
7. owner
8. alert policy
9. error-budget policy
10. review cadence

Example:

```text
Capability: checkout payment authorization
SLI: successful authorization responses / valid authorization requests
Window: 28 days
Target: 99.9%
Owner: payments platform
Alert: burn rate > 2% budget/hour
```

## 16.2 Error Budget Policy

When error budget is healthy:

1. normal feature releases may proceed
2. risky releases still require controls

When error budget is burning fast:

1. freeze risky changes
2. prioritize mitigation
3. increase monitoring
4. notify stakeholders

When error budget is exhausted:

1. stop nonessential releases
2. prioritize reliability repair
3. require owner approval for exceptions
4. conduct review

## 16.3 Timeout Rules

### Rule RE-1: Every remote call MUST have a timeout

Defaults:

| Call Type                 | Default Timeout                      |
| ------------------------- | ------------------------------------ |
| interactive internal call | 1–3 seconds                          |
| interactive external call | 2–5 seconds                          |
| background external call  | 10–30 seconds                        |
| batch operation           | explicit per job                     |
| database query            | explicit budget based on query class |

Rules:

1. No infinite timeouts.
2. Caller timeout MUST be shorter than user-facing timeout budget.
3. Timeouts MUST include connection and read phases.
4. Cancellation MUST propagate where supported.
5. Timeout errors MUST be observable.

## 16.4 Retry Rules

Retries are allowed only when:

1. operation is idempotent or protected by idempotency key
2. error is retryable
3. retry count is bounded
4. backoff exists
5. jitter exists
6. total timeout budget is respected
7. retry metrics are emitted

Default retry policy:

| Context                  | Max Attempts                        |
| ------------------------ | ----------------------------------- |
| user request             | 2 total attempts unless proven safe |
| background job           | 3–5 attempts                        |
| scheduled reconciliation | policy-specific                     |
| non-idempotent mutation  | no retry without idempotency key    |

## 16.5 Circuit Breaker Rules

C2+ services calling unreliable dependencies SHOULD use circuit breakers.

Circuit breaker MUST define:

1. open threshold
2. half-open probe behavior
3. fallback behavior
4. metrics
5. owner
6. manual override policy

## 16.6 Idempotency

Mutation operations exposed to clients or retried internally MUST support idempotency when duplicate execution would cause harm.

Idempotency record MUST include:

1. key
2. operation
3. caller
4. request hash where feasible
5. result
6. expiry
7. conflict behavior

## 16.7 Transaction Handling

Rules:

1. Keep transactions short.
2. Do not perform network calls inside database transactions unless explicitly justified.
3. Protect invariants with constraints where feasible.
4. Use optimistic concurrency for collaborative updates where feasible.
5. Use pessimistic locks only with documented order and timeout.
6. Distributed transactions require architecture review.
7. Sagas require compensation and reconciliation.

## 16.8 Recovery Procedures

Every C2+ service MUST have recovery procedures for:

1. bad deploy
2. dependency outage
3. data corruption
4. data loss
5. queue backlog
6. stuck jobs
7. credential leak
8. capacity exhaustion
9. regional failure where applicable
10. failed migration

## 16.9 Rollback Doctrine

Rollback is preferred when:

1. bad code caused failure
2. state remains compatible
3. rollback is faster than fix
4. rollback does not increase corruption risk

Roll-forward is required when:

1. migration is irreversible
2. rollback would corrupt state
3. security patch cannot be removed
4. old version cannot operate safely

## 16.10 Disaster Recovery

C3+ systems MUST define:

1. recovery time objective
2. recovery point objective
3. backup frequency
4. restore procedure
5. failover procedure
6. communication plan
7. dependency assumptions
8. test cadence

Minimum DR test cadence:

| Criticality | Restore Test                          |
| ----------- | ------------------------------------- |
| C1          | annual where stateful                 |
| C2          | semiannual                            |
| C3          | quarterly                             |
| C4          | assurance-defined, at least quarterly |

## 16.11 Corruption Handling

Data corruption procedure MUST include:

1. detection signal
2. stop-the-bleed action
3. scope assessment
4. backup comparison
5. repair method
6. validation method
7. customer impact analysis
8. audit trail
9. post-repair monitoring

## 16.12 Graceful Degradation

C2+ services SHOULD degrade gracefully by:

1. disabling noncritical features
2. serving cached data with freshness label
3. queuing work
4. reducing detail level
5. falling back to alternate provider
6. rate limiting lower-priority traffic
7. protecting critical operations first

Graceful degradation MUST NOT silently violate correctness or security.

---

# 17. Security Doctrine

## 17.1 Security Philosophy

Security is a system property. It is not a final review step.

Every system MUST assume:

1. inputs are hostile
2. credentials may leak
3. dependencies may be compromised
4. insiders can make mistakes
5. logs may expose data
6. users may exceed intended behavior
7. attackers chain minor weaknesses

## 17.2 Trust Boundaries

Every C2+ architecture document MUST identify trust boundaries:

1. user to application
2. service to service
3. application to database
4. application to external provider
5. employee to production system
6. build system to artifact repository
7. network zone to network zone
8. control plane to data plane

At each boundary, define:

1. authentication
2. authorization
3. encryption
4. validation
5. logging
6. rate limiting
7. failure behavior

## 17.3 Authentication

Rules:

1. Authentication MUST be centralized where feasible.
2. Passwords MUST NOT be stored directly.
3. Multi-factor authentication MUST protect privileged access.
4. Service identity MUST use short-lived credentials where feasible.
5. Authentication failures MUST be logged without leaking secrets.
6. Session lifecycle MUST define creation, refresh, expiry, revocation.

## 17.4 Authorization

Rules:

1. Authorization MUST be checked server-side.
2. Authorization MUST be enforced at resource boundary.
3. Deny by default.
4. Privilege escalation paths MUST be reviewed.
5. Administrative actions MUST be audited.
6. Authorization logic MUST be tested with negative tests.
7. Role definitions MUST have owners.

## 17.5 Secret Handling

Rules:

1. Secrets MUST be stored in approved secret managers.
2. Secrets MUST NOT be committed.
3. Secrets MUST NOT be logged.
4. Secrets MUST be rotated on exposure.
5. Production secrets MUST NOT be available in developer machines by default.
6. Secret access MUST be least privilege.
7. Secret reads MUST be auditable for critical systems.

## 17.6 Cryptography Policies

Rules:

1. Do not implement custom cryptography.
2. Use approved libraries.
3. Use modern algorithms approved by security authority.
4. Keys MUST have lifecycle management.
5. Encryption at rest and in transit MUST be defined for sensitive data.
6. Randomness MUST come from cryptographically secure sources when used for security.
7. Hashing passwords requires password-hashing algorithms, not general-purpose hashes.

## 17.7 Dependency Trust

Security review for dependencies MUST include:

1. maintainer health
2. release history
3. vulnerability history
4. license
5. transitive dependencies
6. update cadence
7. package authenticity
8. build provenance where available
9. scope of access
10. replacement plan

## 17.8 Supply Chain Security

C2+ build pipelines MUST provide:

1. protected source branches
2. reviewed changes
3. isolated CI
4. pinned dependencies or lockfiles
5. artifact provenance
6. signed artifacts where feasible
7. SBOM for C2+ release artifacts
8. vulnerability scanning
9. reproducible or repeatable builds where feasible
10. deployment from built artifact, not developer machine

C3+ systems SHOULD target hardened build provenance equivalent to SLSA Build Level 3 where tooling permits.

## 17.9 Sandboxing

Untrusted code, files, plugins, templates, or user-supplied expressions MUST run with:

1. least privilege
2. resource limits
3. filesystem restrictions
4. network restrictions where feasible
5. timeout
6. audit logging
7. escape testing

## 17.10 Exploit Mitigation

Systems MUST implement:

1. input validation
2. output encoding
3. CSRF protection where applicable
4. injection prevention
5. safe deserialization
6. rate limiting
7. account lockout or abuse throttling
8. security headers where applicable
9. dependency patching
10. least privilege

## 17.11 Auditability

Security-relevant events MUST be audited:

1. login success/failure
2. privilege change
3. secret access
4. administrative action
5. data export
6. permission denial
7. configuration change
8. deployment
9. security policy change
10. access to sensitive records where required

Audit logs MUST be tamper-resistant for C3+ systems.

## 17.12 Secure Defaults

Defaults MUST be safe:

1. deny access
2. disable debug endpoints
3. require authentication
4. use TLS
5. avoid public exposure
6. minimize permissions
7. redact sensitive values
8. fail closed for security checks
9. enable logging for security events
10. require explicit opt-in for risky behavior

---

# 18. Performance Engineering

## 18.1 Performance Philosophy

Performance work MUST be driven by budgets, measurement, and user impact.

Optimization without measurement is allowed only when removing obvious waste without increasing complexity.

## 18.2 Latency Budgeting

Every C2+ user-facing operation MUST define a latency budget.

Example:

| Segment                | Budget   |
| ---------------------- | -------- |
| edge routing           | 50 ms    |
| authentication         | 100 ms   |
| application logic      | 200 ms   |
| database               | 150 ms   |
| external provider      | 500 ms   |
| response serialization | 50 ms    |
| total p95              | 1 second |

Rules:

1. Budgets MUST be assigned before major design.
2. Dependencies MUST receive sub-budgets.
3. Timeout values MUST align with budgets.
4. p95 and p99 MUST be tracked for critical paths.
5. Average latency alone is insufficient.

## 18.3 Throughput Budgeting

Throughput design MUST define:

1. expected steady load
2. expected peak load
3. burst load
4. growth assumption
5. bottleneck resource
6. backpressure behavior
7. overload behavior
8. test evidence

## 18.4 Memory Budgeting

Memory-sensitive components MUST define:

1. maximum heap or resident memory
2. per-request allocation budget where relevant
3. cache size limit
4. queue size limit
5. object retention risk
6. leak detection method

## 18.5 Storage Budgeting

Storage design MUST define:

1. write volume
2. read volume
3. retention
4. index cost
5. growth rate
6. archival policy
7. deletion policy
8. backup cost
9. restore time
10. compaction or vacuum strategy where applicable

## 18.6 Caching Doctrine

Caches are optional performance tools, not correctness sources unless explicitly designed as such.

Cache rules:

1. Define source of truth.
2. Define TTL.
3. Define invalidation.
4. Define staleness tolerance.
5. Define stampede protection.
6. Define cache-miss behavior.
7. Define failure behavior.
8. Define memory limit.
9. Do not cache authorization decisions without expiry and invalidation.
10. Do not hide correctness bugs with cache behavior.

## 18.7 Batching Rules

Batching is allowed when it reduces overhead without violating latency or fairness.

Rules:

1. Batch size MUST be bounded.
2. Batch wait time MUST be bounded.
3. Partial failure behavior MUST be defined.
4. Retries MUST not duplicate successful items.
5. Large batches MUST not starve small urgent work.

## 18.8 Async Rules

Asynchronous processing MUST define:

1. queue ownership
2. message schema
3. retry behavior
4. dead-letter behavior
5. idempotency
6. ordering assumptions
7. visibility timeout
8. backlog alert
9. replay strategy
10. poison message handling

## 18.9 Lock Contention Rules

Locks MUST be measured in critical systems.

Rules:

1. Keep critical sections short.
2. Do not perform I/O while holding locks unless justified.
3. Define lock ordering.
4. Use timeouts for lock acquisition.
5. Emit contention metrics where contention is plausible.
6. Prefer partitioning over global locks.

## 18.10 Scalability Analysis

C2+ systems MUST identify expected scaling dimension:

1. users
2. requests
3. data size
4. tenants
5. regions
6. devices
7. workflows
8. integrations
9. events
10. models or jobs

For each dimension, document current limit and next bottleneck.

## 18.11 Performance Regression Prevention

Performance-sensitive components MUST have:

1. benchmark
2. workload model
3. baseline
4. threshold
5. CI or scheduled execution
6. owner
7. regression triage process

---

# 19. Dependency Management

## 19.1 Dependency Philosophy

Every dependency is imported code, imported risk, imported maintenance, and imported policy.

The organization MUST prefer fewer, better-understood dependencies.

## 19.2 Dependency Approval

A dependency requires approval when it is:

1. runtime dependency
2. build dependency
3. security-relevant tool
4. infrastructure provider
5. database or storage engine
6. framework
7. package with transitive dependencies
8. vendor service
9. license-impacting library

Approval record MUST include:

1. purpose
2. owner
3. alternatives
4. license
5. version
6. transitive risk
7. security posture
8. operational impact
9. upgrade plan
10. exit plan

## 19.3 External Library Evaluation

Reject dependency if:

1. abandoned
2. unclear license
3. poor security history without remediation
4. excessive transitive dependencies
5. incompatible with runtime constraints
6. no replacement path
7. requires broad privileges unnecessarily
8. weak provenance for critical use
9. breaks build reproducibility
10. maintained by unknown or untrusted source for critical function

## 19.4 Transitive Dependencies

Rules:

1. Transitive dependencies MUST be visible in manifests or SBOM.
2. Critical vulnerabilities in transitive dependencies MUST be triaged.
3. Transitive dependency explosion requires review.
4. Dependency pinning strategy MUST be defined.
5. Dependency updates MUST be tested.

## 19.5 Upgrade Doctrine

Dependencies MUST be upgraded deliberately.

Minimum cadence:

| Dependency Type    | Cadence                         |
| ------------------ | ------------------------------- |
| security-critical  | immediate triage on advisory    |
| runtime framework  | quarterly review                |
| minor libraries    | semiannual review               |
| abandoned packages | replacement plan within 90 days |
| C3+ dependencies   | continuous monitoring           |

## 19.6 Deprecation Handling

When a dependency is deprecated:

1. assign owner
2. assess risk
3. identify replacement
4. create migration plan
5. schedule removal
6. monitor exposure
7. block new usage unless exception exists

## 19.7 Vendor Lock-In Prevention

Vendor lock-in is allowed only when:

1. benefit is material
2. exit cost is documented
3. data export path exists
4. failure mode is understood
5. contract risk is reviewed
6. critical operations have contingency

## 19.8 Build Reproducibility

Builds MUST be repeatable.

Rules:

1. Use lockfiles or pinned versions.
2. Build from clean environment.
3. Do not build from developer machines for production.
4. Store build artifacts.
5. Link artifact to source commit.
6. Record build parameters.
7. Protect build credentials.
8. Produce provenance for C2+.

---

# 20. Development Process

## 20.1 Branching Strategy

Default strategy:

1. trunk-based development
2. short-lived branches
3. frequent integration
4. feature flags for incomplete work
5. release branches only when needed for stabilization

Rules:

1. Branches SHOULD live less than 48 hours.
2. Branches older than 7 days require rebase or merge from trunk before review.
3. Long-lived branches require owner and merge plan.
4. Protected branches require CI and review.
5. Direct commits to protected production branches are forbidden except emergency procedure.

## 20.2 Review Rules

Every production code change MUST be reviewed.

Review MUST check:

1. correctness
2. invariants
3. failure behavior
4. observability
5. security
6. compatibility
7. tests
8. performance impact
9. operational impact
10. documentation impact

Reviewers MUST not approve code they do not understand.

## 20.3 Merge Requirements

Merge requires:

1. passing required CI
2. review approval
3. no unresolved critical comments
4. linked work item or change record
5. ownership confirmation
6. tests for changed behavior
7. migration approval where relevant
8. security review where relevant
9. compatibility proof where relevant
10. documentation updates where relevant

## 20.4 Release Process

A release MUST have:

1. version identifier
2. source commit
3. build artifact
4. changelog
5. test evidence
6. deployment plan
7. rollback plan
8. migration plan where relevant
9. monitoring plan
10. owner

## 20.5 Deployment Safety

Deployment MUST be staged for C2+ systems unless impossible.

Preferred order:

```text
dev -> test -> staging -> canary -> partial production -> full production
```

Deployment controls:

1. health checks
2. automated rollback where safe
3. canary metrics
4. error budget awareness
5. dependency monitoring
6. feature flag controls
7. migration status
8. operator visibility

## 20.6 Rollback Strategy

Rollback plan MUST specify:

1. rollback trigger
2. responsible owner
3. command or procedure
4. expected time
5. data compatibility
6. migration implications
7. communication path
8. validation after rollback

## 20.7 Migration Handling

Migration phases:

1. design
2. compatibility analysis
3. test on production-like data
4. backup
5. expand
6. deploy compatible code
7. backfill
8. verify
9. switch reads/writes
10. monitor
11. deprecate old path
12. remove old path

## 20.8 Feature Flag Rules

Feature flags MUST have:

1. owner
2. purpose
3. default value
4. rollout plan
5. kill-switch behavior
6. expiry date
7. cleanup ticket
8. telemetry
9. permission controls for risky flags

Permanent flags are configuration and must be documented as such.

Expired flags MUST be removed.

## 20.9 Incident Management

Severity levels:

| Severity | Definition                                       | Required Response          |
| -------- | ------------------------------------------------ | -------------------------- |
| SEV-1    | safety, security, major outage, severe data loss | immediate incident command |
| SEV-2    | major customer-visible degradation               | coordinated response       |
| SEV-3    | limited degradation or contained failure         | owner response             |
| SEV-4    | minor issue or near miss                         | normal triage              |

Incident roles:

1. incident commander
2. technical lead
3. communications lead
4. scribe
5. subject matter owners

## 20.10 Postmortem Rules

Postmortems are required for SEV-1, SEV-2, and recurring SEV-3 incidents.

Postmortem MUST include:

1. summary
2. impact
3. timeline
4. detection
5. contributing factors
6. what worked
7. what failed
8. root causes
9. corrective actions
10. owners and due dates
11. doctrine updates if needed

Postmortems MUST NOT stop at “human error.” Human error is a starting signal for system improvement.

---

# 21. Documentation Standards

## 21.1 Documentation Philosophy

Documentation is operational infrastructure.

A system is undocumented when a new qualified engineer cannot safely understand, change, deploy, operate, and debug it.

## 21.2 Required Documentation by Component

Every C2+ component MUST have:

1. README
2. owner record
3. architecture document
4. API or interface documentation
5. runbook
6. deployment guide
7. rollback guide
8. dependency rationale
9. data model description where stateful
10. observability guide
11. incident history link
12. decision records

## 21.3 Architecture Documents

Architecture document MUST include:

1. purpose
2. context diagram
3. component diagram
4. data ownership
5. interfaces
6. dependencies
7. failure modes
8. scaling assumptions
9. security boundaries
10. operational model
11. alternatives rejected
12. known limitations

## 21.4 API Documentation

API docs MUST include:

1. endpoint or message name
2. purpose
3. authentication
4. authorization
5. request schema
6. response schema
7. error codes
8. idempotency behavior
9. rate limits
10. versioning
11. examples
12. deprecation status

## 21.5 Operational Docs and Runbooks

Runbooks MUST include:

1. symptoms
2. dashboards
3. alerts
4. immediate mitigation
5. diagnosis steps
6. rollback steps
7. escalation path
8. customer impact guidance
9. data repair procedure where relevant
10. verification after recovery

A runbook that has never been exercised is unproven.

## 21.6 Onboarding Docs

Onboarding docs MUST teach:

1. system purpose
2. local setup
3. architecture
4. test strategy
5. release process
6. operational responsibilities
7. common failure modes
8. code conventions
9. contribution process
10. glossary

## 21.7 Code Comments

Comments SHOULD explain:

1. why code exists
2. invariant being protected
3. non-obvious tradeoff
4. external constraint
5. performance reason
6. security reason
7. compatibility requirement

Comments SHOULD NOT restate obvious code.

Bad:

```text
// increment i by 1
i = i + 1
```

Good:

```text
// Keep the old tax calculation until all invoices created before 2025-04-01 are settled.
// Removing this branch early changes historical invoice totals.
```

## 21.8 Decision Records

Decision records MUST be immutable except for status updates and supersession links.

A superseded decision MUST point to its replacement.

## 21.9 Change History

C2+ systems MUST maintain human-readable change history for releases affecting:

1. public behavior
2. data schema
3. security controls
4. operational procedures
5. dependencies
6. performance characteristics
7. compatibility

---

# 22. Team and Ownership Structure

## 22.1 Ownership Boundaries

Every component has exactly one accountable owner team.

Ownership includes:

1. code quality
2. architecture
3. tests
4. documentation
5. dependencies
6. security posture
7. deployment
8. operations
9. incidents
10. retirement

Shared ownership without final accountability is forbidden.

## 22.2 Escalation Rules

Escalation path MUST be defined for:

1. incidents
2. security vulnerabilities
3. data corruption
4. dependency failure
5. ownership conflict
6. architecture dispute
7. release risk
8. customer-impacting defect

Escalation does not transfer ownership unless explicitly reassigned.

## 22.3 Operational Responsibility

A team that owns a service owns its production behavior.

No team may ship a C2+ service unless it can:

1. monitor it
2. deploy it
3. roll it back
4. debug it
5. repair data it owns
6. answer incidents
7. maintain dependencies
8. update documentation

## 22.4 On-Call Doctrine

On-call MUST be:

1. staffed by trained responders
2. backed by escalation
3. supported by runbooks
4. protected from alert fatigue
5. measured for page quality
6. compensated or recognized according to organization policy
7. limited to actionable alerts

Alerts that do not require action MUST be removed or converted to tickets.

## 22.5 Code Stewardship

Stewardship rules:

1. Authors are responsible for initial quality.
2. Owners are responsible for lifecycle quality.
3. Reviewers are responsible for review quality.
4. Operators are responsible for safe operation but not for hidden design flaws.
5. Leadership is responsible for capacity to maintain reliability.

## 22.6 Institutional Memory Preservation

Teams MUST preserve memory through:

1. decision records
2. postmortems
3. onboarding docs
4. architecture reviews
5. code comments for invariants
6. recorded operational drills
7. ownership registry
8. deprecation history

## 22.7 Bus-Factor Reduction

No C2+ critical component may depend on one person’s private knowledge.

Controls:

1. at least two trained maintainers
2. documented runbook
3. reviewed architecture
4. tested recovery
5. cross-training
6. periodic handoff exercise

---

# 23. Maintenance and Evolution

## 23.1 Refactoring Rules

Refactoring is behavior-preserving structural change.

A refactor MUST have:

1. clear target
2. tests protecting behavior
3. limited scope
4. rollback path
5. no hidden feature change

Large refactors MUST be incremental.

## 23.2 Dead Code Removal

Dead code MUST be removed when:

1. no production path uses it
2. no supported version requires it
3. telemetry confirms inactivity where needed
4. deprecation window ended
5. removal is tested

Dead code may remain only with owner, reason, and review date.

## 23.3 Technical Debt Accounting

Technical debt record MUST include:

1. description
2. cause
3. impact
4. risk
5. owner
6. remediation plan
7. due date or review date
8. affected systems

Debt without owner is hidden risk.

## 23.4 Long-Term Compatibility

Compatibility policy MUST define:

1. supported versions
2. deprecation notice period
3. migration support
4. removal conditions
5. emergency removal conditions
6. communication channel
7. consumer tracking

Default minimum deprecation windows:

| Interface                           | Minimum Window    |
| ----------------------------------- | ----------------- |
| internal same-team API              | 2 weeks           |
| internal cross-team API             | 6 weeks           |
| external customer API               | 6 months          |
| regulated/safety-critical interface | assurance-defined |

## 23.5 Migration Strategies

Allowed migration strategies:

1. parallel run
2. strangler pattern
3. expand-migrate-contract
4. shadow reads
5. dual writes with reconciliation
6. event replay
7. blue/green switch
8. canary migration
9. batch backfill
10. tenant-by-tenant migration

Dual writes require reconciliation and divergence monitoring.

## 23.6 Rewrite Criteria

A rewrite is allowed only when:

1. maintenance cost is demonstrably unsustainable
2. failure risk of current system is higher than rewrite risk
3. incremental migration path exists
4. compatibility plan exists
5. data migration plan exists
6. staffing is committed
7. old system retirement plan exists
8. success metrics are defined

A rewrite is forbidden when motivated primarily by boredom, fashion, or dislike of old code.

## 23.7 Lifecycle Management

Every component MUST have lifecycle status:

1. experimental
2. active
3. maintenance
4. deprecated
5. retiring
6. retired

Each status has rules:

| Status       | Rule                     |
| ------------ | ------------------------ |
| experimental | no production dependency |
| active       | full support             |
| maintenance  | bug/security fixes only  |
| deprecated   | migration expected       |
| retiring     | removal scheduled        |
| retired      | no production use        |

## 23.8 Software Aging Prevention

Teams MUST regularly perform:

1. dependency updates
2. compatibility review
3. documentation refresh
4. backup restore test
5. incident trend review
6. code complexity review
7. ownership audit
8. security review
9. performance baseline review
10. dead code removal

---

# 24. API Design Rules

## 24.1 API Design Principles

APIs MUST be:

1. explicit
2. stable
3. minimal
4. compatible
5. secure
6. observable
7. idempotent where mutation may be retried
8. documented
9. testable
10. versioned when externally consumed

## 24.2 Request Rules

1. Validate all input.
2. Reject unknown enum values only when forward compatibility is not required.
3. Bound request size.
4. Use explicit units.
5. Do not infer identity from mutable display fields.
6. Use stable identifiers.
7. Require idempotency keys for externally retried mutations.

## 24.3 Response Rules

1. Return stable error codes.
2. Do not expose internal stack traces.
3. Include correlation ID where useful.
4. Paginate unbounded collections.
5. Do not return secrets.
6. Distinguish not found from unauthorized according to security policy.
7. Include deprecation metadata where applicable.

## 24.4 Error Code Rules

Error codes MUST be:

1. stable
2. documented
3. machine-readable
4. specific enough for action
5. not tied to internal implementation names

Example:

```json
{
  "errorCode": "payment_provider_timeout",
  "message": "Payment provider did not respond before timeout.",
  "correlationId": "req-123",
  "retryable": true
}
```

---

# 25. Data Model Evolution Rules

## 25.1 Data Ownership

Every table, collection, topic, bucket, index, or file format MUST have an owner.

## 25.2 Data Classification

Data MUST be classified:

1. public
2. internal
3. confidential
4. restricted
5. regulated
6. secret

Classification determines access, logging, retention, encryption, and deletion rules.

## 25.3 Invariant Enforcement

Critical invariants MUST be enforced by at least one of:

1. database constraint
2. transaction
3. domain validation
4. state machine
5. reconciliation job
6. audit check

Prefer storage-level enforcement for invariants that must survive application bugs.

## 25.4 Retention and Deletion

Data stores MUST define:

1. retention period
2. deletion trigger
3. deletion method
4. legal hold behavior
5. backup deletion implications
6. audit evidence
7. owner

## 25.5 Repairability

Data models MUST support repair.

Required for critical data:

1. stable identifiers
2. audit trail
3. creation and update timestamps
4. actor where applicable
5. version or concurrency marker
6. source of truth
7. reconciliation path

---

# 26. State Management Rules

## 26.1 Source of Truth

Every state value MUST have one source of truth.

Copies are allowed only when:

1. purpose is clear
2. freshness is defined
3. rebuild is possible or unnecessary
4. divergence detection exists where critical

## 26.2 State Machine Discipline

State machines MUST define:

1. states
2. allowed transitions
3. forbidden transitions
4. transition triggers
5. actor
6. side effects
7. idempotency behavior
8. audit fields

Example:

```text
PaymentIntent:
  Created -> Authorized -> Captured
  Created -> Cancelled
  Authorized -> Cancelled
  Authorized -> Expired
Forbidden:
  Captured -> Authorized
  Cancelled -> Captured
```

## 26.3 Configuration State

Configuration is state.

Rules:

1. Config changes require review for C2+.
2. Config must be versioned.
3. Risky config must have rollback.
4. Config must be validated at startup or load time.
5. Secrets are not ordinary config.
6. Dynamic config must emit audit events.

---

# 27. Concurrency Rules

## 27.1 Shared Mutable State

Shared mutable state MUST be avoided unless:

1. performance requires it
2. ownership is clear
3. synchronization is explicit
4. tests cover concurrent behavior
5. failure behavior is documented

## 27.2 Race Conditions

Race-sensitive logic MUST use:

1. atomic operations
2. transactions
3. locks
4. compare-and-swap
5. version checks
6. idempotency records
7. message serialization
8. partition ownership

## 27.3 Distributed Concurrency

Distributed concurrency MUST assume stale reads and duplicate writes.

Rules:

1. Use optimistic concurrency with version fields where feasible.
2. Use fencing tokens for distributed locks.
3. Use leases with expiry.
4. Avoid global locks.
5. Reconcile conflicts explicitly.
6. Do not rely on wall-clock ordering for correctness.

---

# 28. Compatibility Requirements

## 28.1 Compatibility Matrix

C2+ systems MUST maintain compatibility matrix for:

1. API versions
2. schema versions
3. client versions
4. deployment versions
5. dependency versions
6. configuration versions
7. protocol versions

## 28.2 Compatibility Testing

Compatibility tests MUST prove:

1. old client with new server
2. new client with old server where rolling deploy requires it
3. old schema with new code during migration
4. new schema with old code during rollback where required
5. unknown fields tolerated
6. deprecated fields still honored until removal

---

# 29. Release Strategies

Allowed release strategies:

1. rolling deployment
2. blue/green
3. canary
4. shadow deployment
5. dark launch
6. feature-flag rollout
7. region-by-region
8. tenant-by-tenant
9. batch release
10. emergency release

Selection rules:

| Risk                 | Required Strategy                          |
| -------------------- | ------------------------------------------ |
| low C1               | rolling acceptable                         |
| C2 user-facing       | staged or canary                           |
| C3 critical          | canary plus rollback and active monitoring |
| schema-risk release  | expand-migrate-contract                    |
| irreversible release | formal approval and roll-forward plan      |
| security emergency   | expedited release with retroactive review  |

---

# 30. Scalability Approaches

Scalability MUST proceed in this order:

1. measure current bottleneck
2. remove accidental inefficiency
3. add indexes or query improvements
4. cache safely
5. batch safely
6. partition data
7. scale vertically where economical
8. scale horizontally
9. isolate workloads
10. introduce service split only when needed
11. introduce distributed coordination only when unavoidable

Microservices are not a default scalability solution.

---

# 31. Review Methodologies

## 31.1 Design Review

Required for:

1. new C2+ service
2. new data store
3. new public API
4. security boundary change
5. dependency with major risk
6. migration affecting critical state
7. architecture split or merge
8. C3+ reliability change

Design review outputs:

1. approved
2. approved with conditions
3. rejected
4. returned for revision

## 31.2 Code Review

Reviewers MUST verify:

1. code matches intent
2. names are clear
3. errors handled
4. tests meaningful
5. invariants protected
6. no secret leaks
7. compatibility preserved
8. operational impact addressed

## 31.3 Operational Review

Before C2+ production launch:

1. dashboards exist
2. alerts exist
3. runbook exists
4. rollback tested
5. backup/restore tested where stateful
6. on-call trained
7. capacity tested
8. security reviewed
9. incident path defined
10. dependency failure tested

---

# 32. Deployment Methodologies

## 32.1 Deployment Invariants

Every deployment MUST preserve:

1. data integrity
2. compatibility during rollout
3. observability
4. rollback or roll-forward path
5. security controls
6. operational ownership

## 32.2 Deployment Gates

Deployment blocks when:

1. critical tests fail
2. security scanner reports unapproved critical issue
3. migration unsafe
4. rollback absent
5. owner absent for high-risk deploy
6. error budget exhausted without approval
7. required telemetry missing
8. dependency outage active and release increases risk

## 32.3 Emergency Deployment

Emergency deployment requires:

1. incident or urgent security issue
2. designated approver
3. minimal scoped change
4. rollback or roll-forward plan
5. post-deployment review
6. retroactive normal review within one business day

---

# 33. Anti-Patterns

## 33.1 Forbidden Practices

Forbidden:

1. unowned production systems
2. silent error swallowing
3. infinite retries
4. infinite timeouts
5. direct production data mutation without procedure
6. secrets in code or logs
7. breaking APIs without migration
8. deploying from developer machines
9. shared writable databases across services
10. ignoring failing tests
11. permanent undocumented feature flags
12. manual recurring operations without automation plan
13. production debugging without audit
14. dependency addition without approval
15. deleting historical data without retention review

## 33.2 Dangerous Organizational Patterns

| Pattern              | Failure Mode                 | Correction                 |
| -------------------- | ---------------------------- | -------------------------- |
| Hero culture         | burnout and hidden knowledge | rotation, docs, automation |
| Local optimization   | system-level failure         | shared SLOs                |
| Review theater       | defects pass unchecked       | reviewer accountability    |
| Deadline absolutism  | quality collapse             | tradeoff hierarchy         |
| Ownership ambiguity  | orphaned failures            | ownership registry         |
| Permanent exceptions | doctrine decay               | expiry and audit           |

## 33.3 Dangerous Architecture Patterns

| Pattern                          | Why Dangerous                      | Correction                        |
| -------------------------------- | ---------------------------------- | --------------------------------- |
| Distributed monolith             | service count without independence | restore boundaries or consolidate |
| Shared database between services | corrupts ownership                 | API or event ownership            |
| Chatty services                  | latency and failure amplification  | aggregate or colocate             |
| God service                      | unbounded responsibility           | split by capability               |
| Event spaghetti                  | untraceable workflows              | ownership and choreography review |
| Global mutable config            | unpredictable behavior             | versioned config and audit        |
| Cache as source of truth         | data loss/corruption               | define source of truth            |

## 33.4 Dangerous Coding Practices

1. clever one-liners hiding logic
2. mode flags controlling unrelated behavior
3. deep inheritance
4. generic names
5. hidden side effects
6. mutable globals
7. broad exception catches
8. reflection or dynamic loading without constraints
9. temporal coupling between functions
10. comments that contradict code

## 33.5 Dangerous Testing Practices

1. testing only happy paths
2. relying on mocks for all behavior
3. ignoring failure modes
4. deleting flaky tests
5. using sleeps instead of synchronization
6. snapshot testing unreadable output
7. treating coverage as quality
8. testing implementation instead of behavior

## 33.6 Reliability Theater

Reliability theater includes:

1. dashboards nobody uses
2. alerts nobody responds to
3. SLOs not tied to decisions
4. postmortems with no actions
5. chaos tests with no hypothesis
6. backups never restored
7. runbooks never exercised
8. error budgets ignored

Detection: incident reviews compare stated reliability process against actual decisions.

Correction: remove fake control or make it operationally binding.

## 33.7 Observability Theater

Observability theater includes:

1. high-volume logs without useful fields
2. metrics without owners
3. traces without propagation
4. dashboards without decision purpose
5. alerts on symptoms nobody acts on
6. telemetry that omits version or deployment
7. logging secrets under “debugging”

## 33.8 Cargo Cult Engineering

Cargo cult engineering is adopting tools or practices without understanding the failure mode they address.

Rule: Any new practice, framework, or tool MUST state:

1. problem solved
2. failure mode reduced
3. cost introduced
4. owner
5. success metric
6. removal condition

## 33.9 Abstraction Abuse

Abstraction abuse exists when indirection increases cognitive load without reducing lifecycle risk.

Correction:

1. inline unnecessary abstraction
2. split real boundaries
3. rename concepts
4. remove speculative interfaces
5. preserve only tested stable seams

## 33.10 Dependency Addiction

Dependency addiction exists when teams import libraries for trivial behavior or avoid understanding core system behavior.

Correction:

1. dependency audit
2. remove low-value packages
3. consolidate libraries
4. isolate frameworks
5. create approval threshold

## 33.11 Framework Overreach

Framework overreach occurs when framework structure dictates domain model, persistence, testing, or operations in ways that weaken correctness.

Rule: Frameworks are infrastructure. Domain logic MUST remain independent where feasible.

---

# 34. Enforcement System

## 34.1 Enforcement Levels

| Level | Mechanism                   |
| ----- | --------------------------- |
| L1    | documentation and ownership |
| L2    | code review                 |
| L3    | automated lint/test/check   |
| L4    | CI/CD gate                  |
| L5    | runtime monitor             |
| L6    | periodic audit              |
| L7    | independent assurance       |

C2+ non-negotiable rules MUST have L3 or stronger enforcement where technically feasible.

## 34.2 Exception Register

Every exception MUST include:

1. violated rule
2. reason
3. risk
4. compensating control
5. owner
6. approval
7. expiry
8. remediation plan

Exceptions without expiry are forbidden.

## 34.3 Compliance Metrics

Teams MUST track:

1. unowned components
2. expired exceptions
3. flaky tests
4. failed deploys
5. rollback frequency
6. incidents by cause
7. SLO compliance
8. dependency vulnerabilities
9. stale docs
10. unsupported API versions
11. dead feature flags
12. backup restore success

---

# 35. Internal Consistency Audit

## 35.1 Contradiction Audit

Potential conflict: **Reliability vs delivery speed**
Resolution: tradeoff hierarchy places reliability, correctness, recoverability, and data integrity above speed.

Potential conflict: **Availability vs data integrity**
Resolution: data integrity outranks availability. Systems may degrade or stop writes to avoid corruption.

Potential conflict: **Microservices vs operational simplicity**
Resolution: modular monolith is default; microservices require evidence and operational readiness.

Potential conflict: **Strict rules vs emergency response**
Resolution: emergency exceptions are allowed only with traceability, minimal scope, and retroactive review.

Potential conflict: **Coverage targets vs meaningful testing**
Resolution: coverage is a floor; meaningless tests are forbidden.

Potential conflict: **Security vs debugging**
Resolution: production debugging is permitted only with audit, least privilege, and privacy controls.

Potential conflict: **Backward compatibility vs urgent security fix**
Resolution: security may override compatibility only through incident-level process and communication.

## 35.2 Undefined Terminology Audit

Defined before operational use:

1. system
2. component
3. module
4. service
5. interface
6. state
7. invariant
8. failure/fault/defect/incident
9. release/deployment
10. compatibility
11. SLI/SLO/SLA/error budget
12. owner
13. exception/waiver

## 35.3 Enforceability Audit

Vague guidance replaced with enforceable mechanisms:

| Vague Idea         | Operational Rule                                     |
| ------------------ | ---------------------------------------------------- |
| “write good code”  | naming, size, side-effect, error-handling rules      |
| “be reliable”      | SLOs, retries, timeouts, recovery drills             |
| “document things”  | required doc set by component class                  |
| “secure software”  | trust boundaries, authn/authz, secrets, supply chain |
| “test thoroughly”  | required test types and ownership                    |
| “avoid complexity” | abstraction rules and architecture decision tree     |
| “handle incidents” | severity model, roles, postmortems                   |

## 35.4 Lifecycle Coverage Audit

Covered lifecycle stages:

1. conception
2. decision
3. architecture
4. implementation
5. testing
6. security review
7. release
8. deployment
9. operation
10. debugging
11. incident response
12. maintenance
13. migration
14. deprecation
15. retirement

## 35.5 Scale Audit

The doctrine scales by:

1. classification system
2. baseline rules for all production
3. stronger controls for C3/C4 and S3-S5
4. modular-monolith default
5. service extraction criteria
6. SLO/error budget governance
7. ownership and operational readiness requirements
8. DR and recovery escalation

## 35.6 Failure Scenario Audit

Explicitly handled:

1. bad deploy
2. dependency failure
3. timeout
4. retry storm
5. data corruption
6. data loss
7. schema migration failure
8. credential leak
9. queue backlog
10. partial workflow completion
11. cache stampede
12. regional failure
13. flaky tests
14. abandoned dependency
15. team turnover

## 35.7 Organizational Implementability Audit

The doctrine is implementable because it assigns:

1. owners
2. decision records
3. exception process
4. review gates
5. CI gates
6. operational runbooks
7. measurable controls
8. audit cadence
9. escalation paths
10. correction procedures

---

# 36. Final Operating Principle

A system governed by this doctrine is production-ready only when it can be:

1. understood by new maintainers
2. changed safely
3. tested meaningfully
4. deployed predictably
5. observed accurately
6. debugged under pressure
7. recovered after failure
8. secured against hostile use
9. evolved without breaking consumers
10. retired without leaving dangerous residue

Software that cannot be operated safely is not complete.
