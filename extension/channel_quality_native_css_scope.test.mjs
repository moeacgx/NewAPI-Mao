import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { resolve } from "node:path";
import { test } from "node:test";
import { fileURLToPath } from "node:url";

const root = resolve(
  process.argv[2] ||
    fileURLToPath(new URL("./builtin/channel-quality/public", import.meta.url)),
);
const scopeStart =
  /^(?::where\(\.channel-quality-native\)|\.channel-quality-native)(?![\w-])/;

function parse(css, rules = []) {
  let index = 0;
  while (index < css.length) {
    const open = css.indexOf("{", index);
    const semicolon = css.indexOf(";", index);
    if (open < 0) break;
    if (semicolon >= 0 && semicolon < open) {
      rules.push(css.slice(index, semicolon).trim());
      index = semicolon + 1;
      continue;
    }
    let depth = 1;
    let end = open + 1;
    for (; depth > 0 && end < css.length; end++) {
      if (css[end] === "{") depth++;
      if (css[end] === "}") depth--;
    }
    const prelude = css.slice(index, open).trim();
    rules.push(prelude);
    if (/^@(layer|media|supports)\b/.test(prelude))
      parse(css.slice(open + 1, end - 1), rules);
    index = end;
  }
  return rules;
}

function selectors(prelude) {
  const list = [];
  let depth = 0;
  let current = "";
  for (let i = 0; i < prelude.length; i++) {
    const char = prelude[i];
    if (char === "\\") {
      current += char + (prelude[++i] ?? "");
      continue;
    }
    if (char === "(" || char === "[") depth++;
    if (char === ")" || char === "]") depth--;
    if (char === "," && depth === 0) {
      list.push(current.trim());
      current = "";
      continue;
    }
    current += char;
  }
  list.push(current.trim());
  return list;
}

for (const target of ["default", "classic"]) {
  const css = readFileSync(
    resolve(root, "native", target + ".css"),
    "utf8",
  ).replace(/\/\*[\s\S]*?\*\//g, "");
  const rules = parse(css);
  test(
    target +
      " 原生样式只输出作用域内选择器，不覆盖宿主顶栏 .w-full/.flex/.block",
    () => {
      const leaked = rules
        .filter((prelude) => !prelude.startsWith("@"))
        .flatMap(selectors)
        .filter((selector) => !scopeStart.test(selector));
      assert.deepEqual([...new Set(leaked)].slice(0, 20), []);
    },
  );
  test(target + " 原生样式不注册全局 @property 与 @keyframes", () => {
    const globals = rules.filter(
      (prelude) =>
        /^@(property|keyframes)\b/.test(prelude) &&
        !prelude.includes("channel-quality-"),
    );
    assert.deepEqual(globals.slice(0, 20), []);
  });
  test(target + " 入口根节点携带作用域类", () => {
    const entry = readFileSync(
      resolve(root, "native", target + ".mjs"),
      "utf8",
    );
    assert.match(entry, /className:"[^"]*\bchannel-quality-native\b/);
  });
}
