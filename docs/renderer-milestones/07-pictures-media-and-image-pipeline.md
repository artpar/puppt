# M07 Pictures, Media, And Image Pipeline

## Objective

Implement the picture/media backend from first principles: source resolution,
decode, color management, crop, transform, sampling, masks, blip effects, and
unsupported reporting.

## Inputs

- `dml-picture.xsd`
- `dml-main.xsd` blip, blip fill, relative rect, black/white mode, and effects declarations
- current picture primitive/backend boundary
- `Picture 4`, `Google Shape;11;p15`, and top `Picture 2` fixtures

## In Scope

- Embedded and linked image relationship handling policy.
- PNG, JPEG, GIF, and supported SVG.
- Source crop and negative padding.
- Stretch/tile decision and `rotWithShape`.
- Color management and output transform.
- Sampling model and fractional bounds.
- Alpha, black/white modes, masks, soft edges, supported blip effects.

## Out Of Scope

- Full SVG renderer unless explicitly adopted as a primitive dependency.
- Video/audio playback.
- Non-static media behavior.

## Required Work

1. Finish picture primitive fields for every supported `CT_BlipFillProperties` and `CT_Blip` subset.
2. Define source-backed image sampling model.
3. Add synthetic fixtures for crop, padding, flip, rotate, alpha, masks, soft edge, and color.
4. Add unsupported reports for every unimplemented visible blip effect.
5. Replace one picture backend stage only when focused fixtures pass.
6. Run same-family neighbor fixtures and full corpus no-regression.

## Acceptance Criteria

- Current picture sampling replacement gate becomes a true pass gate, not only residual lock.
- `Picture 4` and `Google Shape;11;p15` either pass or have source-backed accepted residuals.
- Unsupported blip effects are visible in JSON when relevant.
- Matrix rows for picture/blip clauses are updated.

## Verification

```text
go test ./internal/render -run 'TestRenderPicture|TestPicture|Test.*Blip|Test.*Image|Test.*Sampling' -count=1
PUPPT_RUN_PICTURE_STAGE_ACCEPTANCE=1 go test ./internal/render -run TestCurrentPictureSamplingStageAcceptanceGate -count=1
PUPPT_MICRO_FIXTURE_MANIFEST=<focused-picture-manifest> go test ./internal/render -run TestMicroFixtureManifestComparison -count=1
go test ./internal/render -count=1
git diff --check
```

## Closeout

Update picture/media matrix rows and experiment log with accepted/rejected
sampling, color, and effect decisions.
