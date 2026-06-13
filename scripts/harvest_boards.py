#!/usr/bin/env python3
"""Harvest greenhouse/lever/ashby board slugs from public job aggregators.

Pipeline: collect candidate (provider, slug, company) tuples from a set of
aggregator JSON files on GitHub (and, optionally, GitHub code search), drop the
ones we already track in sources/*.yml, validate the rest against the same public
ATS endpoints our ingest adapters use, and print the survivors as ready-to-paste
YAML.

Usage:
    python3 scripts/harvest_boards.py              # JSON aggregators only
    python3 scripts/harvest_boards.py --github     # also sweep GitHub code search (needs gh, 10 req/min)
    python3 scripts/harvest_boards.py --write      # append survivors to sources/<provider>.yml

Only standard library is used; GitHub code search shells out to `gh`.
"""

from __future__ import annotations

import argparse
import json
import os
import re
import subprocess
import sys
import urllib.request
from collections import defaultdict
from concurrent.futures import ThreadPoolExecutor
from pathlib import Path

REPO = Path(__file__).resolve().parent.parent
SOURCES_DIR = REPO / "sources"
UA = "freehire-harvest/1.0 (+https://freehire.dev)"

# Aggregator JSON files. We sweep the raw text with regex, so the per-file schema
# (key names) does not matter — only that ATS URLs appear somewhere in the JSON.
AGGREGATORS = [
    "https://raw.githubusercontent.com/vanshb03/Summer2026-Internships/dev/.github/scripts/listings.json",
    "https://raw.githubusercontent.com/SimplifyJobs/Summer2026-Internships/dev/.github/scripts/listings.json",
    "https://raw.githubusercontent.com/crypto-jobs-fyi/crawler/HEAD/ai_companies.json",
    "https://raw.githubusercontent.com/crypto-jobs-fyi/crawler/HEAD/crypto_companies.json",
    "https://raw.githubusercontent.com/crypto-jobs-fyi/crawler/HEAD/fin_companies.json",
    "https://raw.githubusercontent.com/crypto-jobs-fyi/crawler/HEAD/tech_companies.json",
]

# (compiled regex, provider). Group 1 is the board slug.
SLUG_PATTERNS = [
    (re.compile(r"(?:boards|job-boards)(?:\.eu)?\.greenhouse\.io/(?:embed/job_app\?for=)?([A-Za-z0-9_-]+)"), "greenhouse"),
    (re.compile(r"jobs\.(?:eu\.)?lever\.co/([A-Za-z0-9_.-]+)"), "lever"),
    (re.compile(r"jobs\.ashbyhq\.com/([A-Za-z0-9_.-]+)"), "ashby"),
]

# Slugs that are path segments of the ATS host itself, not real boards.
SLUG_BLOCKLIST = {"embed", "jobs", "job", "j", "o", "share", "en-us", "en-gb"}

# Workday is a separate beast: a board is host + career-site path (e.g.
# "logitech.wd5.myworkdayjobs.com/Logitech"), and the site segment sits before
# /job/ or /details/ in a posting URL, after an optional locale prefix.
WORKDAY_RE = re.compile(
    r"https?://([a-z0-9-]+\.wd\d+\.myworkdayjobs\.com)/(?:[a-zA-Z]{2}-[a-zA-Z]{2}/)?([^/?\"]+)/(?:job|details)/"
)

# Validation endpoints — identical to internal/sources/{greenhouse,lever,ashby}.go.
VALIDATORS = {
    "greenhouse": lambda s: f"https://boards-api.greenhouse.io/v1/boards/{s}/jobs?content=true",
    "lever": lambda s: f"https://api.lever.co/v0/postings/{s}?mode=json",
    "ashby": lambda s: f"https://api.ashbyhq.com/posting-api/job-board/{s}",
}


def fetch(url: str, timeout: int = 25) -> bytes | None:
    req = urllib.request.Request(url, headers={"User-Agent": UA, "Accept": "application/json"})
    try:
        with urllib.request.urlopen(req, timeout=timeout) as r:
            return r.read()
    except Exception:
        return None


def yaml_name(name: str) -> str:
    """Quote a company name only when it carries YAML-significant characters."""
    name = name.strip()
    if not name or name[0] in "-?:,[]{}#&*!|>'\"%@`" or any(c in name for c in ':#,"'):
        return '"' + name.replace('\\', '\\\\').replace('"', '\\"') + '"'
    return name


def extract_slugs(text: str) -> set[tuple[str, str]]:
    """Sweep raw text for ATS board URLs -> {(provider, slug)}."""
    out: set[tuple[str, str]] = set()
    for pat, provider in SLUG_PATTERNS:
        for slug in pat.findall(text):
            if slug.lower() in SLUG_BLOCKLIST:
                continue
            out.add((provider, slug))
    return out


def harvest_aggregators() -> dict[tuple[str, str], str]:
    """Return {(provider, slug): company_name}. Company name is best-effort."""
    candidates: dict[tuple[str, str], str] = {}
    for url in AGGREGATORS:
        body = fetch(url)
        if not body:
            print(f"  ! skip (fetch failed): {url}", file=sys.stderr)
            continue
        text = body.decode("utf-8", "replace")
        found = extract_slugs(text)
        # Best-effort company-name attribution from structured JSON.
        names = attribute_names(text)
        for prov, slug in found:
            candidates.setdefault((prov, slug), names.get(slug.lower(), slug))
        print(f"  {len(found):4} slugs from {url.split('/')[4]}/{url.split('/')[5]} ({url.rsplit('/',1)[-1]})", file=sys.stderr)
    return candidates


def attribute_names(text: str) -> dict[str, str]:
    """Map lowercased slug -> human company name when the JSON exposes one."""
    names: dict[str, str] = {}
    try:
        data = json.loads(text)
    except Exception:
        return names
    rows = data if isinstance(data, list) else data.values() if isinstance(data, dict) else []
    for row in rows:
        if not isinstance(row, dict):
            continue
        name = next((row[k] for k in ("company_name", "company", "name", "employer") if row.get(k)), None)
        if not name:
            continue
        for v in row.values():
            if not isinstance(v, str):
                continue
            for prov, slug in extract_slugs(v):
                names.setdefault(slug.lower(), str(name).strip())
    return names


def harvest_github(per_provider_pages: int = 2) -> set[tuple[str, str]]:
    """Sweep GitHub code search for the three ATS host patterns (needs gh)."""
    out: set[tuple[str, str]] = set()
    queries = ["job-boards.greenhouse.io", "jobs.lever.co", "jobs.ashbyhq.com"]
    for q in queries:
        for page in range(1, per_provider_pages + 1):
            try:
                raw = subprocess.run(
                    ["gh", "api", "-X", "GET", "search/code",
                     "-H", "Accept: application/vnd.github.text-match+json",
                     "-f", f"q={q} in:file", "-f", "per_page=100", "-f", f"page={page}"],
                    capture_output=True, text=True, timeout=60,
                ).stdout
                items = json.loads(raw).get("items", [])
            except Exception as e:
                print(f"  ! github query failed ({q} p{page}): {e}", file=sys.stderr)
                break
            if not items:
                break
            for it in items:
                for m in it.get("text_matches", []):
                    out |= extract_slugs(m.get("fragment", ""))
    return out


def existing_slugs() -> dict[str, set[str]]:
    out: dict[str, set[str]] = defaultdict(set)
    for prov in VALIDATORS:
        f = SOURCES_DIR / f"{prov}.yml"
        if f.exists():
            for m in re.findall(r"board:\s*\"?([^\"\n]+)\"?", f.read_text()):
                out[prov].add(m.strip().lower())
    return out


def validate(provider: str, slug: str) -> int | None:
    """Return active job count if the board is live and non-empty, else None."""
    body = fetch(VALIDATORS[provider](slug), timeout=20)
    if not body:
        return None
    try:
        data = json.loads(body)
    except Exception:
        return None
    if provider == "lever":
        jobs = data if isinstance(data, list) else []
    else:  # greenhouse / ashby
        jobs = data.get("jobs", []) if isinstance(data, dict) else []
    # is_worth_adding: a board earns a slot only if it currently lists jobs.
    # Tune here if you want a higher bar (e.g. >= 5 jobs, or tech-only titles).
    return len(jobs) if jobs else None


def harvest_workday() -> dict[tuple[str, str], str]:
    """Return {(host, site): company_name} for Workday boards in the aggregators."""
    out: dict[tuple[str, str], str] = {}
    for url in AGGREGATORS:
        body = fetch(url)
        if not body:
            continue
        text = body.decode("utf-8", "replace")
        names = workday_names(text)
        for host, site in WORKDAY_RE.findall(text):
            tenant = host.split(".")[0]
            out.setdefault((host, site), names.get((host, site), tenant.title()))
    return out


def workday_names(text: str) -> dict[tuple[str, str], str]:
    """Best-effort (host, site) -> company name from structured JSON rows."""
    names: dict[tuple[str, str], str] = {}
    try:
        data = json.loads(text)
    except Exception:
        return names
    rows = data if isinstance(data, list) else data.values() if isinstance(data, dict) else []
    for row in rows:
        if not isinstance(row, dict):
            continue
        name = next((row[k] for k in ("company_name", "company", "name", "employer") if row.get(k)), None)
        if not name:
            continue
        for v in row.values():
            if isinstance(v, str):
                for host, site in WORKDAY_RE.findall(v):
                    names.setdefault((host, site), str(name).strip())
    return names


def github_fragments(query: str, pages: int) -> list[str]:
    """Run a GitHub code-search query and return the matched text fragments."""
    frags: list[str] = []
    for page in range(1, pages + 1):
        try:
            raw = subprocess.run(
                ["gh", "api", "-X", "GET", "search/code",
                 "-H", "Accept: application/vnd.github.text-match+json",
                 "-f", f"q={query} in:file", "-f", "per_page=100", "-f", f"page={page}"],
                capture_output=True, text=True, timeout=60,
            ).stdout
            items = json.loads(raw).get("items", [])
        except Exception as e:
            print(f"  ! github query failed ({query} p{page}): {e}", file=sys.stderr)
            break
        if not items:
            break
        for it in items:
            for m in it.get("text_matches", []):
                frags.append(m.get("fragment", ""))
    return frags


def harvest_github_workday(pages: int = 5) -> dict[tuple[str, str], str]:
    """Sweep GitHub code search for Workday board URLs (job lists in READMEs etc.)."""
    out: dict[tuple[str, str], str] = {}
    for frag in github_fragments("myworkdayjobs.com", pages):
        for host, site in WORKDAY_RE.findall(frag):
            out.setdefault((host, site), host.split(".")[0].title())
    return out


def existing_workday() -> set[str]:
    out: set[str] = set()
    f = SOURCES_DIR / "workday.yml"
    if f.exists():
        # CXS is case-insensitive on host AND site, so dedup on the whole board lowered.
        for m in re.findall(r"board:\s*\"?([^\"\n]+)\"?", f.read_text()):
            out.add(m.strip().lower())
    return out


def validate_workday(host: str, site: str) -> int | None:
    """POST the CXS jobs endpoint; return posting count if the board is live."""
    tenant = host.split(".")[0]
    url = f"https://{host}/wday/cxs/{tenant}/{site}/jobs"
    payload = json.dumps({"appliedFacets": {}, "limit": 20, "offset": 0, "searchText": ""}).encode()
    req = urllib.request.Request(
        url, data=payload,
        headers={"User-Agent": UA, "Content-Type": "application/json", "Accept": "application/json"},
    )
    try:
        with urllib.request.urlopen(req, timeout=20) as r:
            data = json.loads(r.read())
    except Exception:
        return None
    total = data.get("total") if isinstance(data, dict) else 0
    total = total or len(data.get("jobPostings", []) if isinstance(data, dict) else [])
    return total or None


def run_workday(write: bool, github: bool) -> int:
    print("== harvesting Workday boards ==", file=sys.stderr)
    cand = harvest_workday()
    if github:
        print("== sweeping GitHub code search (myworkdayjobs) ==", file=sys.stderr)
        for key, name in harvest_github_workday().items():
            cand.setdefault(key, name)
    have = existing_workday()
    # Collapse case-variants of one board (CXS is case-insensitive), keeping the first
    # spelling seen, and drop anything already tracked.
    new: dict[tuple[str, str], str] = {}
    for (h, s), name in cand.items():
        board = f"{h}/{s}"
        if board.lower() in have:
            continue
        have.add(board.lower())
        new[(h, s)] = name
    print(f"{len(cand)} candidates, {len(cand) - len(new)} already tracked/collapsed, "
          f"{len(new)} new to validate", file=sys.stderr)

    items = list(new.items())
    with ThreadPoolExecutor(max_workers=12) as ex:
        counts = list(ex.map(lambda kv: validate_workday(kv[0][0], kv[0][1]), items))

    rows = sorted(
        (((n, f"{h}/{s}", c) for ((h, s), n), c in zip(items, counts) if c)),
        key=lambda r: -r[2],
    )
    print(f"\n# === workday: {len(rows)} new live boards ===")
    for name, board, c in rows:
        print(f"- company: {yaml_name(name)}  # {c} jobs")
        print(f"  board: {board}")
    if write:
        f = SOURCES_DIR / "workday.yml"
        with f.open("a") as fh:
            for name, board, c in rows:
                fh.write(f"- company: {yaml_name(name)}\n  board: {board}\n")
        print(f"  -> appended {len(rows)} entries to {f.relative_to(REPO)}", file=sys.stderr)
    return 0


def main() -> int:
    ap = argparse.ArgumentParser()
    ap.add_argument("--github", action="store_true", help="also sweep GitHub code search")
    ap.add_argument("--workday", action="store_true", help="harvest Workday boards only (host/site, POST validation)")
    ap.add_argument("--write", action="store_true", help="append survivors to sources/<provider>.yml")
    args = ap.parse_args()

    if args.workday:
        return run_workday(args.write, args.github)

    print("== harvesting aggregators ==", file=sys.stderr)
    cand = harvest_aggregators()
    if args.github:
        print("== sweeping GitHub code search ==", file=sys.stderr)
        for key in harvest_github():
            cand.setdefault(key, key[1])

    have = existing_slugs()
    new = {(p, s): n for (p, s), n in cand.items() if s.lower() not in have[p]}
    print(f"\n{len(cand)} unique candidates, {len(cand) - len(new)} already tracked, "
          f"{len(new)} new to validate\n", file=sys.stderr)

    # Validate concurrently.
    items = list(new.items())
    with ThreadPoolExecutor(max_workers=16) as ex:
        counts = list(ex.map(lambda kv: validate(kv[0][0], kv[0][1]), items))

    # Collapse case-variant slugs of the same board (e.g. Etched/etched) — they
    # resolve to one board, so keep the highest-job-count spelling only.
    best: dict[tuple[str, str], tuple[str, str, int]] = {}
    for ((prov, slug), name), n in zip(items, counts):
        if not n:
            continue
        key = (prov, slug.lower())
        if key not in best or n > best[key][2]:
            best[key] = (name, slug, n)
    survivors: dict[str, list[tuple[str, str, int]]] = defaultdict(list)
    for (prov, _), row in best.items():
        survivors[prov].append(row)

    total = 0
    for prov in VALIDATORS:
        rows = sorted(survivors[prov], key=lambda r: -r[2])
        if not rows:
            continue
        total += len(rows)
        print(f"\n# === {prov}: {len(rows)} new live boards ===")
        for name, slug, n in rows:
            print(f"- company: {yaml_name(name)}  # {n} jobs")
            print(f"  board: {slug}")
        if args.write:
            f = SOURCES_DIR / f"{prov}.yml"
            with f.open("a") as fh:
                for name, slug, n in rows:
                    fh.write(f"- company: {yaml_name(name)}\n  board: {slug}\n")
            print(f"  -> appended {len(rows)} entries to {f.relative_to(REPO)}", file=sys.stderr)

    print(f"\n{total} new validated boards total", file=sys.stderr)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
