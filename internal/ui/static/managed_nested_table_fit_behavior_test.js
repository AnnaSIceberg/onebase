"use strict";

const assert = require("node:assert/strict");
const fs = require("node:fs");
const path = require("node:path");
const test = require("node:test");
const vm = require("node:vm");

const source = fs.readFileSync(path.join(__dirname, "managed.js"), "utf8");
const start = source.indexOf("function obFitSiblingTabs(");
const end = source.indexOf("\nobManagedReady(", start);
assert.ok(start >= 0 && end > start, "managed table fit functions must be loaded");

function fixture() {
  const main = {
    clientHeight: 720, scrollHeight: 735, scrollTop: 0,
    children: [{getBoundingClientRect: () => ({bottom: 720})}],
    getBoundingClientRect: () => ({top: 0}),
  };
  const top = 110;
  const headers = 34;
  const note = 60;
  const action = 28;
  const tabs = {getAttribute: () => null, removeAttribute: () => {}, closest: () => main};
  const outer = {
    style: {height: "430px"},
    paddingBottom: 8,
    get clientHeight() { return parseFloat(this.style.height); },
    parentElement: {closest: () => null},
    getBoundingClientRect: () => ({top}),
    children: [{getBoundingClientRect: () => ({bottom: inner.bottom + action})}],
  };
  const inner = {
    style: {height: ""},
    paddingBottom: 8,
    get clientHeight() { return parseFloat(this.style.height); },
    get bottom() { return top + headers + (this.clientHeight || grid.offsetHeight + note); },
    parentElement: {closest: () => outer},
    getBoundingClientRect: () => ({top: top + headers}),
    children: [{getBoundingClientRect: () => ({bottom: top + headers + grid.offsetHeight + note})}],
  };
  const grid = {
    style: {height: ""},
    get offsetHeight() { return parseFloat(this.style.height) || 120; },
    offsetParent: inner,
    getAttribute: () => "work",
    closest: selector => selector === "main" ? main : inner,
  };
  let canvasResizes = 0;
  const context = vm.createContext({
    window: {
      getComputedStyle: node => ({paddingBottom: (node.paddingBottom || 0) + "px"}),
      _obGrids: {work: {div: grid, grid: {resizeCanvas: () => { canvasResizes++; }}}},
    },
    document: {
      querySelectorAll(selector) {
        if (selector === '[data-ob-fit-window="1"]') return [tabs];
        if (selector === '[data-ob-fit-grid="1"]') return [grid];
        return [];
      },
    },
    obFitTabsToWindow: () => { outer.style.height = "430px"; },
    obEqualizeTabs: () => {},
    obManagedOwnTabParts: () => [outer],
  });
  vm.runInContext(source.slice(start, end), context);
  return {
    outer, inner, grid,
    fit: () => context.obFitGridToWindow(grid),
    reflow: () => context.obEqualizeAllTabs(),
    canvasResizes: () => canvasResizes,
  };
}

test("nested grid fills the fitted outer page and reserves its note and action", () => {
  const f = fixture();
  f.fit();
  assert.equal(f.grid.style.height, "300px");
  f.fit();
  assert.equal(f.canvasResizes(), 1);
});

test("an explicitly fitted inner page takes precedence over the outer page", () => {
  const f = fixture();
  f.inner.style.height = "230px";
  f.fit();
  assert.equal(f.grid.style.height, "162px");
});

test("trimming the outer page also refits its nested grid", () => {
  const f = fixture();
  f.reflow();
  assert.equal(f.outer.style.height, "415px");
  assert.equal(f.grid.style.height, "285px");
  f.reflow();
  assert.equal(f.canvasResizes(), 1);
});

test("without a fitted ancestor use main; do not resize a hidden grid", () => {
  const f = fixture();
  f.outer.style.height = "";
  f.fit();
  assert.equal(f.grid.style.height, "120px");
  f.grid.offsetParent = null;
  f.outer.style.height = "430px";
  f.fit();
  assert.equal(f.grid.style.height, "120px");
  assert.equal(f.canvasResizes(), 1);
});
