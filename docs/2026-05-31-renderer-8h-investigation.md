# 2026-05-31 Renderer Parity Investigation

This document is an evidence-backed account of the renderer work from roughly
09:20 IST to 18:50 IST on 2026-05-31. It was written after checking the raw
Codex conversation logs, the local git history, the current worktree, and the
renderer artifact directories.

The short answer is uncomfortable but clear: the day produced real source
understanding, maintainability work, font discovery work, and a long list of
rejected experiments, but it did not materially move the accepted pixel parity
gate. The current accepted exact-font baseline is still:

```text
61/61 slides differ
9,321,023 differing pixels
0 unsupported reports in the scanned exact-font artifacts
```

## Sources Checked

Raw Codex session logs checked:

```text
/Users/artpar/.codex/sessions/2026/05/31/rollout-2026-05-31T09-20-12-019e7c27-5220-75f0-8b12-65d7bd9a527b.jsonl
/Users/artpar/.codex/sessions/2026/05/31/rollout-2026-05-31T09-20-26-019e7c27-8837-7b53-8460-24a18a735138.jsonl
/Users/artpar/.codex/sessions/2026/05/31/rollout-2026-05-31T10-17-08-019e7c5b-71a0-79f3-9682-63afd5216780.jsonl
/Users/artpar/.codex/sessions/2026/05/31/rollout-2026-05-31T12-16-04-019e7cc8-57a0-7b62-a63c-8faf402f59c5.jsonl
```

The log search also matched these May 31 sessions, but they were not primary
sources for the Puppt renderer work:

```text
/Users/artpar/.codex/sessions/2026/05/31/rollout-2026-05-31T09-13-40-019e7c21-56b1-7e20-8305-8ccf0b382f32.jsonl  # agent4/canva cwd
/Users/artpar/.codex/sessions/2026/05/31/rollout-2026-05-31T15-09-43-019e7d67-52a0-7d32-b009-c793e98fd2ae.jsonl  # pragma cwd
/Users/artpar/.codex/sessions/2026/05/31/rollout-2026-05-31T17-47-31-019e7df7-cb2f-7092-8b97-7c34ad2579e1.jsonl  # mostly agent4/canva; includes this report work
```

Session event counts from the relevant large logs:

```text
rollout-2026-05-31T09-20-26...  compacted=8  event_msg=2611  response_item=3907  turn_context=35
rollout-2026-05-31T10-17-08...  compacted=8  event_msg=1391  response_item=3094  turn_context=82
rollout-2026-05-31T12-16-04...  compacted=9  event_msg=2969  response_item=4415  turn_context=33
```

That matters because the work was split across multiple rollouts and
compactions. A single `git log` view cannot reconstruct the sequence or the
rejected experiments.

Other sources checked:

```text
swe_skill.md
git log --since='2026-05-31 09:00'
git status --short
git diff --stat
renderer artifact directories under /tmp/puppt-artifacts-*
```

I initially started to answer from git/artifacts without checking these raw
conversation logs. That was a process error. The account below is based on the
raw logs as well.

The log extraction focused on assistant messages and tool calls containing:

```text
61/61, 9,225, 9,321, 9,325, 9,417, 9,366, 9,339,
Calibri, exact, unsupported, baseline, worsened, reverted,
swe_skill, split renderer, render_test, text-style
```

This was deliberate: the question is why the exact renderer parity number did
not move after a long day, so the useful evidence is the baseline changes,
accepted commits, rejected candidate totals, and workflow/process turns.

## Current State

Current commit:

```text
d534789 render: split color modifier tests
```

Current dirty worktree:

```text
 M internal/render/render_test.go
?? internal/render/render_text_styles_test.go
```

The dirty state is from an interrupted behavior-neutral test split. The new
`internal/render/render_text_styles_test.go` exists, and one test was deleted
from `internal/render/render_test.go`, but the move was not completed. Running
the full test package in this state is expected to fail or at least be invalid
because many tests are duplicated between the old file and the new file.

Current relevant line counts:

```text
9675 internal/render/render_test.go
 343 internal/render/render_text_styles_test.go
 284 internal/render/render_color_test.go
 584 internal/render/render_realworld_test.go
```

This matters because the user explicitly objected to a 10k-line renderer/test
surface and said following `swe_skill.md` is part of the goal. The production
renderer source was split, and some tests were split, but the large
`render_test.go` file is still not fixed.

## What swe_skill.md Required

The binding local doctrine says, among other things:

- Go code must be maintainable production software, not script-like prototype code.
- Every package must have a narrow responsibility and tests.
- The current code path must be inspected before editing.
- Source documents must be checked.
- Golden outputs are part of the system.
- Validation commands must be run and recorded.
- Unsupported behavior must be preserved, skipped with explanation, or rejected
  before mutation.

Against that standard, the work did some things correctly:

- It used source XML, rendered artifacts, and corpus golden runs before accepting
  renderer changes.
- It rejected many plausible changes when the full corpus got worse.
- It split the production renderer into focused files.
- It made unsupported/partial font reporting more honest.

It also fell short:

- The raw conversation logs were not checked before the first report attempt.
- The experiment ledger existed mostly in transient artifacts and conversation
  logs, not as a durable repo-local record.
- The baseline number drift was not reconciled early enough.
- The test split was started but left dirty and incomplete.
- `internal/render/render_test.go` remains far too large.

## Accepted Commits Since 09:20 IST

These are the commits currently visible from the local git history after the
work started:

```text
d61a011 2026-05-31 09:29:58 render: report Office font substitutions
705e328 2026-05-31 09:38:42 render: tag PNG color metadata
b48ef72 2026-05-31 09:54:01 render: discover Office cloud Calibri fonts
90a1263 2026-05-31 10:06:01 render: resolve canonical Calibri font files
3c18ae0 2026-05-31 10:16:13 render: search common Calibri font roots
fc0e9b5 2026-05-31 10:30:23 render: report unsupported table border joins
20a6034 2026-05-31 10:37:00 render: paint round table border joins
6990c99 2026-05-31 10:50:42 render: ignore hidden image placeholder marker fonts
8f78626 2026-05-31 11:06:13 render: skip hidden image placeholder marker text
0190556 2026-05-31 11:22:08 render: enrich real-world diff artifacts
12b3b05 2026-05-31 11:35:42 test: cover empty placeholder defRPr inheritance
07cc232 2026-05-31 11:55:13 render: antialias fractional rectangle fills
b1824a5 2026-05-31 11:59:37 render: preserve thin fractional rect edges
0db42a0 2026-05-31 12:09:40 render: honor normal autofit line spacing reduction
3f6ef1c 2026-05-31 12:37:08 render: discover Office Calibri font locations
3bbd692 2026-05-31 15:34:36 render: honor explicit false text styles
652fc70 2026-05-31 15:56:00 render: honor dashed rectangle line caps
479ed5c 2026-05-31 16:09:22 render: discover cached Office Calibri fonts
0d98074 2026-05-31 16:35:53 render: split renderer responsibilities
c29191f 2026-05-31 17:25:10 render: split real-world renderer tests
3083a27 2026-05-31 17:40:59 render: resolve capitalized Office Calibri styles
d534789 2026-05-31 18:15:45 render: split color modifier tests
```

Not all of these were direct pixel-parity improvements. Several were honesty,
diagnostics, font discovery, or maintainability changes.

## Timeline From Conversation Logs

### 09:20 to 09:30 IST: Initial Parity Check and Honest Font Reporting

The session started by reading the local project state and `swe_skill.md`.
The renderer golden harness was run with real-world artifacts.

The current 61-slide gate failed every slide. At this point the renderer was
already returning `status=ok` for all slides even though visual parity was far
off. The logs show the first tracked current total around `9,225,602` differing
pixels before the exact Calibri font work was introduced.

The first accepted change made Calibri and Calibri Light substitutions honest:
when the renderer used Carlito or another substitute instead of the exact Office
font, it reported that as partial support instead of silently calling the render
fully supported.

Accepted commit:

```text
d61a011 render: report Office font substitutions
```

Impact:

- Improved reporting honesty.
- Did not solve the pixel gap.

### 09:30 to 10:16 IST: PNG Metadata, Font Discovery, and Rejected Placement Work

The logs show several candidate changes around placement and rendering metadata.

Rejected experiments:

- Fractional EMU placement experiment: worsened the golden corpus and was
  reverted.
- `spAutoFit` resize suppression: worsened the corpus to about `9,230,532` and
  was reverted.
- Table-border alpha change: worsened the corpus to about `9,226,655` and was
  reverted.
- CatmullRom all-in-bounds raster change: worsened the corpus to about
  `9,256,856` and was reverted.

Accepted commits:

```text
705e328 render: tag PNG color metadata
b48ef72 render: discover Office cloud Calibri fonts
90a1263 render: resolve canonical Calibri font files
3c18ae0 render: search common Calibri font roots
```

Impact:

- Better metadata and font lookup behavior.
- No meaningful accepted movement on the parity total.

### 10:16 to 12:10 IST: Table Joins, Placeholder Text, Artifact Enrichment

The renderer began reporting table border join limitations and then painting
round table border joins. Hidden placeholder marker text and marker fonts were
handled. The real-world artifact harness was enriched to make diff analysis
easier.

Accepted commits included:

```text
fc0e9b5 render: report unsupported table border joins
20a6034 render: paint round table border joins
6990c99 render: ignore hidden image placeholder marker fonts
8f78626 render: skip hidden image placeholder marker text
0190556 render: enrich real-world diff artifacts
```

This phase produced source-correct behavior and better diagnostics, but it still
did not close the 61-slide visual gap.

### 12:32 to 12:50 IST: Exact Calibri Download and Font Reality Check

The user said to download the real Calibri files. The logs show the official
Microsoft Word package path was used, a large package was downloaded/extracted,
and exact Calibri fonts were mapped through `PUPPT_FONT_MAP`.

Important outcome:

- Exact Calibri and Calibri Light removed the earlier partial font substitution
  reports.
- The accepted exact-font total became about `9,321,046`.
- This was worse than the earlier non-exact-font `9,225,602` number, but it was
  the more honest render because the missing-font condition was gone.

The logs also show that a hardcoded PowerPoint app-path candidate was caught and
removed. Word/Excel DFonts remained.

### 12:42 to 13:00 IST: Calibri Light Bold Investigation

A recurring finding was that titles often inherit bold from the master while
using Calibri Light. The available Office cache has Calibri Light regular and
italic, but no real Calibri Light bold face.

Rejected experiments:

- Synthetic bold for Calibri Light: worsened the corpus to about `9,326M`.
- Mapping Calibri Light bold to Calibri Bold: worsened the corpus.

The conclusion was that the problem is real but a broad synthetic-bold fix is
not corpus-safe.

This is also where the user challenged the 10k-line file issue and whether
`swe_skill.md` permits that. The answer is no: it does not. The later split work
was a response to this, but it remains incomplete for tests.

### 15:13 to 16:10 IST: More Fidelity Probes and Two Source-Correct Commits

After the pause, the logs show another set of direct renderer experiments.

Rejected experiments:

- Radial gradient fill-tile bounds: worsened to about `10,508,009`.
- Narrower font mapping: worsened to about `9,332,904`.
- Full hinting: worsened to about `9,336,889`.
- Per-segment gamma: worsened to about `10,130,460`.
- Other color and placement probes were reverted when corpus totals worsened.

Accepted commits:

```text
3bbd692 render: honor explicit false text styles
652fc70 render: honor dashed rectangle line caps
479ed5c render: discover cached Office Calibri fonts
```

Impact:

- Fixed real DrawingML/source interpretation issues.
- Did not materially improve the current exact-font parity total.

### 16:24 to 16:36 IST: Production Renderer Split

The user's maintainability concern became explicit. The renderer implementation
was split into focused files. The raw log notes that `render.go` became about
135 lines, with responsibilities moved into files for background, color, fonts,
geometry, gradients, inheritance/theme, output, paint, parse, pictures, shape
parse, tables, text, unsupported reporting, XML, and types.

Accepted commit:

```text
0d98074 render: split renderer responsibilities
```

Impact:

- This was the strongest `swe_skill.md` compliance improvement of the day.
- It was behavior-neutral and did not improve pixel parity.

### 17:06 to 17:41 IST: Exact-Font Baseline, More Rejections, Test Split, Capitalized Fonts

The logs report the current exact-font baseline as:

```text
61/61 slides differ
9,321,023 differing pixels
```

Rejected experiments:

- Synthetic bold after the split: worsened to `9,325,889`.
- Disabling derived normal autofit: worsened to `9,417,726`.
- Vertical hinting: no accepted improvement.
- P3 quantization floor: worsened badly.
- Picture crop floor rounding: worsened to about `9,322,424`.

The real-world renderer tests were split out of the giant test file.

Accepted commits:

```text
c29191f render: split real-world renderer tests
3083a27 render: resolve capitalized Office Calibri styles
```

The capitalized-font commit mattered because auto-discovery had failed to match
files such as `Calibrib.ttf`, `Calibrii.ttf`, and `Calibriz.ttf` due case
sensitivity. After that fix, auto-discovery matched the explicit font map at
`9,321,023`.

### 17:45 to 18:16 IST: Color, Paragraph, Table, and Color-Test Split Work

More candidate fixes were tried and reverted.

Rejected experiments:

- Fluent Calibri experiment: worsened to `9,546,831`.
- ICC matrix / ColorSync style color conversion: helped one inspected light-blue
  fill but worsened the corpus to `9,407,951`.
- Trimming leading/trailing empty text paragraphs: improved one inspected WHO
  box but worsened the corpus to `9,368,715`.

Accepted commit:

```text
d534789 render: split color modifier tests
```

Impact:

- Another maintainability split.
- No parity improvement.

### 18:17 to 18:44 IST: Late Object-Specific Investigations

The logs show deeper inspection of:

- WHO slide 12 table tint mismatch.
- Table row-span behavior.
- Table border antialiasing via vector rendering.
- EPA residential slide 16 text/autofit.
- NormalAutoFit line spacing.
- EPA slide 1 title font inheritance.

Rejected experiments:

- Table border antialiasing via vector: worsened to `9,341,892`.
- Mapping Calibri to Arial: worsened to `9,520,819`.
- Line-spacing change: focused tests passed, but full golden worsened to
  `9,366,321`.
- Calibri Light bold remap for EPA slide 1: worsened to `9,339,531`.

Useful findings:

- The WHO table row-span was not the main culprit.
- One-channel color/tint mismatch exists in WHO slide 12, but broad color
  changes are not safe.
- Unscaled wrapping matched the Apple reference better in one EPA slide 16
  case, but the broad normal-autofit change made the full corpus much worse.
- EPA slide 1 title inherits master bold with Calibri Light, but there is no
  exact Calibri Light bold face in the local Office cache.

The final action before the report request was an attempted behavior-neutral
split of text-style tests. That was interrupted and left the dirty worktree
state described above.

## Raw Log Evidence Highlights

These are representative raw-log facts that anchor the timeline above. Times
below are UTC from the JSONL log, with IST in parentheses.

```text
03:52Z (09:22 IST)
"The current 61-slide gate fails on every slide, but the renderer reports status=ok with no unsupported items."

04:23Z (09:53 IST)
"The final golden failure is unchanged at 9,225,602 differing pixels, which is expected here because no exact Calibri/Calibri Light files were found."

05:07Z (10:37 IST)
"0/61 Apple Notes reference slides pass and 61/61 still differ, with 9,225,630 total differing pixels."

07:09Z (12:39 IST)
"Current Calibri-mapped evidence is regenerated: all 61 slides still dimension-match, all unsupported/partial reports are now zero, and the total diff is 9,321,046."

07:17Z (12:47 IST)
"Synthetic-bold implementation is wrong for the target parity: total diff worsened from 9,321,046 to 9,325,976."

09:45Z (15:15 IST)
"Radial gradient fill-tile bounds ... total diff jumped to 10,508,009."

10:03Z (15:33 IST)
"The 61-slide diff is unchanged at 9,321,046 ... corrects a real source-style inheritance bug."

10:28Z (15:58 IST)
"The authoritative baseline ... current total is 9,321,023 pixels with no unsupported reports."

11:06Z (16:36 IST)
"render.go is now 135 lines instead of ~12.9k."

11:36Z (17:06 IST)
"61 / 61 Apple Notes reference slides still differ ... Total differing pixels: 9,321,023."

12:10Z (17:40 IST)
"Auto-discovery now matches the explicit PUPPT_FONT_MAP result: 9,321,023 total differing pixels and no unsupported reports."

12:18Z (17:48 IST)
"The separate 0db42a0 worktree also lands at roughly 9,321,046 with the current reference artifacts, so the 9,225,601 number is not reproducible from the current filesystem."

12:46Z (18:16 IST)
"Committed one coherent checkpoint: d534789 render: split color modifier tests."

13:02Z (18:32 IST)
"The corpus rejected the line-spacing change: total differing pixels went from 9,321,023 to 9,366,321."

13:20Z (18:50 IST)
"You're right. I checked memory and repo evidence, but not the raw Codex session logs. I'm going to do that now..."
```

The raw logs therefore support the main conclusion: the team did not simply sit
at one metric for eight hours, but the accepted parity metric remained stuck
because most candidate visual fixes either worsened the full corpus or were
behavior-neutral/diagnostic/maintainability work.

## Artifact Metrics

The artifact directories show why most work was not accepted: almost every
candidate got worse on the full corpus.

```text
puppt-artifacts-0db42a0                 61  9321046   61  1194124124
puppt-artifacts-current                 61  9348360   61  1202101989
puppt-artifacts-current-exact           61  9321023   61  1194211240
puppt-artifacts-capitalized-fonts       61  9321023   61  1194211240
puppt-artifacts-line-spacing            61  9366321   61  1226811387
puppt-artifacts-calibri-light-bold-map  61  9339531   61  1202694222
puppt-artifacts-table-aa                61  9341892   61  1194820909
puppt-artifacts-arial-calibri           61  9520819   61  1287956154
puppt-artifacts-synthbold               61  9325889   61  1196492095
puppt-artifacts-noderivedautofit        61  9417726   61  1247636710
puppt-artifacts-hintvertical            61  9321023   61  1194211240
puppt-artifacts-radialbounds            61 10507986   61  1267656761
puppt-artifacts-p3floor                 61 12036116   61  1202462344
puppt-artifacts-cropfloor               61  9322424   61  1196120661
puppt-artifacts-fluent-calibri          61  9546831   61  1247461636
puppt-artifacts-colorsync-matrix        61  9407951   61  1194216778
puppt-artifacts-trim-empty-paragraphs   61  9368715   61  1217599519
```

The important comparison is:

```text
0db42a0 exact-ish baseline:      9,321,046 differing pixels
current exact accepted baseline: 9,321,023 differing pixels
net accepted movement:          23 pixels
```

If measured against the older `9,225,601` or `9,225,602` number, current exact
font rendering looks worse. The logs indicate that number belonged to a
different font/substitution state and was not reproducible later in the separate
worktree check. The honest current comparison is the exact-font baseline.

## Current Worst Slides

The current exact-font artifact set still has large mismatches across both
decks. The worst slides include:

```text
EPA-generate-2021-presentation             slide 1   308113
EPA-generate-2021-presentation             slide 10  306058
WHO-HIV-testing-algorithms-toolkit         slide 12  300716
WHO-HIV-testing-algorithms-toolkit         slide 6   295925
EPA-generate-2021-presentation             slide 6   295889
WHO-HIV-testing-algorithms-toolkit         slide 7   288964
EPA-generate-2021-presentation             slide 2   277239
WHO-HIV-testing-algorithms-toolkit         slide 2   275083
EPA-generate-2021-presentation             slide 12  235027
EPA-generate-2021-presentation             slide 5   228448
EPA-generate-2021-presentation             slide 9   223070
WHO-HIV-testing-algorithms-toolkit         slide 3   216426
```

This is not a one-slide or one-object problem. It is broad renderer fidelity.

## Why So Much Time Produced So Little Pixel Movement

### 1. The gate was strict and rejected broad plausible fixes

Many changes looked plausible from PowerPoint/OpenXML reasoning or improved one
inspected object, but the full corpus got worse. Those changes were reverted.
That was the correct discipline for a golden renderer gate, but it means the
visible accepted pixel total barely moved.

Examples:

- ColorSync matrix improved one light-blue fill but worsened the corpus.
- Trimming empty paragraphs improved one WHO text box but worsened the corpus.
- Line spacing passed focused tests but worsened real-world parity.
- Synthetic bold was plausible for Calibri Light bold inheritance, but worsened
  the corpus.
- NormalAutoFit changes helped one visual hypothesis but made many slides much
  worse.

### 2. Exact fonts changed the baseline instead of magically closing it

Downloading and mapping exact Calibri files solved an honesty problem: the
renderer was no longer silently substituting fonts. It did not solve font
metrics, rasterization, text layout, autofit, color, gradients, antialiasing, or
picture sampling differences.

After exact fonts, the renderer had fewer unsupported/partial reports, but the
visual mismatch remained.

### 3. Some work was maintainability and compliance, not parity

The renderer split and test splits matter for `swe_skill.md`, but they were not
intended to reduce pixel diffs. They consumed time because the user correctly
challenged the huge-file direction.

Accepted maintainability work:

- `0d98074 render: split renderer responsibilities`
- `c29191f render: split real-world renderer tests`
- `d534789 render: split color modifier tests`

Remaining maintainability debt:

- `internal/render/render_test.go` is still 9675 lines.
- `internal/render/render_text_styles_test.go` is an incomplete dirty split.

### 4. The baseline drift was not controlled tightly enough

The session moved between:

- pre-exact-font substitution totals around `9,225,602`;
- exact-font totals around `9,321,046`;
- current exact-font totals at `9,321,023`;
- rejected experiment totals ranging from `9,322,424` to over `12,036,116`.

Those numbers were not put into a durable experiment ledger early enough. That
created avoidable confusion about whether the renderer was improving, worsening,
or merely being measured under a different font state.

### 5. The search space stayed too broad

The day touched fonts, gradients, color management, antialiasing, line caps,
table joins, autofit, paragraph trimming, placeholder inheritance, picture crop
rounding, and test structure. That breadth found useful facts, but it also
spread the investigation across too many uncertain renderer domains.

For parity, the next work needs a smaller loop: one primitive, one source XML
claim, one expected reference behavior, one focused test, one full-corpus check.

## What Actually Improved

Meaningful improvements:

- Font substitution reporting became honest.
- Exact Office Calibri discovery now works better, including capitalized style
  files such as `Calibrib.ttf`.
- Unsupported table border join reporting improved.
- Round table border joins were implemented.
- Hidden placeholder marker text/font noise was reduced.
- Real-world diff artifact output improved.
- Explicit false text style inheritance was fixed.
- Dashed rectangle line caps were corrected.
- The renderer implementation was split into focused files.
- Some renderer tests were split out of the giant test file.

What did not improve:

- The 61-slide visual parity gate.
- The total differing-pixel count in any meaningful way.
- The large test file problem enough to satisfy the spirit of `swe_skill.md`.

## Process Failures

### Raw logs were not checked before the first report attempt

This was the user's latest objection and it was correct. The first report
attempt started from git history, artifacts, and memory rather than the raw
Codex session logs. That is not enough for a "what happened over 8 hours"
investigation.

### Baseline definitions were allowed to drift

The work moved from substitution fonts to exact fonts. That is a legitimate
measurement change, but it should have been called out as a baseline reset at
the moment it happened.

### The experiment record was too implicit

The artifact directories contain the evidence, and the conversation logs contain
the narrative, but the repo did not get a durable `RENDERER_EXPERIMENT_LOG.md`
as the experiments happened.

### Too much broad probing continued after repeated corpus regressions

After several broad fixes worsened the full corpus, the process should have
shifted earlier to a tighter per-object proof loop.

### The test split was left dirty

The last attempted `swe_skill.md` cleanup was interrupted after creating
`internal/render/render_text_styles_test.go` and editing `internal/render/render_test.go`.
That state needs to be resolved before more renderer changes.

## What Remains

Renderer parity remaining:

- All 61 real-world slides still differ.
- Current exact-font baseline is `9,321,023` differing pixels.
- The mismatch spans fonts, text layout/autofit, colors, gradients, table
  rendering, antialiasing, and picture sampling.

Maintenance remaining:

- Finish or revert the partial text-style test split.
- Continue splitting `internal/render/render_test.go` into focused test files.
- Add a durable renderer experiment ledger so rejected paths are not repeated.

Evidence remaining:

- For any future accepted visual change, preserve:
  - source XML snippet or part path;
  - focused test;
  - before/after full-corpus artifact total;
  - reason the change is source-correct even if pixel-neutral.

## Recommended Next Steps

1. Resolve the dirty test split before touching renderer behavior again.
2. Add `docs/RENDERER_EXPERIMENT_LOG.md` from the artifact results above.
3. Pick one low-scope object mismatch, not EPA slide 1 as a whole.
4. Prove one primitive at a time with source XML plus focused test.
5. Run the full corpus after each candidate.
6. Commit only if the change improves the corpus, is behavior-neutral
   maintainability work, or fixes a source-proven bug with no corpus regression.

## Bottom Line

The last 8 hours did not achieve renderer parity. It produced a cleaner and more
honest renderer, exact font discovery, many rejected experiments, and partial
`swe_skill.md` cleanup, but the accepted pixel result is effectively flat:

```text
9,321,046 -> 9,321,023
```

That is a 23-pixel accepted improvement on a 9.3 million pixel mismatch. The
remaining work is still the main renderer fidelity problem, not a small tail.
