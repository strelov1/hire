#!/usr/bin/env python3
"""Query-driven open-web ATS board discovery.

Run a free-text query across search channels (DuckDuckGo, Google CSE, GitHub code
search, Common Crawl), extract ATS board URLs, dedup against sources/*.yml,
validate each board live, and print (or --write) ready-to-paste YAML.

Usage:
    python3 scripts/discover_boards.py --query "fintech berlin" \
            --provider ashby,lever --channel ddg,github,google,cc [--write] [--limit N]

Stdlib only; the github channel shells out to `gh`; google needs GOOGLE_CSE_KEY/_CX.
"""

from __future__ import annotations

import json
import os
import re
import sys
import urllib.parse
import urllib.request
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))
from ats_boards import VALIDATORS, github_fragments  # noqa: E402

# Provider -> the ATS host to put in a `site:` search / Common Crawl prefix.
PROVIDER_HOSTS = {
    "greenhouse": "job-boards.greenhouse.io",
    "lever": "jobs.lever.co",
    "ashby": "jobs.ashbyhq.com",
    "smartrecruiters": "jobs.smartrecruiters.com",
    "workable": "apply.workable.com",
    "recruitee": "recruitee.com",
    "bamboohr": "bamboohr.com",
    "breezy": "breezy.hr",
    "personio": "jobs.personio.com",
    "teamtailor": "teamtailor.com",
}

# DuckDuckGo's HTML endpoint rejects unusual UAs; use a browser-like one.
BROWSER_UA = (
    "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 "
    "(KHTML, like Gecko) Chrome/124.0 Safari/537.36"
)


def get_text(url: str, timeout: int = 25) -> str:
    """GET a URL with a browser UA, returning decoded text ('' on failure)."""
    req = urllib.request.Request(url, headers={"User-Agent": BROWSER_UA})
    try:
        with urllib.request.urlopen(req, timeout=timeout) as r:
            return r.read().decode("utf-8", "replace")
    except Exception as e:
        print(f"  ! GET failed ({url[:60]}...): {e}", file=sys.stderr)
        return ""


def parse_ddg_html(html: str) -> set[str]:
    """Extract target URLs from DuckDuckGo HTML results (unwrap ?uddg= redirects)."""
    return {urllib.parse.unquote(m) for m in re.findall(r"uddg=([^\"&]+)", html)}


def channel_ddg(host: str, query: str, limit: int) -> set[str]:
    """site:<host> <query> via DuckDuckGo HTML -> raw target URLs."""
    q = urllib.parse.quote(f"site:{host} {query}")
    html = get_text(f"https://html.duckduckgo.com/html/?q={q}")
    urls = parse_ddg_html(html)
    return set(list(urls)[:limit]) if limit else urls


def parse_cse_items(obj: dict) -> set[str]:
    """Extract result links from a Google Custom Search JSON response."""
    return {it["link"] for it in obj.get("items", []) if it.get("link")}


def channel_google(host: str, query: str, limit: int) -> set[str]:
    """site:<host> <query> via Google Custom Search JSON API (env-gated)."""
    key, cx = os.environ.get("GOOGLE_CSE_KEY"), os.environ.get("GOOGLE_CSE_CX")
    if not (key and cx):
        print("  ! google channel skipped (set GOOGLE_CSE_KEY and GOOGLE_CSE_CX)", file=sys.stderr)
        return set()
    q = urllib.parse.quote(f"site:{host} {query}")
    num = min(limit, 10) if limit else 10
    body = get_text(
        f"https://www.googleapis.com/customsearch/v1?key={key}&cx={cx}&q={q}&num={num}"
    )
    try:
        return parse_cse_items(json.loads(body))
    except Exception:
        return set()


def channel_github(host: str, query: str, limit: int, pages: int = 2) -> set[str]:
    """`<query> <host>` via GitHub code search -> the matched fragments as text."""
    frags = github_fragments(f"{query} {host}", pages)
    out = set(frags)
    return set(list(out)[:limit]) if limit else out


_CC_BASE: str | None = None


def cc_api_base() -> str | None:
    """Latest Common Crawl URL-index CDX API base (cached). None if unreachable."""
    global _CC_BASE
    if _CC_BASE is None:
        body = get_text("https://index.commoncrawl.org/collinfo.json")
        try:
            _CC_BASE = json.loads(body)[0]["cdx-api"]
        except Exception:
            print("  ! common-crawl index list unreachable", file=sys.stderr)
            _CC_BASE = ""
    return _CC_BASE or None


def parse_cc_jsonl(text: str) -> set[str]:
    """Collect the `url` field from a Common Crawl CDX JSONL response."""
    urls: set[str] = set()
    for line in text.splitlines():
        line = line.strip()
        if not line:
            continue
        try:
            row = json.loads(line)
        except Exception:
            continue
        if isinstance(row, dict) and row.get("url"):
            urls.add(row["url"])
    return urls


def channel_cc(host: str, query: str, limit: int) -> set[str]:
    """Bulk-dump board URLs for <host>/* from Common Crawl. Ignores the keyword."""
    base = cc_api_base()
    if not base:
        return set()
    cap = (limit or 100) * 5  # over-fetch; many rows collapse to one slug
    url = f"{base}?url={urllib.parse.quote(host)}/*&output=json&limit={cap}"
    return parse_cc_jsonl(get_text(url, timeout=60))
