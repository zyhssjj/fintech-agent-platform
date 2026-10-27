"""Normalize model answers for the interactive UI and external API callers."""

import re


def plain_text(value):
    """Remove common Markdown markers while preserving readable plain-text lists."""
    text = (value or "").replace("\r\n", "\n").replace("\r", "\n")
    text = re.sub(r"(?m)^\s{0,3}#{1,6}\s+", "", text)
    text = re.sub(r"(?m)^\s{0,3}>\s?", "", text)
    text = re.sub(r"(?m)^\s{0,3}([-*_])(?:\s*\1){2,}\s*$", "", text)
    text = re.sub(r"!\[([^\]]*)\]\([^)]*\)", r"\1", text)
    text = re.sub(r"\[([^\]]+)\]\([^)]*\)", r"\1", text)
    text = re.sub(r"(`{1,3})(.*?)\1", r"\2", text, flags=re.DOTALL)
    text = re.sub(r"(\*\*|__)(.*?)\1", r"\2", text, flags=re.DOTALL)
    text = re.sub(r"(?<!\w)[*_]([^\n*_]+)[*_](?!\w)", r"\1", text)
    text = re.sub(r"(?m)^\s*[-*+]\s+", "• ", text)
    text = re.sub(r"\n{3,}", "\n\n", text)
    return text.strip()


def external_plain_text(value):
    """Return plain text safe for consumers that cannot open KB references."""
    text = plain_text(value)
    # Internal references are generated solely as [positive integer].  Remove
    # only that exact pattern so bracketed factual content is not affected.
    text = re.sub(r"\s*\[\d+\]", "", text)
    text = re.sub(r"[ \t]+\n", "\n", text)
    text = re.sub(r" {2,}", " ", text)
    return text.strip()
