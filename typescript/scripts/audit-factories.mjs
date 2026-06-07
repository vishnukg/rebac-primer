#!/usr/bin/env node
// ─────────────────────────────────────────────────────────────────────────────
// Factory-naming audit — enforces the make*/compose* convention.
//
//   make*    : a LEAF factory. Returns one port; defines its work inline; it
//              calls NO other make*/compose* factory in its body.
//   compose* : WIRING. It builds its collaborators — it calls at least one other
//              make*/compose* factory (and/or selects a concrete adapter).
//
// Violations reported:
//   • a make*    whose body DOES call a factory  → should be compose*
//   • a compose* whose body calls NO factory      → should be make*
//
// Run:   npm run audit          (exits non-zero on any violation)
// Scope: every *.ts file under the project (excludes node_modules/dist/coverage
//        and *.d.ts). Plain functions (no make/compose prefix) are ignored, as
//        are factories declared with the `function` keyword — this repo uses
//        `const name = (...) => ...` for every factory.
// ─────────────────────────────────────────────────────────────────────────────
import { readFileSync, readdirSync, statSync } from "node:fs";
import { join, relative } from "node:path";

const ROOT = process.cwd();
const SKIP_DIRS = new Set(["node_modules", "dist", "coverage", ".git", "build", ".next"]);

const walk = (dir, acc = []) => {
  for (const name of readdirSync(dir)) {
    const p = join(dir, name);
    if (statSync(p).isDirectory()) {
      if (!SKIP_DIRS.has(name)) walk(p, acc);
    } else if (name.endsWith(".ts") && !name.endsWith(".d.ts")) {
      acc.push(p);
    }
  }
  return acc;
};

// Blank out comment bodies (preserving length + newlines) so prose like
// "calls makeFoo()" can't be mistaken for a real call, while file offsets and
// line numbers stay valid.
const blankComments = (s) =>
  s
    .replace(/\/\*[\s\S]*?\*\//g, (m) => m.replace(/[^\n]/g, " "))
    .replace(/([^:]|^)\/\/[^\n]*/g, (m, p) => p + " ".repeat(m.length - p.length));

const defRe = /(?:export\s+)?const\s+((?:make|compose)[A-Za-z0-9]*)\s*=/g;
const callRe = /\b((?:make|compose)[A-Za-z0-9]*)\s*\(/g;

// Given the index just after `const NAME =`, return the factory body text.
// Skips the parameter list by paren-matching (so a `=>` inside a callback param
// type is not mistaken for the arrow), then captures the arrow body — a
// brace-matched block, or an expression up to the terminating `;`.
const factoryBody = (src, from) => {
  const open = src.indexOf("(", from);
  if (open < 0) return "";
  let depth = 0;
  let i = open;
  for (; i < src.length; i++) {
    if (src[i] === "(") depth++;
    else if (src[i] === ")" && --depth === 0) {
      i++;
      break;
    }
  }
  const arrow = src.indexOf("=>", i);
  if (arrow < 0) return "";
  let j = arrow + 2;
  while (j < src.length && /\s/.test(src[j])) j++;
  if (src[j] === "{") {
    let d = 0;
    for (let k = j; k < src.length; k++) {
      if (src[k] === "{") d++;
      else if (src[k] === "}" && --d === 0) return src.slice(j, k + 1);
    }
    return src.slice(j);
  }
  let d = 0;
  for (let k = j; k < src.length; k++) {
    const c = src[k];
    if ("([{".includes(c)) d++;
    else if (")]}".includes(c)) d--;
    else if (c === ";" && d <= 0) return src.slice(j, k);
  }
  return src.slice(j);
};

const lineOf = (src, idx) => src.slice(0, idx).split("\n").length;

const violations = [];
for (const file of walk(ROOT)) {
  const src = blankComments(readFileSync(file, "utf8"));
  for (const m of src.matchAll(defRe)) {
    const name = m[1];
    const body = factoryBody(src, m.index + m[0].length);
    const calls = [...new Set([...body.matchAll(callRe)].map((c) => c[1]))].filter((c) => c !== name);
    const isMake = name.startsWith("make");
    if (isMake && calls.length > 0) {
      violations.push({
        file,
        line: lineOf(src, m.index),
        name,
        msg: `make* but its body builds collaborators ${JSON.stringify(calls)} → rename to compose*`,
      });
    } else if (!isMake && calls.length === 0) {
      violations.push({
        file,
        line: lineOf(src, m.index),
        name,
        msg: `compose* but its body calls no factory → it builds nothing, rename to make*`,
      });
    }
  }
}

if (violations.length === 0) {
  console.log("✓ factory audit passed — every make* is a leaf, every compose* wires.");
  process.exit(0);
}

console.error(`✗ factory naming audit: ${violations.length} violation(s)\n`);
for (const v of violations) {
  console.error(`  ${relative(ROOT, v.file)}:${v.line}  ${v.name}`);
  console.error(`      ${v.msg}\n`);
}
console.error("Rule: make* = leaf factory (one port, logic inline, calls no factory);");
console.error("      compose* = wiring (calls make*/compose* and/or selects an adapter).");
process.exit(1);
