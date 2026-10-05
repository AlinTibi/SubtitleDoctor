# Contributing

Use a feature branch and submit a pull request. Keep subtitle processing offline,
protect originals by default, and describe any format loss or new limitations.
Run `npm ci` and `npm run build` in `frontend`, then `go vet ./...`,
`go test ./...` and `wails build -clean -s` on Windows.

Add legal, original fixtures and meaningful tests for parser, timing and output
safety changes. Manually check the built executable for UI changes. Do not commit
local subtitle collections, build binaries, dependency folders or user settings.
