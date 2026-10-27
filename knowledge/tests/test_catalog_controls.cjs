const assert = require("node:assert/strict");
const fs = require("node:fs");
const path = require("node:path");
const controls = require("../static/catalog-controls.js");
const catalog = JSON.parse(fs.readFileSync(path.join(__dirname, "../config/catalog_v2.json"), "utf8"));
let checks = 0;
for (const platform of catalog.platforms) {
  assert.deepEqual(controls.model(platform.leaves).disabled, [false, true, true]);
  for (const leaf of platform.leaves) {
    let selection = ["", "", ""];
    selection = controls.change(selection, 0, leaf.level1);
    assert.equal(controls.model(platform.leaves, selection).disabled[2], true);
    selection = controls.change(selection, 1, `${leaf.level1}/${leaf.level2}`);
    assert.ok(controls.model(platform.leaves, selection).options[2].includes(leaf.domain_path));
    selection = controls.change(selection, 2, leaf.domain_path);
    assert.equal(controls.model(platform.leaves, selection).leaf.leaf_id, leaf.leaf_id);
    assert.equal(controls.prefix(selection), leaf.domain_path);
    assert.deepEqual(controls.change(selection, 0, ""), ["", "", ""]);
    assert.deepEqual(controls.change(selection, 1, ""), [leaf.level1, "", ""]);
    checks += 6;
  }
}
console.log(`PASS: ${checks} cascade assertions across 111 leaves and 3 platforms`);
