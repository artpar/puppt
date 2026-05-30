# Decision 0003: Use Go x/image for Renderer Font Primitives

## Status

Accepted.

## Context

`puppt render` must remain a Puppt-owned `.pptx` interpretation and render pipeline. Real-world renderer fixtures contain substantial text, and Go's standard library can write PNG images but does not provide font parsing, shaping, or glyph rasterization APIs.

The renderer still owns PresentationML parsing, object ordering, geometry, relationship resolution, unsupported reporting, and PNG command behavior. A bounded rendering primitive dependency is acceptable when it does not read, write, mutate, or interpret `.pptx` packages.

The real-world fixtures use Office theme fonts such as Calibri and Calibri Light, but `.pptx` packages generally do not embed those fonts. Host font availability is not deterministic enough for production renderer tests, and silently using unrelated system fonts hides fidelity gaps.

## Decision

Use `golang.org/x/image` for low-level image/font rendering primitives behind `internal/render`.

This dependency is allowed only for drawing primitives such as font faces and glyph rasterization. It must not own `.pptx` package reading, PresentationML interpretation, mutation, validation, or command JSON behavior.

Allow exact renderer font files to be pinned through `PUPPT_FONT_MAP` for production and CI environments that can provide Office-compatible fonts without changing code. Prefer locally installed Carlito metric-compatible fonts for Calibri and Calibri Light when exact Calibri-family fonts are unavailable, then bundle the unmodified Carlito TTF files under `internal/render/assets/fonts/carlito` as the deterministic fallback. Carlito is pinned from `googlefonts/carlito` commit `3a810cab78ebd6e2e4eed42af9e8453c4f9b850a` and carries its upstream `OFL.txt` license file.

## Consequences

- Puppt can render real glyphs without shelling out to office software or browser engines.
- Renderer behavior remains testable through focused PNG tests and the real-world golden comparison harness.
- Production and CI can pin exact font files explicitly instead of relying on ambient host font discovery.
- Font selection and text layout remain Puppt-owned code and must report partial or unsupported behavior when fidelity is incomplete.
- Rendering Calibri-family text no longer depends on host-specific Helvetica/Arial availability. The JSON reports an unresolved font issue only when neither the exact requested family nor a supported Carlito substitute can be resolved.
