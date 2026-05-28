# Puppt Create Examples

`puppt create` builds an editable `.pptx` deck from structured JSON and validates the output after writing.

Run:

```sh
puppt create --input deck.json --out output.pptx --json
```

## Deck JSON

```json
{
  "metadata": {
    "title": "Quarterly Review",
    "author": "Puppt",
    "subject": "Q4"
  },
  "slides": [
    {
      "layout": "title",
      "title": "Quarterly Review"
    },
    {
      "layout": "section",
      "title": "Operating Highlights",
      "notes": "Pause before the metrics."
    },
    {
      "layout": "title_body",
      "title": "Metrics",
      "body": "Revenue and retention moved together.",
      "bullets": ["Revenue up", "Retention stable"],
      "image_path": "chart.png"
    }
  ]
}
```

Supported layouts are `title`, `section`, and `title_body`. Generated decks use deterministic editable XML parts, simple layout/master parts, and local image bytes when `image_path` is provided.
