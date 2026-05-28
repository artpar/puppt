# Puppt User Experience

## Experience Goal

Puppt should feel like a precise presentation assistant for agents and automation workflows.

The experience should be calm, direct, and trustworthy. A user should be able to ask for a change to a presentation and know that Puppt will inspect the deck, target the right content, preserve unrelated material, and report the result clearly.

The most important experience quality is confidence. Users should trust Puppt with existing business decks because it makes careful changes and avoids silent damage.

## Primary Users

### AI Agents

The main user of Puppt is often an AI agent acting on behalf of a person.

The agent needs to understand a deck, decide what to change, apply the requested edit, validate the result, and explain the outcome. Puppt should give the agent enough structure to avoid guessing.

The agent should be able to answer questions such as:

- What slides are in this deck?
- Which slide contains this phrase?
- Where is this title?
- Which images are present?
- Which slides have speaker notes?
- What changed after the edit?
- Is the output safe to return to the user?

### Human Operators

Some people will use Puppt directly, especially technical users, analysts, consultants, and operators who work with repeatable presentation tasks.

They need clear actions, readable output, and confidence that Puppt is changing the intended parts of a deck. They may not care about the inner structure of the file, but they care deeply about the result staying editable and shareable.

### Human Reviewers

A reviewer may not use Puppt directly. They may only receive the edited deck and a summary of changes.

For this user, the experience succeeds when the deck opens normally, looks familiar, remains editable, and reflects the requested changes without unexpected damage.

## Interaction Model

Puppt should support a simple loop:

1. Inspect the presentation.
2. Decide what should change.
3. Apply precise edits.
4. Validate the result.
5. Review what changed.

This loop should work for a single quick edit and for larger automated workflows.

Puppt should not require the user or agent to act blindly. Inspection should come before edits whenever there is uncertainty about where content lives in the deck.

## Agent-First Behavior

Puppt should make presentations understandable to agents.

An agent should receive clear facts about the deck rather than vague descriptions. The facts should identify slides, text, images, notes, and warnings in a stable way. This lets the agent reason about a presentation before making changes.

Edits should also be explicit. The agent should be able to describe what it intends to change, review the intended change, apply it, and then confirm the result.

Good agent behavior with Puppt should look like this:

- Inspect the deck before editing.
- Locate the exact slide or object.
- Make the smallest change that satisfies the user request.
- Validate the changed presentation.
- Report the specific changes made.
- Surface warnings instead of hiding uncertainty.

## Human-Facing Behavior

Even when Puppt is used by an agent, the human experience matters.

The user should feel that the tool is careful with their files. It should avoid dramatic language, hidden assumptions, and unexplained changes. It should tell the user what it found, what it changed, and what still needs attention.

Puppt should use plain, direct product language:

- "Updated title on slide 4."
- "Replaced image on slide 7."
- "No matching text found."
- "This file type is not supported."
- "The deck was updated, but preview is incomplete for two slides."

The tone should be professional and minimal. It should not feel like a design generator, a chatbot, or a marketing product. It should feel like a reliable editing instrument.

## Core Journeys

### Inspect a Deck

The user wants to understand what is inside an existing presentation.

Puppt should provide a structured view of the deck:

- Presentation name and basic properties.
- Slide count and slide order.
- Slide titles.
- Visible text.
- Images and media references.
- Speaker notes.
- Repeated phrases.
- Potential issues or unsupported features.

The inspection should help an agent or human decide where edits should happen.

Success means the user can answer, "What is in this deck, and where is it?"

### Make a Specific Edit

The user asks for a precise change, such as:

- "Change the title on slide 2."
- "Replace the logo everywhere."
- "Update all mentions of the old product name."
- "Add speaker notes to the final slide."
- "Move the appendix slide before the summary."

Puppt should target the requested content without disturbing unrelated slides.

Success means the edited deck contains the requested change, preserves the rest of the presentation, and provides a clear change summary.

### Create a New Deck

The user or agent starts from structured content and needs a new presentation.

Puppt should create an editable deck with useful slide structure, titles, body content, notes, images, and basic visual organization.

The first version should be practical and editable rather than overly designed. It should give the user a clean starting point that can be refined later.

Success means the user receives a normal PowerPoint file, not a flattened visual artifact.

### Refresh a Recurring Presentation

The user has a recurring deck that changes every week, month, or quarter.

Puppt should help update dates, metrics, commentary, screenshots, speaker notes, and status slides while preserving the deck's established layout.

Success means the update becomes repeatable and less dependent on manual slide-by-slide editing.

### Replace Images or Branding

The user needs to update logos, screenshots, product images, customer names, taglines, or brand references.

Puppt should help identify where those assets or phrases appear, replace the intended items, and confirm the update afterward.

Success means the user can make controlled brand changes across a deck without accidentally changing unrelated content.

### Validate Before Sharing

Before sending a presentation, the user wants confidence that the file is still usable.

Puppt should check for obvious structural or content issues and explain anything that needs attention.

Success means the user knows whether the file is ready to share, has warnings, or needs further review.

### Preview a Result

Sometimes the user needs a visual check.

Puppt should eventually provide previews that help catch obvious layout, image, or text issues. Preview should support review, not replace editable deck output.

Success means the user or agent can quickly confirm that edited slides look reasonable before delivery.

## Editing Expectations

Puppt should make common edits feel straightforward:

- Replace text on a slide.
- Replace text across the whole deck.
- Add a new slide.
- Delete a slide.
- Reorder slides.
- Duplicate a slide.
- Add or update speaker notes.
- Replace an image.
- Add a text box.
- Update deck title or author information.
- Identify where a phrase or image appears.

The user should be able to target content by slide number, visible text, title, object identity, or deck-level property.

When multiple matches exist, Puppt should avoid guessing unless the request clearly allows a broad change. It should make ambiguity visible so the user or agent can choose the right target.

## Review Expectations

After changes are made, Puppt should summarize the outcome clearly.

A useful review should answer:

- Which slides changed?
- What content was added, removed, or replaced?
- Were any requested edits skipped?
- Were there ambiguous matches?
- Were there unsupported features?
- Is the deck still valid?
- Is visual preview available or incomplete?

The review should be useful to both agents and humans. Agents need structured facts. Humans need concise explanations.

## Failure Experience

Failure should be explicit and recoverable.

Puppt should avoid vague errors. When something fails, the user should know what happened and what can be done next.

Examples of good failure behavior:

- If the file type is unsupported, say that clearly.
- If a requested phrase is not found, report that no matching text was found.
- If a slide number does not exist, say which slide range is available.
- If an edit would affect multiple matches, report the ambiguity.
- If a feature cannot be previewed, preserve the deck and explain the preview limitation.
- If validation finds a problem, identify the affected slide or deck area when possible.

Puppt should never silently drop content, flatten a deck, or claim full success when the result has warnings.

## Trust Principles

### No Silent Destruction

Puppt should preserve unrelated content and explain anything it cannot safely handle.

### Clear Targeting

The user should know what part of the deck is being changed.

### Reversible Intent

The intended edit should be understandable before it is applied. A user or agent should be able to review the change request and see what it is supposed to do.

### Honest Support Boundaries

Puppt should be clear about what it fully supports, what it preserves without editing, and what it cannot handle yet.

### Editable Output

The result should remain a normal editable PowerPoint presentation.

## Product Voice

Puppt should sound precise, professional, and restrained.

Use direct language:

- "Found 12 slides."
- "Updated 3 text matches."
- "Skipped 1 ambiguous match."
- "Validated presentation structure."
- "Preview unavailable for this slide."

Avoid playful, promotional, or overly conversational language. Puppt is a tool for careful document work, and its voice should reflect that.

## What Good Feels Like

A successful Puppt experience feels uneventful in the best way.

The user gives a deck and a request. The agent inspects the deck, uses Puppt to make the edit, validates the result, and returns an editable presentation with a clear summary. The deck still feels like the original deck, just updated.

The user does not have to wonder whether the tool recreated the slides, flattened the design, lost speaker notes, or changed unrelated content. Puppt makes the edit traceable and the outcome reviewable.

That is the standard: precise changes, preserved decks, clear results.
