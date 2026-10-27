"""Local-only PDF text extraction with an offline OCR fallback."""

import math
import os
import re
import threading
from pathlib import Path

import numpy as np
from pypdf import PdfReader

try:
    import pymupdf
except ImportError:  # Keeps text PDFs usable until optional server deps are installed.
    pymupdf = None


PROJECT_DIR = Path(__file__).resolve().parent.parent


def positive_env_int(name, default):
    try:
        return max(1, int(os.getenv(name, default)))
    except (TypeError, ValueError):
        return default


def resolve_project_path(value):
    path = Path(value).expanduser()
    return path if path.is_absolute() else PROJECT_DIR / path


PDF_MAX_PAGES = positive_env_int("PDF_MAX_PAGES", 500)
PDF_OCR_MAX_PAGES = positive_env_int("PDF_OCR_MAX_PAGES", 200)
PDF_OCR_DPI = positive_env_int("PDF_OCR_DPI", 144)
PDF_OCR_MAX_PIXELS = positive_env_int("PDF_OCR_MAX_PIXELS", 16_000_000)
PDF_OCR_TILE_HEIGHT = positive_env_int("PDF_OCR_TILE_HEIGHT", 4_000)
PDF_OCR_TILE_OVERLAP = positive_env_int("PDF_OCR_TILE_OVERLAP", 160)
PDF_MIN_PAGE_TEXT = positive_env_int("PDF_MIN_PAGE_TEXT", 30)
OCR_MODEL_DIR = resolve_project_path(
    os.getenv("OCR_MODELS_DIR")
    or str(Path(os.getenv("MODELS_DIR", "models")) / "rapidocr")
)
OCR_MODEL_FILES = (
    "ch_PP-OCRv4_det_mobile.onnx",
    "ch_ppocr_mobile_v2.0_cls_mobile.onnx",
    "ch_PP-OCRv4_rec_mobile.onnx",
)
OCR_MIN_CONFIDENCE = 0.55

_ocr_engine = None
_ocr_lock = threading.RLock()


class PDFParseError(RuntimeError):
    """A PDF cannot be converted into usable local text."""


def normalize_text(value):
    lines = [re.sub(r"\s+", " ", line).strip() for line in value.splitlines()]
    return "\n".join(line for line in lines if line)


def pypdf_page_texts(path):
    try:
        reader = PdfReader(str(path))
        if reader.is_encrypted:
            try:
                unlocked = bool(reader.decrypt(""))
            except Exception:
                unlocked = False
            if not unlocked:
                raise PDFParseError("PDF已加密，请先解除密码或访问限制后再入库")
        if len(reader.pages) > PDF_MAX_PAGES:
            raise PDFParseError(
                "PDF共{0}页，超过系统允许的{1}页".format(
                    len(reader.pages), PDF_MAX_PAGES
                )
            )
        return [normalize_text(page.extract_text() or "") for page in reader.pages]
    except PDFParseError:
        raise
    except Exception:
        return None


def local_ocr_engine():
    global _ocr_engine
    with _ocr_lock:
        if _ocr_engine is not None:
            return _ocr_engine
        missing = [
            name for name in OCR_MODEL_FILES if not (OCR_MODEL_DIR / name).is_file()
        ]
        if missing:
            raise PDFParseError(
                "本地OCR模型缺失：{0}；请将服务器OCR模型部署到 {1}".format(
                    "、".join(missing), OCR_MODEL_DIR
                )
            )
        try:
            from rapidocr import RapidOCR
        except ImportError as error:
            raise PDFParseError(
                "服务器尚未安装本地OCR组件，请重新安装项目 requirements.txt"
            ) from error
        try:
            _ocr_engine = RapidOCR(
                params={
                    "Global.model_root_dir": str(OCR_MODEL_DIR),
                    "Global.log_level": "warning",
                    "Global.max_side_len": 2500,
                }
            )
        except Exception as error:
            raise PDFParseError(
                "本地OCR初始化失败：{0}".format(error)
            ) from error
        return _ocr_engine


def render_page_tiles(page):
    base_scale = PDF_OCR_DPI / 72
    width_pixels = max(1, math.ceil(page.rect.width * base_scale))
    height_pixels = max(1, math.ceil(page.rect.height * base_scale))
    tile_height = min(PDF_OCR_TILE_HEIGHT, height_pixels)
    if width_pixels * tile_height > PDF_OCR_MAX_PIXELS:
        base_scale *= math.sqrt(
            PDF_OCR_MAX_PIXELS / (width_pixels * tile_height)
        )
        width_pixels = max(1, math.ceil(page.rect.width * base_scale))
        height_pixels = max(1, math.ceil(page.rect.height * base_scale))
        tile_height = min(PDF_OCR_TILE_HEIGHT, height_pixels)
    if width_pixels * tile_height > PDF_OCR_MAX_PIXELS:
        tile_height = max(512, PDF_OCR_MAX_PIXELS // width_pixels)

    overlap = min(PDF_OCR_TILE_OVERLAP, max(0, tile_height // 4))
    step = max(1, tile_height - overlap)
    top = 0
    while top < height_pixels:
        bottom = min(height_pixels, top + tile_height)
        clip = pymupdf.Rect(
            0,
            top / base_scale,
            page.rect.width,
            bottom / base_scale,
        )
        pixmap = page.get_pixmap(
            matrix=pymupdf.Matrix(base_scale, base_scale),
            clip=clip,
            colorspace=pymupdf.csRGB,
            alpha=False,
        )
        image = np.frombuffer(pixmap.samples, dtype=np.uint8).reshape(
            pixmap.height, pixmap.width, pixmap.n
        )
        yield image, top, width_pixels, height_pixels
        if bottom >= height_pixels:
            break
        top += step


def deduplicate_ocr_lines(lines):
    kept = []
    for line in sorted(lines, key=lambda item: (item["top"], item["left"])):
        duplicate = any(
            previous["text"] == line["text"]
            and abs(previous["top"] - line["top"]) <= PDF_OCR_TILE_OVERLAP * 1.5
            and abs(previous["left"] - line["left"]) <= 80
            for previous in kept[-30:]
        )
        if not duplicate:
            kept.append(line)
    return kept


def order_column_segment(lines, page_width):
    if not lines:
        return []
    threshold = max(80, page_width * 0.16)
    columns = []
    for line in sorted(lines, key=lambda item: item["left"]):
        best = None
        best_distance = None
        for column in columns:
            center = sum(item["left"] for item in column) / len(column)
            distance = abs(line["left"] - center)
            if distance <= threshold and (
                best_distance is None or distance < best_distance
            ):
                best = column
                best_distance = distance
        if best is None:
            columns.append([line])
        else:
            best.append(line)

    columns.sort(
        key=lambda column: sum(item["left"] for item in column) / len(column)
    )
    if len(columns) > 4:
        return sorted(lines, key=lambda item: (item["top"], item["left"]))
    ordered = []
    for column in columns:
        ordered.extend(
            sorted(column, key=lambda item: (item["top"], item["left"]))
        )
    return ordered


def order_ocr_lines(lines, page_width, page_height):
    """Use wide headings as separators, then read each section column by column."""
    if not lines:
        return []
    lines = deduplicate_ocr_lines(lines)
    wide = sorted(
        [
            line
            for line in lines
            if line["right"] - line["left"] >= page_width * 0.58
        ],
        key=lambda item: (item["top"], item["left"]),
    )
    wide_ids = {id(line) for line in wide}
    regular = [line for line in lines if id(line) not in wide_ids]
    ordered = []
    cursor = -1.0
    for anchor in wide:
        segment = [
            line
            for line in regular
            if cursor < (line["top"] + line["bottom"]) / 2 < anchor["top"]
        ]
        ordered.extend(order_column_segment(segment, page_width))
        ordered.append(anchor)
        cursor = max(cursor, anchor["bottom"])
    ordered.extend(
        order_column_segment(
            [
                line
                for line in regular
                if (line["top"] + line["bottom"]) / 2 > cursor
            ],
            page_width,
        )
    )
    return ordered


def ocr_page(page):
    engine = local_ocr_engine()
    lines = []
    page_width = 1
    page_height = 1
    with _ocr_lock:
        for image, offset_y, page_width, page_height in render_page_tiles(page):
            result = engine(image)
            boxes = result.boxes if result is not None else None
            texts = result.txts if result is not None else None
            scores = result.scores if result is not None else None
            if boxes is None or texts is None or scores is None:
                continue
            for box, text, score in zip(boxes, texts, scores):
                cleaned = normalize_text(text or "")
                if not cleaned or float(score) < OCR_MIN_CONFIDENCE:
                    continue
                xs = [float(point[0]) for point in box]
                ys = [float(point[1]) + offset_y for point in box]
                lines.append(
                    {
                        "text": cleaned,
                        "left": min(xs),
                        "right": max(xs),
                        "top": min(ys),
                        "bottom": max(ys),
                    }
                )
    ordered = order_ocr_lines(lines, page_width, page_height)
    return normalize_text("\n".join(line["text"] for line in ordered))


def read_pdf(path):
    path = Path(path)
    pypdf_texts = pypdf_page_texts(path)
    if pypdf_texts and all(
        not text or len(text) >= PDF_MIN_PAGE_TEXT for text in pypdf_texts
    ):
        combined = normalize_text("\n".join(pypdf_texts))
        if combined:
            return combined, "pypdf"

    if pymupdf is None:
        raise PDFParseError(
            "PDF没有可提取的文字层，且服务器尚未安装PyMuPDF本地解析组件"
        )
    try:
        document = pymupdf.open(path)
    except Exception as error:
        raise PDFParseError("PDF结构无法解析：{0}".format(error)) from error

    with document:
        if document.needs_pass and not document.authenticate(""):
            raise PDFParseError("PDF已加密，请先解除密码或访问限制后再入库")
        if document.page_count > PDF_MAX_PAGES:
            raise PDFParseError(
                "PDF共{0}页，超过系统允许的{1}页".format(
                    document.page_count, PDF_MAX_PAGES
                )
            )
        page_texts = []
        used_pymupdf = False
        ocr_pages = 0
        for page_index, page in enumerate(document):
            extracted = (
                pypdf_texts[page_index]
                if pypdf_texts and page_index < len(pypdf_texts)
                else ""
            )
            if len(extracted) < PDF_MIN_PAGE_TEXT:
                pymupdf_text = normalize_text(page.get_text("text", sort=True))
                if len(pymupdf_text) > len(extracted):
                    extracted = pymupdf_text
                    used_pymupdf = True
            if len(extracted) < PDF_MIN_PAGE_TEXT and page.get_images(full=True):
                ocr_pages += 1
                if ocr_pages > PDF_OCR_MAX_PAGES:
                    raise PDFParseError(
                        "需要OCR的页面超过{0}页，请拆分PDF后分批入库".format(
                            PDF_OCR_MAX_PAGES
                        )
                    )
                extracted = ocr_page(page)
            page_texts.append(extracted)

    combined = normalize_text("\n".join(page_texts))
    if not combined:
        raise PDFParseError(
            "PDF没有可识别正文；文件可能是空白页、低清扫描件或受损图片"
        )
    parser = "local-ocr" if ocr_pages else ("pymupdf" if used_pymupdf else "pypdf")
    return combined, parser
