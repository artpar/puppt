# Puppt Plan Examples

`puppt plan` inspects a deck, resolves the requested target, and reports whether a future edit can proceed without ambiguity. It does not write an output deck.

Run:

```sh
puppt plan input.pptx --edit edit.json --json
```

## Replace One Text Object

```json
{
  "operation": "replace_text",
  "target": {
    "type": "object_id",
    "object_id": "ppt/slides/slide1.xml#shape-2"
  },
  "replacement": "Updated title"
}
```

## Replace Text Across The Deck

Deck-wide scope allows multiple visible-text matches.

```json
{
  "operation": "replace_text",
  "target": {
    "type": "visible_text",
    "scope": "deck",
    "text": "Old product name"
  },
  "replacement": "New product name"
}
```

## Update Speaker Notes

```json
{
  "operation": "update_notes",
  "target": {
    "type": "notes",
    "slide_number": 3
  },
  "replacement": "Speaker notes for slide 3."
}
```

## Update Metadata

```json
{
  "operation": "update_metadata",
  "target": {
    "type": "metadata",
    "property": "title"
  },
  "replacement": "Quarterly Review"
}
```

## Replace An Image

Image targets use object IDs from `puppt inspect --json`.

```json
{
  "operation": "replace_image",
  "target": {
    "type": "object_id",
    "object_id": "ppt/slides/slide1.xml#rId2"
  },
  "image_path": "replacement-logo.png"
}
```

To replace the only image on a slide, use an image selector. If more than one image matches, planning returns `ambiguous`.

```json
{
  "operation": "replace_image",
  "target": {
    "type": "image",
    "slide_number": 1
  },
  "image_path": "replacement-logo.png"
}
```

## Add A Text Box

```json
{
  "operation": "add_text_box",
  "target": {
    "type": "slide_number",
    "slide_number": 2
  },
  "replacement": "Editable callout"
}
```

## Add A Simple Shape

```json
{
  "operation": "add_shape",
  "target": {
    "type": "slide_number",
    "slide_number": 2
  },
  "replacement": "Shape label"
}
```

## Move A Slide

```json
{
  "operation": "slide_move",
  "target": {
    "type": "slide_number",
    "slide_number": 7
  },
  "destination_slide_number": 2
}
```

## Add A Slide

`replacement` is the editable text for the new fixture-safe slide. The slide is inserted after the target slide number.

```json
{
  "operation": "slide_add",
  "target": {
    "type": "slide_number",
    "slide_number": 2
  },
  "replacement": "New slide title"
}
```

## Delete A Slide

```json
{
  "operation": "slide_delete",
  "target": {
    "type": "slide_number",
    "slide_number": 5
  }
}
```

## Duplicate A Slide

```json
{
  "operation": "slide_duplicate",
  "target": {
    "type": "slide_number",
    "slide_number": 4
  },
  "insert_after_slide": 8
}
```

## Failure Semantics

- `ready`: target is safe for the requested planning operation.
- `no_match`: no inspected object matched the target.
- `ambiguous`: more than one target matched without deck scope or another narrowing selector.
- `unsupported`: the operation, target type, or required planning fields are unsupported or incomplete.

Ambiguous, no-match, and unsupported plans emit JSON and return a non-zero CLI exit status.
