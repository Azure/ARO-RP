#!/usr/bin/env python3

# Copyright (c) Microsoft Corporation.
# Licensed under the Apache License 2.0.

"""Extract every CloudError call site from ARO-RP using gopls LSP.

Drives gopls to find all api.NewCloudError and api.WriteError references,
reads source at each call site to extract arguments, and resolves Go constants
via gopls hover (cached per token) with a pre-built table for CloudErrorCode*
constants as a fast path.

Output: Markdown table to stdout, progress to stderr.

Rows that contain <token> placeholders represent call sites where an argument
is a runtime-determined variable (e.g., a function parameter or a value
computed from a condition). These cannot be statically resolved and are shown
verbatim so the caller can locate the source and inspect it directly.

Usage:
    make generate-cloud-errors
    python3 hack/extract-cloud-errors.py [repo_root] > docs/clouderrors.md
"""

import json
import os
import re
import subprocess
import sys
from pathlib import Path
from typing import Optional

REPO_ROOT = Path(sys.argv[1]).resolve() if len(sys.argv) > 1 else Path(__file__).parent.parent.resolve()


def build_error_code_table(repo_root: Path) -> dict[str, str]:
    """Parse CloudErrorCode* constants from pkg/api/error.go."""
    text = (repo_root / "pkg" / "api" / "error.go").read_text()
    table: dict[str, str] = {}
    # Matches both typed (CloudErrorCodeFoo CloudErrorCode = "Bar")
    # and untyped (CloudErrorCodeFoo = "Bar") forms.
    for m in re.finditer(r'(CloudErrorCode\w+)(?:\s+\w+)?\s*=\s*"([^"]*)"', text):
        sym, val = m.group(1), m.group(2)
        table[sym] = val
        table[f"api.{sym}"] = val
    return table


# ── LSP client ─────────────────────────────────────────────────────────────────

class LSP:
    def __init__(self, root: Path):
        self.root = root
        self._id = 0
        self._opened: set[Path] = set()
        self._hover_cache: dict[str, Optional[str]] = {}
        self.proc = subprocess.Popen(
            [os.environ.get("GOPLS", "gopls")],
            stdin=subprocess.PIPE,
            stdout=subprocess.PIPE,
            stderr=subprocess.DEVNULL,
            cwd=root,
        )
        self._rpc("initialize", {
            "processId": os.getpid(),
            "rootUri": root.as_uri(),
            "capabilities": {},
            "workspaceFolders": [{"uri": root.as_uri(), "name": "root"}],
        })
        self._notify("initialized", {})

    def _send(self, msg: dict):
        body = json.dumps(msg).encode()
        self.proc.stdin.write(f"Content-Length: {len(body)}\r\n\r\n".encode() + body)
        self.proc.stdin.flush()

    def _recv(self) -> dict:
        headers: dict[str, str] = {}
        while True:
            line = self.proc.stdout.readline().decode()
            if line in ("\r\n", "\n", ""):
                break
            if ":" in line:
                k, _, v = line.partition(":")
                headers[k.strip()] = v.strip()
        if "Content-Length" not in headers:
            raise RuntimeError("gopls closed stdout unexpectedly")
        body = self.proc.stdout.read(int(headers["Content-Length"]))
        return json.loads(body)

    def _rpc(self, method: str, params) -> dict:
        self._id += 1
        rid = self._id
        self._send({"jsonrpc": "2.0", "id": rid, "method": method, "params": params})
        while True:
            msg = self._recv()
            if msg.get("id") == rid:
                return msg
            # Discard notifications (window/logMessage, etc.)

    def _notify(self, method: str, params):
        self._send({"jsonrpc": "2.0", "method": method, "params": params})

    def _open(self, path: Path):
        if path not in self._opened:
            self._notify("textDocument/didOpen", {
                "textDocument": {
                    "uri": path.as_uri(),
                    "languageId": "go",
                    "version": 1,
                    "text": path.read_text(),
                }
            })
            self._opened.add(path)

    def references(self, path: Path, line: int, char: int) -> list[dict]:
        self._open(path)
        resp = self._rpc("textDocument/references", {
            "textDocument": {"uri": path.as_uri()},
            "position": {"line": line, "character": char},
            "context": {"includeDeclaration": False},
        })
        return resp.get("result") or []

    def hover(self, path: Path, line: int, char: int) -> str:
        self._open(path)
        resp = self._rpc("textDocument/hover", {
            "textDocument": {"uri": path.as_uri()},
            "position": {"line": line, "character": char},
        })
        result = resp.get("result") or {}
        c = result.get("contents", "")
        if isinstance(c, dict):
            return c.get("value", "")
        if isinstance(c, list):
            return "\n".join(x.get("value", x) if isinstance(x, dict) else str(x) for x in c)
        return str(c)

    def definition(self, path: Path, line: int, char: int) -> Optional[tuple[Path, int, int]]:
        """Return the (file, line, char) of the symbol's declaration, or None."""
        self._open(path)
        resp = self._rpc("textDocument/definition", {
            "textDocument": {"uri": path.as_uri()},
            "position": {"line": line, "character": char},
        })
        result = resp.get("result")
        if not result:
            return None
        # Result may be a Location, []Location, or []LocationLink
        loc = result[0] if isinstance(result, list) else result
        uri = loc.get("uri") or loc.get("targetUri", "")
        rng = loc.get("range") or loc.get("targetSelectionRange", {})
        if not uri or not rng:
            return None
        def_path = Path(uri.removeprefix("file://"))
        def_line = rng["start"]["line"]
        def_char = rng["start"]["character"]
        return def_path, def_line, def_char

    def shutdown(self):
        self._rpc("shutdown", None)
        self._notify("exit", None)
        try:
            self.proc.wait(timeout=5)
        except subprocess.TimeoutExpired:
            self.proc.kill()
            self.proc.wait()


# ── call-site discovery ────────────────────────────────────────────────────────

def find_call_sites(repo_root: Path, func_name: str, skip_file: Path) -> list[tuple[Path, int, int]]:
    """Scan all .go files for call sites of func_name.

    Returns (file, line, char) tuples pointing to the start of the identifier.
    Uses text search rather than LSP references so that both Go modules in the
    repo (root and pkg/api) are covered; gopls references only returns callers
    within the same module as the declaration.
    """
    # Match the identifier but not its own func declaration
    call_re = re.compile(rf'(?<!func )\b{re.escape(func_name)}\s*\(')
    sites: list[tuple[Path, int, int]] = []
    for go_file in sorted(repo_root.rglob("*.go")):
        if go_file == skip_file or go_file.name.endswith("_test.go") or "vendor" in go_file.parts:
            continue
        try:
            lines = go_file.read_text().splitlines()
        except Exception:
            continue
        for line_idx, line in enumerate(lines):
            for m in call_re.finditer(line):
                sites.append((go_file, line_idx, m.start()))
    return sites


# ── argument extractor ─────────────────────────────────────────────────────────

def extract_args(
    source_lines: list[str], start_line: int, start_char: int
) -> Optional[tuple[list[str], list[tuple[int, int]]]]:
    """Starting from the position of the function identifier, extract its
    argument list.

    Returns (arg_strings, arg_positions) where arg_positions[i] is the
    (line, char) of the first non-whitespace character of arg i, suitable
    for LSP hover requests.
    """
    # Build a flat list of (char, line, col) starting from start_line col 0.
    chars: list[tuple[str, int, int]] = []
    for li in range(start_line, min(start_line + 40, len(source_lines))):
        for ci, ch in enumerate(source_lines[li]):
            chars.append((ch, li, ci))
        chars.append(("\n", li, len(source_lines[li])))

    # Find the opening '(' of the call, scanning forward from start_char.
    open_idx: Optional[int] = None
    for i in range(start_char, len(chars)):
        if chars[i][0] == "(":
            open_idx = i
            break
    if open_idx is None:
        return None

    # Walk the balanced content, collecting args and their start positions.
    args: list[str] = []
    positions: list[tuple[int, int]] = []

    depth = 0
    in_str = False
    str_char = ""
    seg: list[str] = []
    seg_pos: Optional[tuple[int, int]] = None

    i = open_idx
    while i < len(chars):
        ch, li, ci = chars[i]

        if in_str:
            seg.append(ch)
            if str_char != "`" and ch == "\\" and i + 1 < len(chars):
                i += 1
                seg.append(chars[i][0])
            elif ch == str_char:
                in_str = False
            i += 1
            continue

        if ch in ('"', "`", "'"):
            in_str = True
            str_char = ch
            if seg_pos is None:
                seg_pos = (li, ci)
            seg.append(ch)

        elif ch in ("(", "[", "{"):
            if depth == 0:
                # This is the opening paren of the call itself; skip it.
                depth += 1
                i += 1
                continue
            depth += 1
            if seg_pos is None:
                seg_pos = (li, ci)
            seg.append(ch)

        elif ch in (")", "]", "}"):
            depth -= 1
            if depth == 0:
                # Closing paren of the call: finalize last arg.
                args.append("".join(seg).strip())
                positions.append(seg_pos or (li, ci))
                break
            seg.append(ch)

        elif ch == "," and depth == 1:
            args.append("".join(seg).strip())
            positions.append(seg_pos or (li, ci))
            seg = []
            seg_pos = None

        else:
            if ch not in (" ", "\t", "\n", "\r") and seg_pos is None:
                seg_pos = (li, ci)
            seg.append(ch)

        i += 1

    return (args, positions) if args else None


# ── constant resolution ────────────────────────────────────────────────────────

def hover_value(lsp: LSP, token: str, path: Path, line: int, char: int) -> Optional[str]:
    """Ask gopls for a constant's value via hover, with a per-token cache.

    For qualified names like http.StatusBadRequest, hover at the identifier
    after the dot; hovering at 'h' returns package info, not the constant.
    """
    if token in lsp._hover_cache:
        return lsp._hover_cache[token]
    # Advance char to point at the final identifier component (after last '.')
    dot = token.rfind(".")
    if dot >= 0:
        char += dot + 1
    text = lsp.hover(path, line, char)
    result = None
    if text:
        # Integer constant: "= 400" or "StatusBadRequest = 400"
        m = re.search(r"=\s*(\d+)", text)
        if m:
            result = m.group(1)
        else:
            # String constant: = "InternalServerError"
            m = re.search(r'=\s*"([^"]*)"', text)
            if m:
                result = f'"{m.group(1)}"'
    lsp._hover_cache[token] = result
    return result


def _resolve_via_definition(
    lsp: LSP, path: Path, call_pos: tuple[int, int], var_name: str,
    source_lines: Optional[list[str]] = None,
) -> Optional[str]:
    """Resolve a variable's string value by navigating to its definition via LSP.

    Strategy:
    1. Find var_name in the source near call_pos (it appears as an arg in the call).
    2. Use textDocument/definition to jump to its declaration.
    3. Hover at the declaration; gopls includes the initializer text there.
    4. Parse the string value from the hover text.
    """
    # Find the position of var_name in the source near the call site
    try:
        lines = source_lines if source_lines is not None else path.read_text().splitlines()
    except Exception:
        return None
    pattern = re.compile(rf'\b{re.escape(var_name)}\b')
    var_pos: Optional[tuple[int, int]] = None
    for li in range(call_pos[0], min(call_pos[0] + 15, len(lines))):
        search_from = call_pos[1] if li == call_pos[0] else 0
        m = pattern.search(lines[li], search_from)
        if m:
            var_pos = (li, m.start())
            break
    if var_pos is None:
        return None

    # Navigate to the declaration
    decl = lsp.definition(path, var_pos[0], var_pos[1])
    if decl is None:
        return None
    decl_path, decl_line, decl_char = decl

    # Read the declaration line directly; gopls hover only shows type for var,
    # not the initializer value, so we read the source instead.
    try:
        decl_source = decl_path.read_text().splitlines()[decl_line]
    except Exception:
        return None

    m = re.search(r'=\s*("(?:[^"\\]|\\.)*"|`[^`]*`)', decl_source)
    if m:
        s = m.group(1)
        return s[1:-1]
    return None


def resolve(
    token: str,
    lsp: LSP,
    path: Path,
    pos: tuple[int, int],
    code_table: dict[str, str],
    source_lines: Optional[list[str]] = None,
) -> str:
    t = re.sub(r"\s+", " ", token.strip())

    # String literal
    if (t.startswith('"') and t.endswith('"')) or (t.startswith("`") and t.endswith("`")):
        return t[1:-1]

    # Bare integer
    if re.fullmatch(r"\d+", t):
        return t

    # fmt.Sprintf with a string literal format string
    m = re.match(r'fmt\.Sprintf\s*\(\s*("(?:[^"\\]|\\.)*")', t)
    if m:
        inner = m.group(1)
        return f"Sprintf:{inner[1:-1]}"

    # fmt.Sprintf with a variable format string: navigate to definition then hover
    m = re.match(r'fmt\.Sprintf\s*\(\s*(\w+)\s*,', t)
    if m:
        var_name = m.group(1)
        val = _resolve_via_definition(lsp, path, pos, var_name, source_lines)
        if val is not None:
            return f"Sprintf:{val}"

    # CloudErrorCode* fast path (avoids a hover round-trip for the common case)
    if t in code_table:
        return f'"{code_table[t]}"'

    # All other identifiers (http.Status*, variables, etc.): ask gopls
    resolved = hover_value(lsp, t, path, pos[0], pos[1])
    if resolved:
        return resolved

    # Give up: return the raw token so the caller can see what it is
    return f"<{t}>"


# ── main ───────────────────────────────────────────────────────────────────────

def _is_admin(sources: list[str]) -> bool:
    """Return True if any source path is admin-only.

    Matches /admin/ subdirectories, pkg/frontend/admin_*.go files, and
    pkg/frontend/adminactions/. Does not match pkg/frontend/adminreplies.go.
    """
    return any(
        "/admin/" in s or "/frontend/admin_" in s or "/frontend/adminactions/" in s
        for s in sources
    )


def main():
    error_go = REPO_ROOT / "pkg" / "api" / "error.go"
    code_table = build_error_code_table(REPO_ROOT)

    results: list[dict] = []

    # (func_name, arg_offset): WriteError has a leading 'w http.ResponseWriter'
    # that we skip; NewCloudError args start at index 0.
    # Note: we use file scanning rather than LSP textDocument/references because
    # gopls only returns callers within the same module as the declaration, and
    # NewCloudError lives in pkg/api (a separate Go module from the repo root).
    targets = [
        ("NewCloudError", 0),
        ("WriteError",    1),
    ]

    lsp = None
    try:
        print("Starting gopls...", file=sys.stderr)
        lsp = LSP(REPO_ROOT)
        print("Ready.", file=sys.stderr)

        for func_name, arg_offset in targets:
            print(f"Scanning for {func_name} call sites...", file=sys.stderr)
            sites = find_call_sites(REPO_ROOT, func_name, skip_file=error_go)
            print(f"  {len(sites)} call sites found", file=sys.stderr)

            for ref_path, ref_line, ref_char in sites:
                try:
                    source_lines = ref_path.read_text().splitlines()
                except Exception as e:
                    print(f"  Cannot read {ref_path}: {e}", file=sys.stderr)
                    continue

                parsed = extract_args(source_lines, ref_line, ref_char)
                if parsed is None:
                    print(f"  Parse failed: {ref_path}:{ref_line + 1}", file=sys.stderr)
                    continue

                all_args, all_positions = parsed
                args = all_args[arg_offset:]
                positions = all_positions[arg_offset:]

                if len(args) < 4:
                    print(
                        f"  Too few args ({len(args)}) at {ref_path}:{ref_line + 1}",
                        file=sys.stderr,
                    )
                    continue

                status  = resolve(args[0], lsp, ref_path, positions[0], code_table, source_lines)
                code    = resolve(args[1], lsp, ref_path, positions[1], code_table, source_lines)
                target  = resolve(args[2], lsp, ref_path, positions[2], code_table, source_lines)
                message = resolve(args[3], lsp, ref_path, positions[3], code_table, source_lines)

                results.append({
                    "file":        str(ref_path.relative_to(REPO_ROOT)),
                    "line":        ref_line + 1,
                    "func":        func_name,
                    "status_code": status,
                    "error_code":  code,
                    "target":      target,
                    "message":     message,
                })
    finally:
        if lsp is not None:
            lsp.shutdown()

    # Deduplicate: group by (status_code, error_code, target, message),
    # collecting all source locations into a list.
    groups: dict[tuple, list[str]] = {}
    for r in results:
        key = (r["status_code"], r["error_code"], r["target"], r["message"])
        groups.setdefault(key, []).append(f"`{r['file']}:{r['line']}`")

    def sort_key(kv):
        return (kv[0][0], kv[0][1], kv[0][3])
    rows = sorted(groups.items(), key=sort_key)

    # Partition: a row is "admin" if any source path is under /admin/ or is an admin_*.go / adminactions/ file
    regular = [(k, v) for k, v in rows if not _is_admin(v)]
    admin   = [(k, v) for k, v in rows if     _is_admin(v)]

    commit = subprocess.run(
        ["git", "rev-parse", "--short", "HEAD"],
        capture_output=True, text=True, cwd=REPO_ROOT,
    ).stdout.strip() or "unknown"

    def _print_table(section_rows):
        print("| Status | Code | Message | Sources |")
        print("|--------|------|---------|---------|")
        for (status, code, target, message), sources in section_rows:
            msg = _render_message(message, target)
            code_display = f"`{code.strip(chr(34))}`"
            sources_cell = ", ".join(sources)
            print(f"| {status} | {code_display} | {msg} | {sources_cell} |")

    print("# CloudErrors\n")
    print(f"_Autogenerated by `hack/extract-cloud-errors.py` based on commit {commit}. Do not edit._\n")
    _print_table(regular)
    print("\n## Admin CloudErrors\n")
    _print_table(admin)
    print(f"\n_{len(rows)} distinct errors ({len(regular)} user-facing, {len(admin)} admin) across {len(results)} call sites._")
    print(f"\n{len(rows)} distinct errors, {len(results)} call sites written.", file=sys.stderr)


def _render_message(message: str, target: str) -> str:
    """Produce a human-readable message cell.

    - Strips the 'Sprintf:' prefix from both message and target.
    - Substitutes all printf-style format verbs in the message with <target>,
      embedding the target directly so no separate annotation is needed.
    - Shows (target: ...) annotation only when the message has no format verbs
      and a non-empty target exists.
    """
    msg = message.removeprefix("Sprintf:")
    tgt = target.removeprefix("Sprintf:")

    if "%" in msg and tgt:
        # Embed target in place of each format verb.
        # If target is already wrapped in <>, use it bare; otherwise add <>.
        subst = tgt if tgt.startswith("<") else f"<{tgt}>"
        msg = re.sub(r"%[-+#0-9.*]*[a-zA-Z]", f"`{subst}`", msg)
    elif tgt:
        msg += f" (target: `{tgt}`)"

    # Wrap any remaining bare <...> tokens in backticks for markdown code rendering
    msg = re.sub(r"(?<!`)<([^>]+)>(?!`)", r"`<\1>`", msg)

    return msg


if __name__ == "__main__":
    main()
