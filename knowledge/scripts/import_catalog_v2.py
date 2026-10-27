"""Generate the runtime catalog from the approved XLSX (standard library only)."""

from __future__ import annotations

import argparse
import hashlib
import json
from pathlib import Path
import re
from xml.etree import ElementTree as ET
from zipfile import ZipFile

PLATFORMS = {"航科君富": ("beijing_hangke", 38), "膜天君富": ("beijing_motian", 34), "聚元私募": ("neimeng_juyuan", 39)}
NS = {"s": "http://schemas.openxmlformats.org/spreadsheetml/2006/main"}


def cell_position(ref):
    letters, row = re.fullmatch(r"([A-Z]+)(\d+)", ref).groups()
    col = 0
    for char in letters:
        col = col * 26 + ord(char) - 64
    return int(row), col


def workbook_rows(path):
    with ZipFile(path) as archive:
        shared = []
        if "xl/sharedStrings.xml" in archive.namelist():
            shared = ["".join(node.text or "" for node in item.findall(".//s:t", NS)) for item in ET.fromstring(archive.read("xl/sharedStrings.xml")).findall("s:si", NS)]
        relations = {item.attrib["Id"]: item.attrib["Target"] for item in ET.fromstring(archive.read("xl/_rels/workbook.xml.rels"))}
        book = ET.fromstring(archive.read("xl/workbook.xml"))
        for sheet in book.findall("s:sheets/s:sheet", NS):
            target = relations[sheet.attrib["{http://schemas.openxmlformats.org/officeDocument/2006/relationships}id"]]
            filename = target.lstrip("/") if target.startswith("/") else "xl/" + target
            xml = ET.fromstring(archive.read(filename))
            cells = {}
            for cell in xml.findall("s:sheetData/s:row/s:c", NS):
                value = cell.find("s:v", NS)
                text = value.text if value is not None else ""
                if cell.attrib.get("t") == "s":
                    text = shared[int(text)]
                elif cell.attrib.get("t") == "inlineStr":
                    text = "".join(node.text or "" for node in cell.findall("s:is//s:t", NS))
                cells[cell_position(cell.attrib["r"])] = text or ""
            # Only real merged cells inherit values, never arbitrary blanks.
            for merged in xml.findall("s:mergeCells/s:mergeCell", NS):
                start, end = (cell_position(ref) for ref in merged.attrib["ref"].split(":"))
                value = cells.get(start, "")
                for row in range(start[0], end[0] + 1):
                    for col in range(start[1], end[1] + 1):
                        cells[row, col] = value
            yield sheet.attrib["name"], [(row, [cells.get((row, col), "") for col in range(1, 8)]) for row in range(4, max(r for r, _ in cells) + 1)]


def build_catalog(path):
    platforms = []
    for name, rows in workbook_rows(path):
        if name not in PLATFORMS:
            raise ValueError("Unexpected sheet: " + name)
        code, expected = PLATFORMS[name]
        leaves = {}
        for row_number, row in rows:
            _, level1, level2, definition, level3, standard, related_file = row
            if not level3.strip():
                if related_file.strip():
                    raise ValueError(f"{name}:{row_number} file without a leaf")
                continue
            levels = [value.strip() for value in (level1, level2, level3)]
            if not all(levels) or any("/" in level for level in levels):
                raise ValueError(f"{name}:{row_number} invalid levels")
            domain_path = "/".join(levels)
            leaf_id = code + "-" + hashlib.sha256(domain_path.encode("utf-8")).hexdigest()[:16]
            leaf = leaves.setdefault(domain_path, {
                "leaf_id": leaf_id, "level1": levels[0], "level2": levels[1], "level3": levels[2],
                "domain_path": domain_path, "level2_definition": definition.strip(),
                "classification_standards": [], "related_files": [], "file_items": [], "source_rows": [],
            })
            if definition.strip() != leaf["level2_definition"]:
                raise ValueError(f"{name}:{row_number} inconsistent definition")
            if standard.strip() and standard.strip() not in leaf["classification_standards"]:
                leaf["classification_standards"].append(standard.strip())
            if related_file.strip():
                leaf["related_files"].append(related_file.strip())
                leaf["file_items"].append({"name": related_file.strip(), "classification_standard": standard.strip(), "source_row": row_number})
            leaf["source_rows"].append(row_number)
        if len(leaves) != expected:
            raise ValueError(f"{name}: expected {expected} leaves, got {len(leaves)}")
        for leaf in leaves.values():
            leaf["classification_standard"] = "\n".join(leaf["classification_standards"])
        platforms.append({"code": code, "name": name, "type": "平台", "leaves": list(leaves.values())})
    if len(platforms) != 3:
        raise ValueError("Exactly three sheets required")
    return {"version": 2, "source_file": path.name, "source_sha256": hashlib.sha256(path.read_bytes()).hexdigest(), "platforms": platforms}


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("xlsx", type=Path)
    parser.add_argument("--output", type=Path, default=Path(__file__).resolve().parents[1] / "config" / "catalog_v2.json")
    args = parser.parse_args()
    catalog = build_catalog(args.xlsx)
    args.output.parent.mkdir(parents=True, exist_ok=True)
    args.output.write_text(json.dumps(catalog, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
    print([(platform["code"], len(platform["leaves"])) for platform in catalog["platforms"]])


if __name__ == "__main__":
    main()
