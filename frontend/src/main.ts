import "./style.css";
import "./dialog.css";

type Entry = {
  index: number;
  start: number;
  end: number;
  text: string;
  fields?: string[];
  invalid?: boolean;
};
type Issue = { code: string; entry: number; message: string; severity: string };
type Report = {
  issues: Issue[];
  repairs: string[];
  originalCount: number;
  finalCount: number;
  output: string;
  error?: string;
};
type Summary = {
  path: string;
  name: string;
  format: string;
  encoding: string;
  count: number;
  first: number;
  last: number;
  duration: number;
  issues: number;
  dirty: boolean;
  output: string;
};
type View = {
  doc: { entries: Entry[]; format: string; encoding: string; path: string };
  report: Report;
  target: string;
  canUndo: boolean;
  canRedo: boolean;
  total: number;
};
type Settings = {
  outputFolder: string;
  suffix: string;
  encoding: string;
  format: string;
  maxChars: number;
  maxLines: number;
  cps: number;
  backup: boolean;
  overwrite: boolean;
};
type Operation = { kind: string; [key: string]: unknown };
type Preview = {
  changed: number;
  matches: number;
  examples: Entry[];
  warnings: string[];
};
type Progress = {
  kind: string;
  current: string;
  done: number;
  total: number;
  error: string;
  finished: boolean;
  cancelled: boolean;
};
declare global {
  interface Window {
    go: { main: { App: Record<string, (...args: unknown[]) => Promise<any>> } };
    runtime: {
      EventsOn: (name: string, cb: (p: Progress) => void) => void;
      OnFileDrop: (
        cb: (x: number, y: number, paths: string[]) => void,
        useDropTarget: boolean,
      ) => void;
    };
  }
}
const api = window.go.main.App;
const $ = <T extends HTMLElement = HTMLElement>(s: string) =>
  document.querySelector<T>(s)!;
const esc = (s: unknown) =>
  String(s ?? "").replace(
    /[&<>"']/g,
    (c) =>
      ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" })[
        c
      ]!,
  );
let queue: Summary[] = [],
  path = "",
  page = 0,
  view: View | undefined,
  prefs: Settings,
  busy = false,
  draft = false,
  selected = new Set<number>(),
  previewOp: Operation | undefined;
let errors: string[] = [],
  results: string[] = [];
function time(ms: number) {
  const sign = ms < 0 ? "-" : "";
  ms = Math.abs(ms);
  return `${sign}${String(Math.floor(ms / 3600000)).padStart(2, "0")}:${String(Math.floor(ms / 60000) % 60).padStart(2, "0")}:${String(Math.floor(ms / 1000) % 60).padStart(2, "0")}.${String(ms % 1000).padStart(3, "0")}`;
}
function parseTime(s: string) {
  const m = /^(\d+):([0-5]\d):([0-5]\d)[,.](\d{3})$/.exec(s.trim());
  if (!m) throw new Error("Use HH:MM:SS.mmm for entry times.");
  return (+m[1] * 3600 + +m[2] * 60 + +m[3]) * 1000 + +m[4];
}
function notify(s: string, error = false) {
  $("#notice").textContent = s;
  $("#notice").className = error ? "notice error" : "notice";
  $("#status-text").textContent = s;
}
async function run(fn: () => Promise<void>) {
  try {
    await fn();
  } catch (e) {
    notify(String(e), true);
  }
}
const btn = (id: string, label: string, title = label) =>
  `<button id="${id}" title="${title}">${label}</button>`;
$("#app").innerHTML =
  `<header><div class="brand"><span class="brand-mark">SD</span><div><strong>Subtitle Doctor</strong><small>OFFLINE SUBTITLE WORKSPACE</small></div></div><span class="offline"><i></i> Local & private</span></header>
<nav>${btn("add", "＋ Add Files", "Ctrl+O")}${btn("folder", "Add Folder")}${btn("scan", "Scan")}${btn("repair", "Repair")}${btn("sync", "Sync")}${btn("convert", "Convert")}${btn("save", "Save / Export", "Ctrl+S")}${btn("settings", "Settings")}<span class="spacer"></span>${btn("undo", "↶ Undo", "Ctrl+Z")}${btn("redo", "↷ Redo", "Ctrl+Y")}</nav>
<div id="notice" class="notice">Ready. Add subtitles or drop files anywhere in this window.</div>
<main><aside class="queue"><div class="section-heading">FILE QUEUE <span id="queue-count">0</span></div><div id="queue"></div><div class="queue-note">SRT · ASS · SSA · VTT<br>Sources are protected by default.</div></aside>
<section class="workspace"><div class="workspace-heading"><div><h1 id="file-title">Your subtitles, in good shape.</h1><span id="file-detail">Inspect, repair and synchronize. Entirely offline.</span></div><span id="target" class="badge">NO FILE SELECTED</span></div><div id="stats" class="stats"></div>
<div class="editor-toolbar">${btn("find", "Find / Replace", "Ctrl+F")}${btn("text-tools", "Text / Lines")}${btn("insert", "Insert")}${btn("duplicate", "Duplicate")}${btn("delete", "Delete")}${btn("merge", "Merge")}${btn("split", "Split")}${btn("up", "↑")}${btn("down", "↓")}<span class="spacer"></span><button id="commit" class="accent">Apply Edits</button></div>
<div class="table-wrap"><table><thead><tr><th><input type="checkbox" id="select-all" aria-label="Select visible entries"></th><th>#</th><th>Start</th><th>End</th><th>Duration</th><th class="text-col">Subtitle text</th></tr></thead><tbody id="entries"></tbody></table><div id="empty" class="empty"><div class="empty-icon">≋</div><h2>A clearer view of every line</h2><p>Add a file to see its timing, text and detected issues.</p><button id="empty-add" class="accent">＋ Add subtitle files</button></div></div>
<div class="pagination"><button id="prev">← Previous</button><span id="page">0 entries</span><button id="next">Next →</button><span class="spacer"></span><span id="draft">Edits are undoable for this session.</span></div></section>
<aside class="issues"><div class="section-heading">INSPECTION <span id="issue-count">0</span></div><div class="issue-tabs"><button id="issues-tab" class="active">Issues</button><button id="report-tab">Report</button></div><div id="issue-list"><p class="muted">Scan results appear here.</p></div><div class="report-export">${btn("report-json", "JSON report")}${btn("report-csv", "CSV report")}</div></aside></main>
<footer><span id="status-dot">●</span><span id="status-text">Ready</span><span class="spacer"></span><progress id="progress" max="100" value="0"></progress><span id="progress-label"></span><button id="cancel" hidden>Cancel job</button></footer>
<dialog id="dialog"><form id="dialog-form"><div class="dialog-heading"><h2 id="dialog-title"></h2><button type="button" id="close-dialog" aria-label="Close dialog">×</button></div><div id="dialog-body"></div><div id="preview" hidden></div><div id="dialog-error" class="error"></div><div class="dialog-actions"><span id="dialog-hint"></span><span class="spacer"></span><button type="button" id="preview-button">Preview</button><button type="submit" id="apply-button" class="accent">Apply</button></div></form></dialog>`;
function setBusy(v: boolean) {
  busy = v;
  for (const b of document.querySelectorAll<HTMLButtonElement>(
    "nav button,.editor-toolbar button,.pagination button,#empty-add",
  ))
    b.disabled = v;
  $("#cancel").hidden = !v;
  $("#status-dot").className = v ? "working" : "";
  if (!v) {
    $("#undo").toggleAttribute("disabled", !view?.canUndo);
    $("#redo").toggleAttribute("disabled", !view?.canRedo);
  }
}
async function refresh() {
  queue = await api.Queue();
  $("#queue-count").textContent = String(queue.length);
  $("#queue").innerHTML = queue
    .map(
      (f) =>
        `<button class="file ${f.path === path ? "active" : ""}" data-path="${esc(f.path)}"><span class="file-icon">${esc(f.format.toUpperCase())}</span><span class="file-name">${esc(f.name)}${f.dirty ? ' <i class="dirty">●</i>' : ""}<small>${f.count.toLocaleString()} entries · ${esc(f.encoding)}</small></span><span class="file-issues ${f.issues ? "warning" : ""}">${f.issues}</span></button>`,
    )
    .join("");
  for (const b of document.querySelectorAll<HTMLButtonElement>(".file"))
    b.onclick = () =>
      run(async () => {
        if (busy) return;
        if (draft && !confirm("Discard unapplied table edits?")) return;
        path = b.dataset.path!;
        page = 0;
        draft = false;
        selected.clear();
        await load();
        await refresh();
      });
  if (!path && queue.length) {
    path = queue[0].path;
    await load();
    await refresh();
  }
}
async function load() {
  if (!path) return;
  view = await api.GetFile(path, page);
  selected.clear();
  draft = false;
  render();
  $("#undo").toggleAttribute("disabled", !view!.canUndo || busy);
  $("#redo").toggleAttribute("disabled", !view!.canRedo || busy);
}
function render() {
  if (!view) return;
  const s = queue.find((f) => f.path === path);
  $("#empty").hidden = true;
  $("#file-title").textContent = s?.name ?? path.split(/[\\/]/).pop()!;
  $("#file-detail").textContent = path;
  $("#target").textContent =
    `${view.doc.format.toUpperCase()} → ${view.target.toUpperCase()}`;
  $("#stats").innerHTML = [
    ["ENTRIES", view.total.toLocaleString()],
    ["ENCODING", view.doc.encoding],
    ["FIRST", time(s?.first ?? 0)],
    ["LAST", time(s?.last ?? 0)],
    ["DURATION", `${((s?.duration ?? 0) / 1000).toFixed(3)} s`],
  ]
    .map(([k, v]) => `<div><small>${k}</small><strong>${esc(v)}</strong></div>`)
    .join("");
  renderEntries();
  showIssues();
  $("#page").textContent =
    `Page ${page + 1} / ${Math.max(1, Math.ceil(view.total / 200))} · ${view.total.toLocaleString()} entries`;
  $("#prev").toggleAttribute("disabled", page === 0 || busy);
  $("#next").toggleAttribute(
    "disabled",
    (page + 1) * 200 >= view.total || busy,
  );
  $("#draft").textContent = "Edits are undoable for this session.";
}
function renderEntries() {
  if (!view) return;
  $("#entries").innerHTML = view.doc.entries
    .map(
      (e, i) =>
        `<tr data-i="${i}" class="${selected.has(i) ? "selected" : ""}"><td><input type="checkbox" data-select="${i}" ${selected.has(i) ? "checked" : ""} aria-label="Select entry ${e.index}"></td><td class="index">${e.index}</td><td><input data-field="start" value="${time(e.start)}" aria-label="Entry ${e.index} start"></td><td><input data-field="end" value="${time(e.end)}" aria-label="Entry ${e.index} end"></td><td class="duration">${((e.end - e.start) / 1000).toFixed(3)} s</td><td><textarea data-field="text" rows="${Math.min(5, Math.max(2, e.text.split("\n").length))}" aria-label="Entry ${e.index} text">${esc(e.text)}</textarea></td></tr>`,
    )
    .join("");
  for (const b of document.querySelectorAll<HTMLInputElement>("[data-select]"))
    b.onchange = () => {
      const i = +b.dataset.select!;
      b.checked ? selected.add(i) : selected.delete(i);
      b.closest("tr")!.classList.toggle("selected", b.checked);
    };
  for (const el of document.querySelectorAll<
    HTMLInputElement | HTMLTextAreaElement
  >("[data-field]"))
    el.oninput = () => {
      draft = true;
      $("#draft").textContent = "Unapplied edits · click Apply Edits";
    };
  $("#select-all").toggleAttribute("checked", false);
}
function capture() {
  if (!view) return;
  for (const row of document.querySelectorAll<HTMLTableRowElement>(
    "tr[data-i]",
  )) {
    const e = view.doc.entries[+row.dataset.i!];
    e.start = parseTime(
      row.querySelector<HTMLInputElement>('[data-field="start"]')!.value,
    );
    e.end = parseTime(
      row.querySelector<HTMLInputElement>('[data-field="end"]')!.value,
    );
    e.text = row.querySelector<HTMLTextAreaElement>("textarea")!.value;
    if (e.end <= e.start)
      throw new Error(`Entry ${e.index}: end must be after start.`);
    e.invalid = false;
  }
}
async function commit() {
  if (!view) return;
  capture();
  await api.EditPage(path, page, view.doc.entries);
  draft = false;
  await refresh();
  await load();
  notify("Entry edits applied. Undo is available.");
}
function showIssues() {
  if (!view) return;
  $("#issue-count").textContent = String(view.report.issues.length);
  $("#issue-list").innerHTML = view.report.issues.length
    ? view.report.issues
        .slice(0, 600)
        .map(
          (v) =>
            `<button class="issue ${v.severity}" data-entry="${v.entry}"><span class="issue-code">${esc(v.code.replaceAll("_", " "))} ${v.entry ? `· #${v.entry}` : ""}</span><span>${esc(v.message)}</span></button>`,
        )
        .join("") +
      (view.report.issues.length > 600
        ? "<p>Showing first 600 issues. Export the full report.</p>"
        : "")
    : '<div class="clean"><span>✓</span><strong>No issues detected</strong><p>Current checks passed.</p></div>';
  for (const b of document.querySelectorAll<HTMLButtonElement>("[data-entry]"))
    b.onclick = () =>
      run(async () => {
        if (draft) await commit();
        const n = +b.dataset.entry!;
        if (n) {
          page = Math.floor((n - 1) / 200);
          await load();
          const row = $<HTMLTableRowElement>(`tr[data-i="${(n - 1) % 200}"]`);
          row?.scrollIntoView({ block: "center" });
          row?.classList.add("selected");
        }
      });
}
function showReport() {
  if (!view) return;
  $("#issue-list").innerHTML =
    `<dl><dt>Original entries</dt><dd>${view.report.originalCount}</dd><dt>Final entries</dt><dd>${view.report.finalCount}</dd><dt>Output</dt><dd>${esc(view.report.output || "Not saved yet")}</dd></dl>${view.report.error ? `<p class="error">${esc(view.report.error)}</p>` : ""}<h3>Applied operations</h3>${view.report.repairs.map((x) => `<p class="repair-log">${esc(x)}</p>`).join("") || '<p class="muted">No operations applied.</p>'}`;
}
let dialogMode = "",
  getOperation: () => Operation = () => ({ kind: "" });
const field = (
  name: string,
  label: string,
  value: string | number,
  type = "text",
) =>
  `<label>${label}<input name="${name}" type="${type}" value="${esc(value)}" ${type === "number" ? 'step="any"' : ""}></label>`;
const check = (name: string, label: string, on = false) =>
  `<label class="check"><input type="checkbox" name="${name}" ${on ? "checked" : ""}>${label}</label>`;
const select = (
  name: string,
  label: string,
  values: string[],
  current: string,
) =>
  `<label>${label}<select name="${name}">${values.map((v) => `<option ${v === current ? "selected" : ""} value="${esc(v)}">${esc(v)}</option>`).join("")}</select></label>`;
const val = (name: string) =>
  $<HTMLInputElement | HTMLSelectElement>(`[name="${name}"]`).value;
const checked = (name: string) =>
  $<HTMLInputElement>(`[name="${name}"]`).checked;
const num = (name: string) => {
  const n = Number(val(name));
  if (!Number.isFinite(n) || val(name).trim() === "")
    throw new Error(`${name} must be a finite number`);
  return n;
};
function operationCommon() {
  return `<div class="scope">${check("all", "Apply to every file in the queue")}${check("save-batch", "Also save outputs (output folder required)")}</div><p class="muted">Preview shows the selected file. Batch results are reported per file.</p>`;
}
function openDialog(mode: string) {
  if (busy) return;
  if (!["settings", "save"].includes(mode) && !view) {
    notify("Add and select a subtitle file first.", true);
    return;
  }
  if (draft) {
    notify("Apply table edits before opening another tool.", true);
    return;
  }
  dialogMode = mode;
  previewOp = undefined;
  $("#preview").hidden = true;
  $("#preview").innerHTML = "";
  $("#dialog-error").textContent = "";
  $("#preview-button").hidden = mode === "settings" || mode === "save";
  $("#apply-button").textContent =
    mode === "save"
      ? "Save outputs"
      : mode === "settings"
        ? "Save settings"
        : "Apply";
  $("#apply-button").toggleAttribute(
    "disabled",
    mode !== "settings" && mode !== "save",
  );
  let html = "";
  let title = "";
  if (mode === "repair") {
    title = "Repair subtitles";
    html = `<p>Select the repairs to apply. Scanning never changes your file.</p><div class="checks">${check("renumber", "Renumber SRT entries", true)}${check("empty", "Remove empty entries", true)}${check("duplicates", "Remove exact duplicates", true)}${check("order", "Sort entries by start time", true)}${check("duration", "Fix zero / negative durations (review inferred end times)")}${check("lineEndings", "Normalize line endings to CRLF", true)}${check("spacing", "Normalize spacing", true)}${check("utf8", "Convert encoding to UTF-8", true)}${check("removeBOM", "Remove UTF-8 BOM", true)}${check("tags", "Clean unmatched known HTML tag delimiters")}</div><p class="muted">Unbalanced tags and unreadable timestamps require manual review. Dialogue is never invented.</p>`;
    getOperation = () => ({
      kind: "repair",
      repair: Object.fromEntries(
        [
          "renumber",
          "empty",
          "duplicates",
          "order",
          "duration",
          "lineEndings",
          "spacing",
          "utf8",
          "removeBOM",
          "tags",
        ].map((k) => [k, checked(k)]),
      ),
    });
  }
  if (mode === "sync") {
    title = "Synchronize timing";
    html = `${select("sync-mode", "Method", ["offset", "resync", "fps"], "offset")}<div id="offset-fields" class="fields">${field("offset", "Offset (positive = later)", 1500, "number")}${select("unit", "Unit", ["milliseconds", "seconds"], "milliseconds")}</div><div id="resync-fields" class="fields" hidden>${field("a", "Original reference 1 (ms)", 1000, "number")}${field("b", "Desired reference 1 (ms)", 1500, "number")}${field("c", "Original reference 2 (ms)", 60000, "number")}${field("d", "Desired reference 2 (ms)", 60500, "number")}</div><div id="fps-fields" class="fields" hidden>${field("from", "Source FPS", 23.976, "number")}${field("to", "Target FPS", 25, "number")}<p>Presets: <button type="button" data-fps="23.976">23.976</button> <button type="button" data-fps="24">24</button> <button type="button" data-fps="25">25</button> <button type="button" data-fps="29.97">29.97</button> <button type="button" data-fps="30">30</button></p></div><p class="muted">FPS uses source FPS / target FPS. Changes that produce negative times are rejected.</p>`;
    getOperation = () => {
      const kind = val("sync-mode");
      return kind === "offset"
        ? {
            kind,
            offset: num("offset") * (val("unit") === "seconds" ? 1000 : 1),
          }
        : kind === "fps"
          ? { kind, from: num("from"), to: num("to") }
          : { kind, a: num("a"), b: num("b"), c: num("c"), d: num("d") };
    };
  }
  if (mode === "convert") {
    title = "Convert format";
    html = `${select("format", "Target format", ["srt", "ass", "ssa", "vtt"], view!.target)}<p>Timing and dialogue are retained. Review styling warnings in the preview.</p>`;
    getOperation = () => ({ kind: "convert", format: val("format") });
  }
  if (mode === "find") {
    title = "Find / Replace";
    html = `${field("find", "Find", "")}${field("replace", "Replace with", "")}<div class="checks">${check("case", "Case sensitive")}${check("whole", "Whole word (ASCII regex boundaries)")}${check("regex", "Regular expression (Go RE2 syntax)")}</div><p class="muted">Regex replacement supports $1 and named captures. Preview counts matches before applying.</p>`;
    getOperation = () => ({
      kind: "replace",
      find: val("find"),
      replace: val("replace"),
      caseSensitive: checked("case"),
      wholeWord: checked("whole"),
      regex: checked("regex"),
    });
  }
  if (mode === "text") {
    title = "Text & line tools";
    html = `${select("tool", "Operation", ["trim", "spaces", "blank", "upper", "lower", "sentence", "quotes", "html", "ass", "join", "wrap"], "trim")}<p class="muted">Quotes: explicitly convert straight double quotes to curly pairs. HTML / ASS removes matching tag delimiters. Case and quote tools preserve inline tags and ASS overrides. HTML removal keeps line-break separation.</p><div class="fields">${field("maxChars", "Max characters per line", prefs.maxChars, "number")}${field("maxLines", "Max lines per subtitle", prefs.maxLines, "number")}</div><p>Wrap never cuts words. If text cannot fit the configured limits, no change is applied.</p>`;
    getOperation = () => ({
      kind: "text",
      tool: val("tool"),
      maxChars: num("maxChars"),
      maxLines: num("maxLines"),
    });
  }
  if (mode === "settings") {
    title = "Local settings";
    html = `${field("outputFolder", "Output folder", prefs.outputFolder)}<button type="button" id="pick-folder">Browse…</button><div class="fields">${field("suffix", "Output suffix", prefs.suffix)}${select("encoding", "Preferred output encoding", ["UTF-8", "UTF-16LE", "Windows-1252"], prefs.encoding)}${select("default-format", "Default output format", ["source", "srt", "ass", "ssa", "vtt"], prefs.format)}${field("maxChars", "Max characters per line", prefs.maxChars, "number")}${field("maxLines", "Max lines per subtitle", prefs.maxLines, "number")}${field("cps", "CPS warning threshold", prefs.cps, "number")}</div>${check("overwrite", "Replace original files (requires a warning confirmation for each save job)", prefs.overwrite)}<p class="warning">Backup behavior: always create a unique .bak backup before replacing originals. Backups cannot be disabled.</p><p class="muted">Saved locally in your Windows application data folder. UTF-8 is recommended.</p>`;
  }
  if (mode === "save") {
    title = "Save / Export";
    html = `<p>Output folder: <strong>${esc(prefs.outputFolder || "Not selected")}</strong></p><button type="button" id="pick-save-folder">Choose output folder…</button><div class="fields">${select("save-format", "Format", ["current target", "srt", "ass", "ssa", "vtt"], prefs.format === "source" ? "current target" : prefs.format)}${select("save-encoding", "Encoding", ["automatic", "UTF-8", "UTF-16LE", "Windows-1252"], "automatic")}</div><p class="muted">Automatic uses the encoding chosen by Repair, otherwise your preferred encoding (${esc(prefs.encoding)}). Select an encoding here to override it.</p>${check("all", "Save every file in the queue")}<p class="${prefs.overwrite ? "warning" : "muted"}">${prefs.overwrite ? "Replace originals enabled. You must confirm a native warning. A backup is created first." : `New files use suffix ${esc(prefs.suffix)}. Existing output files get (2), (3), etc.`}</p><p class="warning" id="save-warnings"></p>`;
  }
  if (!["settings", "save"].includes(mode)) html += operationCommon();
  $("#dialog-title").textContent = title;
  $("#dialog-body").innerHTML = html;
  $("#dialog-hint").textContent = ["settings", "save"].includes(mode)
    ? ""
    : "Preview before applying.";
  if (mode === "sync") {
    $('[name="sync-mode"]').onchange = () => {
      for (const k of ["offset", "resync", "fps"])
        $(`#${k}-fields`).hidden = val("sync-mode") !== k;
      invalidatePreview();
    };
    for (const b of document.querySelectorAll<HTMLButtonElement>("[data-fps]"))
      b.onclick = () => {
        $<HTMLInputElement>('[name="to"]').value = b.dataset.fps!;
        invalidatePreview();
      };
  }
  if (mode === "settings")
    $("#pick-folder").onclick = () =>
      run(async () => {
        const p = await api.PickFolder();
        if (p) $<HTMLInputElement>('[name="outputFolder"]').value = p;
      });
  if (mode === "save") {
    $("#pick-save-folder").onclick = () =>
      run(async () => {
        const p = await api.PickFolder();
        if (p) {
          prefs = { ...prefs, outputFolder: p };
          await api.SaveSettings(prefs);
          $("#dialog-body p strong").textContent = p;
        }
      });
    const warning = () => {
      $("#save-warnings").textContent =
        val("save-format") === "current target"
          ? "Any pending target conversions may lose styling. Review conversion preview before saving."
          : "Cross-format conversion may lose styling, positions, cue metadata and precision. Basic bold, italic and underline are supported.";
    };
    $('[name="save-format"]').onchange = warning;
    warning();
  }
  $("#dialog-body").oninput = () => invalidatePreview();
  $<HTMLDialogElement>("#dialog").showModal();
}
function invalidatePreview() {
  if (["settings", "save"].includes(dialogMode)) return;
  previewOp = undefined;
  $("#apply-button").setAttribute("disabled", "");
  $("#preview").hidden = true;
}
$("#close-dialog").onclick = () => $<HTMLDialogElement>("#dialog").close();
$("#preview-button").onclick = () =>
  run(async () => {
    try {
      if (!path) throw new Error("Select a file");
      const o = getOperation();
      const p: Preview = await api.Preview(path, o);
      previewOp = o;
      $("#preview").hidden = false;
      $("#preview").innerHTML =
        `<h3>Preview · ${p.changed} changed entries${o.kind === "replace" ? ` · ${p.matches} matches` : ""}</h3>${p.warnings.map((w) => `<p class="warning">${esc(w)}</p>`).join("")}<div class="preview-entries">${p.examples.map((e) => `<div><small>#${e.index} · ${time(e.start)} → ${time(e.end)}</small><pre>${esc(e.text)}</pre></div>`).join("") || "<p>No timing or text changes.</p>"}</div>`;
      $("#apply-button").removeAttribute("disabled");
      $("#dialog-error").textContent = "";
    } catch (e) {
      $("#dialog-error").textContent = String(e);
    }
  });
$("#dialog-form").onsubmit = (e) => {
  e.preventDefault();
  void run(async () => {
    try {
      if (dialogMode === "settings") {
        const s: Settings = {
          outputFolder: val("outputFolder"),
          suffix: val("suffix"),
          encoding: val("encoding"),
          format: val("default-format"),
          maxChars: num("maxChars"),
          maxLines: num("maxLines"),
          cps: num("cps"),
          backup: true,
          overwrite: checked("overwrite"),
        };
        await api.SaveSettings(s);
        prefs = s;
        await api.Scan("");
        await refresh();
        await load();
        notify("Settings saved locally.");
      } else if (dialogMode === "save") {
        if (!queue.length) throw new Error("Add subtitle files first");
        const paths = checked("all") ? queue.map((f) => f.path) : [path];
        const format = val("save-format");
        if (!prefs.overwrite && !prefs.outputFolder)
          throw new Error("Choose an output folder first");
        const o =
          format === "current target"
            ? { kind: "" }
            : { kind: "convert", format };
        await api.StartBatch(
          paths,
          o,
          true,
          val("save-encoding") === "automatic" ? "" : val("save-encoding"),
        );
        results = [];
        errors = [];
        setBusy(true);
      } else {
        if (!previewOp) throw new Error("Preview the operation first");
        if (checked("all") || checked("save-batch")) {
          if (checked("save-batch") && !prefs.overwrite && !prefs.outputFolder)
            throw new Error("Set an output folder in Settings first");
          await api.StartBatch(
            checked("all") ? queue.map((f) => f.path) : [path],
            previewOp,
            checked("save-batch"),
            "",
          );
          results = [];
          errors = [];
          setBusy(true);
        } else {
          await api.Apply(path, previewOp);
          await refresh();
          await load();
          notify("Operation applied. Review the result and save when ready.");
        }
      }
      $<HTMLDialogElement>("#dialog").close();
    } catch (err) {
      $("#dialog-error").textContent = String(err);
    }
  });
};
$("#add").onclick = $("#empty-add").onclick = () =>
  run(async () => {
    await api.AddFiles();
  });
$("#folder").onclick = () =>
  run(async () => {
    await api.AddFolder();
  });
$("#scan").onclick = () =>
  run(async () => {
    await api.Scan("");
    await refresh();
    await load();
    notify("Queue scanned. No subtitle text or timing was modified.");
  });
for (const [id, mode] of [
  ["repair", "repair"],
  ["sync", "sync"],
  ["convert", "convert"],
  ["find", "find"],
  ["text-tools", "text"],
  ["settings", "settings"],
  ["save", "save"],
])
  $(`#${id}`).onclick = () => openDialog(mode);
$("#undo").onclick = () =>
  run(async () => {
    if (draft) {
      await load();
      notify("Unapplied edits discarded.");
      return;
    }
    await api.History(path, false);
    await refresh();
    await load();
    notify("Undone.");
  });
$("#redo").onclick = () =>
  run(async () => {
    await api.History(path, true);
    await refresh();
    await load();
    notify("Redone.");
  });
$("#commit").onclick = () => run(commit);
$("#select-all").onchange = () => {
  if (!view) return;
  selected.clear();
  if ($<HTMLInputElement>("#select-all").checked)
    view.doc.entries.forEach((_, i) => selected.add(i));
  capture();
  renderEntries();
};
function editAction(action: string) {
  void run(async () => {
    if (!view) throw new Error("Select a file");
    capture();
    const es = view.doc.entries;
    const ids = [...selected].sort((a, b) => a - b);
    const one = () => {
      if (ids.length !== 1) throw new Error("Select one entry");
      return ids[0];
    };
    if (action === "insert") {
      const i = ids.length ? ids.at(-1)! + 1 : es.length;
      const start = i ? es[i - 1].end : 0;
      es.splice(i, 0, {
        index: 0,
        start,
        end: start + 2000,
        text: "",
        fields: es[Math.max(0, i - 1)]?.fields?.slice(),
      });
    }
    if (action === "delete") {
      if (!ids.length) throw new Error("Select entries to delete");
      view.doc.entries = es.filter((_, i) => !selected.has(i));
    }
    if (action === "duplicate") {
      const i = one();
      es.splice(i + 1, 0, { ...es[i], fields: es[i].fields?.slice() });
    }
    if (action === "merge" || action === "split") {
      view.doc.entries = await api.EditCues(view.doc, ids, action);
    }
    if (action === "up" || action === "down") {
      const i = one();
      const j = i + (action === "up" ? -1 : 1);
      if (j < 0 || j >= es.length) throw new Error("Already at page boundary");
      [es[i], es[j]] = [es[j], es[i]];
    }
    view.doc.entries.forEach((e, i) => (e.index = page * 200 + i + 1));
    draft = true;
    selected.clear();
    renderEntries();
    $("#draft").textContent = "Unapplied edits · click Apply Edits";
  });
}
for (const id of [
  "insert",
  "delete",
  "duplicate",
  "merge",
  "split",
  "up",
  "down",
])
  $(`#${id}`).onclick = () => {
    if (!busy) editAction(id);
  };
for (const [id, delta] of [
  ["prev", -1],
  ["next", 1],
] as const)
  $(`#${id}`).onclick = () =>
    run(async () => {
      if (draft) await commit();
      page += delta;
      await load();
    });
$("#issues-tab").onclick = () => {
  $("#issues-tab").classList.add("active");
  $("#report-tab").classList.remove("active");
  showIssues();
};
$("#report-tab").onclick = () => {
  $("#report-tab").classList.add("active");
  $("#issues-tab").classList.remove("active");
  showReport();
};
for (const format of ["json", "csv"])
  $(`#report-${format}`).onclick = () =>
    run(async () => {
      const p = await api.ExportReport(format);
      if (p) notify(`Report saved: ${p}`);
    });
$("#cancel").onclick = () =>
  run(async () => {
    await api.Cancel();
    notify("Cancellation requested. Finishing the current file safely.");
  });
window.runtime.EventsOn("progress", (p) => {
  setBusy(!p.finished);
  $("#progress").setAttribute("max", String(Math.max(1, p.total)));
  $("#progress").setAttribute("value", String(p.done));
  $("#progress-label").textContent = `${p.done} / ${p.total}`;
  $("#status-text").textContent = `${p.kind}: ${p.current}`;
  if (p.error) {
    errors.push(`${p.current}: ${p.error}`);
  } else if (p.current)
    results.push(`${p.current.split(/[\\/]/).pop()}: success`);
  if (p.finished) {
    void run(async () => {
      await refresh();
      await load();
      notify(
        `${p.cancelled ? "Job cancelled" : "Job finished"} · ${p.done}/${p.total} · ${errors.length} errors${errors.length ? ": " + errors.join(" | ") : ""}`,
        errors.length > 0,
      );
      if (dialogMode === "" || !$<HTMLDialogElement>("#dialog").open) {
        $("#issue-list").insertAdjacentHTML(
          "afterbegin",
          `<details ${errors.length ? "open" : ""}><summary>Job results (${p.done}/${p.total})</summary>${[...errors, ...results].map((r) => `<p class="repair-log">${esc(r)}</p>`).join("")}</details>`,
        );
      }
    });
  }
});
window.runtime.OnFileDrop((_x, _y, paths) => {
  if (busy) return;
  void run(async () => {
    results = [];
    errors = [];
    await api.ImportPaths(paths);
  });
}, false);
document.addEventListener("keydown", (e) => {
  if (busy) return;
  const editing = (e.target as HTMLElement).matches("input,textarea,select");
  if (e.ctrlKey) {
    const key = e.key.toLowerCase();
    const ids: Record<string, string> = {
      o: "add",
      s: "save",
      z: "undo",
      y: "redo",
      f: "find",
    };
    if (ids[key] && (!editing || ["o", "s", "f"].includes(key))) {
      e.preventDefault();
      $<HTMLButtonElement>(`#${ids[key]}`).click();
    }
  } else if (
    e.key === "Delete" &&
    !editing &&
    !$<HTMLDialogElement>("#dialog").open
  ) {
    e.preventDefault();
    editAction("delete");
  }
});
void run(async () => {
  prefs = await api.GetSettings();
  await refresh();
  setBusy(false);
});

// Explicit paths complement the native file dialogs and support pasted file lists.
$("#folder").insertAdjacentHTML(
  "afterend",
  btn("paths", "Paths…", "Import files or folders by path"),
);
document.body.insertAdjacentHTML(
  "beforeend",
  `<dialog id="paths-dialog"><form id="paths-form" style="padding:22px"><h2>Import paths</h2><p>One subtitle file or folder per line. Folder import is recursive.</p><textarea id="import-paths" rows="5" aria-label="Subtitle paths" placeholder="C:\\Subtitles\\movie.srt"></textarea><div class="dialog-actions"><button type="button" id="paths-close">Cancel</button><span class="spacer"></span><button type="submit" class="accent">Import</button></div><p id="paths-error" class="error"></p></form></dialog>`,
);
$("#paths").onclick = () => {
  if (busy) return;
  $("#paths-error").textContent = "";
  $<HTMLDialogElement>("#paths-dialog").showModal();
};
$("#paths-close").onclick = () => $<HTMLDialogElement>("#paths-dialog").close();
$("#paths-form").onsubmit = (e) => {
  e.preventDefault();
  void run(async () => {
    try {
      const paths = $<HTMLTextAreaElement>("#import-paths")
        .value.split(/\r?\n/)
        .map((p) => p.trim().replace(/^"|"$/g, ""))
        .filter(Boolean);
      if (!paths.length) throw new Error("Enter at least one path");
      results = [];
      errors = [];
      await api.ImportPaths(paths);
      $<HTMLDialogElement>("#paths-dialog").close();
    } catch (err) {
      $("#paths-error").textContent = String(err);
    }
  });
};
