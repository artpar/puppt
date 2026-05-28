# Decision 0001: Use Cobra for CLI Command Structure

## Status

Accepted.

## Context

Puppt v1 requires a stable agent-friendly CLI with subcommands for inspect, plan, edit, create, validate, and review. The project doctrine now prefers reliable third-party Go libraries where they reduce correctness, compatibility, parsing, writing, validation, or maintenance risk.

## Decision

Use `github.com/spf13/cobra` for CLI command structure and `github.com/spf13/pflag` transitively for flag handling.

## Evidence

- Version adopted: `github.com/spf13/cobra v1.10.2`.
- License: Apache-2.0.
- Maintenance: widely used Go CLI framework with active releases and broad production usage.
- Scope: command routing, help output, argument and flag structure.
- Does not read, write, validate, or mutate `.pptx` content.

## Why Library Use Is Safer

Hand-rolled command parsing would add low-value bespoke code around help output, subcommand routing, flags, and test setup. Cobra gives a stable command model and keeps `cmd/puppt` thin, matching the Puppt Go package rules.

## Boundaries

Business logic MUST stay outside Cobra command handlers. Cobra command handlers should call internal workflow packages and return explicit errors.

## Fallback

If Cobra becomes abandoned or unsafe, replace it behind `internal/cli` while preserving the external command names and JSON contracts.
