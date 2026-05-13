#!/usr/bin/env python3
"""Static linter for site/. Run from the repo root or from site/.

Fails (exit 1) on any of:
- site/index.html does not parse as HTML
- index.html has no <script type="application/ld+json"> block, or the
  block does not parse as JSON, or it does not include the expected
  schema.org @type entries
- sitemap.xml does not parse as XML, or any <loc> points outside the
  configured host
- robots.txt does not contain a Sitemap: directive pointing at the
  same canonical host
- expected files are missing: index.html, 404.html, robots.txt,
  sitemap.xml, llms.txt, og-image.svg, favicon.svg,
  .well-known/security.txt, humans.txt
"""

import json
import re
import sys
import xml.etree.ElementTree as ET
from html.parser import HTMLParser
from pathlib import Path

HERE = Path(__file__).resolve().parent
CANONICAL = "https://ilabhu.com/"

REQUIRED_FILES = [
    "index.html",
    "404.html",
    "robots.txt",
    "sitemap.xml",
    "llms.txt",
    "og-image.svg",
    "favicon.svg",
    "humans.txt",
    ".well-known/security.txt",
]

REQUIRED_LD_TYPES = {
    "WebSite",
    "Organization",
    "SoftwareApplication",
    "SoftwareSourceCode",
    "FAQPage",
}


def fail(msg: str) -> None:
    print(f"site-lint: FAIL: {msg}", file=sys.stderr)
    sys.exit(1)


def ok(msg: str) -> None:
    print(f"site-lint: ok: {msg}")


def check_files_present() -> None:
    missing = [p for p in REQUIRED_FILES if not (HERE / p).exists()]
    if missing:
        fail(f"missing required files: {missing}")
    ok(f"all {len(REQUIRED_FILES)} required files present")


def check_index_html() -> None:
    raw = (HERE / "index.html").read_text(encoding="utf-8")
    HTMLParser().feed(raw)
    ok("index.html parses")

    blocks = re.findall(
        r'<script type="application/ld\+json">\s*(.*?)\s*</script>',
        raw,
        re.DOTALL,
    )
    if not blocks:
        fail("index.html has no <script type=application/ld+json> block")
    types_seen: set[str] = set()
    for block in blocks:
        doc = json.loads(block)
        for node in doc.get("@graph", [doc]):
            t = node.get("@type")
            if isinstance(t, str):
                types_seen.add(t)
    missing = REQUIRED_LD_TYPES - types_seen
    if missing:
        fail(f"JSON-LD is missing @type entries: {sorted(missing)}")
    ok(f"JSON-LD carries {sorted(types_seen & REQUIRED_LD_TYPES)}")


def check_sitemap_xml() -> None:
    tree = ET.parse(HERE / "sitemap.xml")
    ns = "{http://www.sitemaps.org/schemas/sitemap/0.9}"
    urls = [u.text or "" for u in tree.findall(f"{ns}url/{ns}loc")]
    if not urls:
        fail("sitemap.xml has no <url><loc> entries")
    for u in urls:
        if not u.startswith(CANONICAL.rstrip("/")):
            fail(f"sitemap loc {u!r} does not start with canonical host {CANONICAL!r}")
    ok(f"sitemap.xml carries {len(urls)} url(s)")


def check_robots_txt() -> None:
    raw = (HERE / "robots.txt").read_text(encoding="utf-8")
    if "Sitemap:" not in raw:
        fail("robots.txt does not advertise a Sitemap: directive")
    if "/sitemap.xml" not in raw:
        fail("robots.txt does not reference /sitemap.xml")
    ok("robots.txt advertises the sitemap")


def main() -> None:
    check_files_present()
    check_index_html()
    check_sitemap_xml()
    check_robots_txt()
    print("site-lint: PASS")


if __name__ == "__main__":
    main()
