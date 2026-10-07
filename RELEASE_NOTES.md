# Subtitle Doctor v1.0.1 — release candidate

- Splits balanced italic, bold, underline and nested formatting at dialogue word boundaries, closing and reopening spanning tags for each new cue.
- Wraps dialogue without splitting tag attributes, commands or entities, while retaining existing line breaks and the document's output line-ending choice.
- Refuses unsafe cue merges: incompatible style/positioning/settings, cue identifiers, invalid or empty cues, gaps/overlaps, and stateful ASS overrides require explicit manual editing.
- Preserves centisecond boundaries when splitting ASS/SSA cues. Complex ruby/karaoke/ASS splits are refused rather than guessed.
- Retains SRT timing-line positioning metadata when saving in the same format.
- Adds regression coverage for formatting, timing, Unicode and subtitle round trips.
- Adds a distinct blue caption-panel/check Windows icon.

This is a release candidate, not a public release. Microsoft Edge WebView2 Runtime remains a separate dependency; it is not bundled or automatically installed.
