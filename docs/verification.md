# Windows verification

Verified locally on Windows amd64 on 2026-10-05 using Go 1.27.0,
Node.js/npm, and Wails CLI v2.16.0 matching the module version.

## Automated checks

- `npm ci`: passed, zero audit vulnerabilities.
- `npm run build`: passed TypeScript checking and Vite production build.
- `go vet ./...`: passed.
- `go test ./...`: passed all packages.
- `wails build -clean -s`: passed after explicitly building the frontend.
- The analyzer benchmark processed 50,000 entries in approximately 136 ms
  on the local machine. This is a backend measurement, not a full UI or
  hundreds-of-files stress test.

## Actual executable checks

Original sample dialogue was copied into three disposable SRT, ASS and VTT
files under a folder with spaces and Unicode in its name. No third-party
subtitle content was used. The production executable was launched, and the
following operations were exercised through its visible interface:

| Check | Observed result |
| --- | --- |
| Import directory through Paths | Three files loaded; 3/3, zero errors |
| Inspect SRT | Numbering, duplicate timestamps, exact duplicate, overlaps and CPS issues displayed |
| Repair preview and apply | Duplicate removed; entries renumbered 1 and 2; remaining issues still displayed |
| Offset +1500 ms | Starts changed from 3000/4500 to 4500/6000 ms |
| Offset -2000 ms | Starts changed to 2500/4000 ms |
| FPS 23.976 to 25 | Starts changed to 2398/3836 ms; batch processed all three files |
| Find/replace | Preview reported one match; Hello became Greetings |
| Save batch | Three separate outputs created; 3/3, zero errors |
| Repeated save | `doctor demo_fixed (2).srt` created without replacing the first output |
| SRT to VTT | Exported WEBVTT with matching millisecond timestamps and dialogue |
| SRT to ASS | Exported valid script/style/event sections, two Dialogue rows and centisecond timestamps |
| Original preservation | SHA-256 hashes of all three inputs matched before and after every export |
| Restart/settings | Output folder, suffix, encoding, format and analysis thresholds persisted |
| Final executable smoke test | Rebuilt executable started and imported the same three files successfully |

The temporary inputs, outputs and hash manifest were deleted after testing.
The temporary output-folder setting was cleared. Small permanent MIT test
fixtures remain in `internal/parser/testdata` for repeatable automated checks.

SRT, ASS and VTT were exercised in the desktop interface; SSA was exercised
by automated parsing and the sixteen-way conversion matrix. Native file
selection was opened, but its completion and drag/drop were not manually
verified. Backup/replacement, regex, two-point sync, wrapping, encoding,
undo/redo and reports have automated coverage; they were not all exercised
through the desktop interface. No claim of exhaustive manual verification
or a hundreds-of-files stress test is made.

## Correctness follow-up

The focused PR review pass added regression tests for ASS/SSA duplicate
comparison using current timing/text and style/name/effect metadata, invalid
cue preservation, shared settings/output suffix validation, WebVTT class,
voice, language and ruby markup, HTML line breaks, and markup-safe case tools.
Encoding tests perform actual safe writes and source replacement to verify
UTF-8 repair precedence, explicit export overrides, BOM state, backups and
subsequent saves.

The rebuilt executable imported a disposable WebVTT cue containing voice,
class and line-break tags with zero issues. Uppercase preview and apply
changed only dialogue; the tag spellings remained identical. Save / Export
displayed automatic encoding with the repair/preference rule and explicit
override options. The temporary fixture was deleted and settings were left
unchanged. Replacement encoding tests and ASS/SSA edit-created duplicates
were verified automatically rather than through every desktop workflow.
