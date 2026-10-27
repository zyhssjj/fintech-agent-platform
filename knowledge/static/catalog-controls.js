/* Shared deterministic cascade logic, also exercised by Node regression tests. */
const CatalogControls = (() => {
  function model(leaves, selection = ["", "", ""]) {
    const roots = [...new Set(leaves.map((leaf) => leaf.level1))];
    const secondary = [...new Set(leaves.filter((leaf) => leaf.level1 === selection[0]).map((leaf) => `${leaf.level1}/${leaf.level2}`))];
    const third = leaves.filter((leaf) => `${leaf.level1}/${leaf.level2}` === selection[1]).map((leaf) => leaf.domain_path);
    return { options: [roots, secondary, third], disabled: [false, !selection[0], !selection[1]], leaf: leaves.find((leaf) => leaf.domain_path === selection[2]) || null };
  }
  function change(selection, depth, value) {
    return selection.map((item, index) => index === depth ? value : index > depth ? "" : item);
  }
  function prefix(selection) { return selection[2] || selection[1] || selection[0] || ""; }
  return { model, change, prefix };
})();
if (typeof module !== "undefined") module.exports = CatalogControls;
