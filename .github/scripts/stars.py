#!/usr/bin/env python3
"""Star history of Nimbus Backup.

Run daily by .github/workflows/stars.yml. Standard library only.

    stars.py OUT_DIR

In OUT_DIR (the checked-out `stats` branch):
- stars.csv: one row per day (stars, forks, watchers, open issues),
  appended to; a second run on the same day replaces that day.
- stars.svg: our stars over time, from the date of every star (GitHub
  lists them to the repository's maintainers, so GITHUB_TOKEN is needed;
  without it the curve comes from stars.csv).
"""

import csv
import datetime as dt
import json
import os
import sys
import urllib.error
import urllib.request

OURS = "rdemsystems/NimbusBackupClient"
API = "https://api.github.com"


def get(path, accept="application/vnd.github+json"):
    """Every page of a GitHub API GET (a list), or the object for a single one."""
    url = API + path
    out = None
    while url:
        req = urllib.request.Request(url, headers={"Accept": accept, "User-Agent": "nimbus-stars"})
        token = os.environ.get("GITHUB_TOKEN")
        if token:
            req.add_header("Authorization", "Bearer " + token)
        with urllib.request.urlopen(req, timeout=60) as resp:
            data = json.load(resp)
            link = resp.headers.get("Link", "")
        if not isinstance(data, list):
            return data
        out = (out or []) + data
        url = None
        for part in link.split(","):
            if 'rel="next"' in part:
                url = part[part.index("<") + 1 : part.index(">")]
    return out or []


def star_dates(repo):
    """The date of every star, or None when GitHub does not list them to us."""
    try:
        stars = get(f"/repos/{repo}/stargazers?per_page=100", accept="application/vnd.github.star+json")
    except urllib.error.HTTPError as e:
        print(f"{repo}: stargazers not listed ({e.code}), using the daily log")
        return None
    return sorted(dt.datetime.fromisoformat(s["starred_at"].replace("Z", "+00:00")) for s in stars)


def steps(dates):
    """(time, count) points of a star history."""
    return [(d, n) for n, d in enumerate(dates, 1)]


def logged(rows, repo):
    """(time, count) points of a repository from the daily log."""
    pts = []
    for r in rows:
        if r["repo"] == repo:
            t = dt.datetime.strptime(r["date"], "%Y-%m-%d").replace(tzinfo=dt.timezone.utc)
            pts.append((t, int(r["stars"])))
    return pts


def log_csv(path, today, infos):
    rows = []
    if os.path.exists(path):
        with open(path, newline="") as f:
            rows = [r for r in csv.DictReader(f) if r["date"] != today]
    for repo, info in infos.items():
        rows.append({
            "date": today,
            "repo": repo,
            "stars": info["stargazers_count"],
            "forks": info["forks_count"],
            "watchers": info["subscribers_count"],
            "open_issues": info["open_issues_count"],
        })
    rows.sort(key=lambda r: (r["date"], r["repo"]))
    with open(path, "w", newline="") as f:
        w = csv.DictWriter(f, fieldnames=["date", "repo", "stars", "forks", "watchers", "open_issues"])
        w.writeheader()
        w.writerows(rows)
    return rows


# Readable in light and dark mode.
COLORS = ["#2a78d6"]


def svg_chart(path, title, series, now):
    """A step line per repository: stars (y) over time (x). series is a list
    of (label, (time, count) points in time order)."""
    # The legend sits right of the plot, as wide as its longest line.
    right = 60 + 7 * max(len(f"{label} · {p[-1][1] if p else 0}") for label, p in series)
    w, h = 620 + right, 360
    left, top, bottom = 56, 44, 40
    pw, ph = w - left - right, h - top - bottom
    start = min(p[0][0] for _, p in series if p)
    start = start.replace(day=1, hour=0, minute=0, second=0, microsecond=0)
    span = (now - start).total_seconds() or 1
    ymax = max((p[-1][1] for _, p in series if p), default=1) or 1
    # A round step for the y grid.
    step = next(s for s in (1, 2, 5, 10, 20, 25, 50, 100, 200, 250, 500, 1000, 2000, 5000) if ymax / s <= 6)
    ytop = ((ymax + step - 1) // step) * step

    def x(t):
        return left + pw * (t - start).total_seconds() / span

    def y(n):
        return top + ph - ph * n / ytop

    out = [
        f'<svg xmlns="http://www.w3.org/2000/svg" width="{w}" height="{h}" viewBox="0 0 {w} {h}" '
        'font-family="-apple-system,Segoe UI,Helvetica,Arial,sans-serif" font-size="12">',
        f'<text fill="#8b949e" x="{left}" y="24" font-size="15" font-weight="600">{title}</text>',
    ]
    for n in range(0, ytop + 1, step):
        out.append(f'<line stroke="#8b949e" stroke-opacity="0.25" x1="{left}" x2="{left + pw}" y1="{y(n):.1f}" y2="{y(n):.1f}"/>')
        out.append(f'<text fill="#8b949e" x="{left - 8}" y="{y(n) + 4:.1f}" text-anchor="end">{n}</text>')
    # One tick per month, or per quarter on long spans.
    months = (now.year - start.year) * 12 + now.month - start.month
    every = 1 if months <= 12 else 3 if months <= 36 else 6
    m = start
    while m <= now:
        if (m.month - 1) % every == 0:
            out.append(f'<line stroke="#8b949e" stroke-opacity="0.25" x1="{x(m):.1f}" x2="{x(m):.1f}" y1="{top}" y2="{top + ph}"/>')
            out.append(f'<text fill="#8b949e" x="{x(m):.1f}" y="{top + ph + 18}" text-anchor="middle">{m.strftime("%b %y")}</text>')
        m = m.replace(year=m.year + (m.month == 12), month=m.month % 12 + 1)
    for i, (label, points) in enumerate(series):
        if not points:
            continue
        color = COLORS[i % len(COLORS)]
        pts, prev = [], None
        for t, n in points:
            if prev is not None:
                pts.append(f"{x(t):.1f},{y(prev):.1f}")
            pts.append(f"{x(t):.1f},{y(n):.1f}")
            prev = n
        pts.append(f"{x(now):.1f},{y(prev):.1f}")
        width = 2.5 if i == 0 else 1.6
        out.append(f'<path fill="none" stroke="{color}" stroke-width="{width}" stroke-linejoin="round" d="M{" L".join(pts)}"/>')
        out.append(f'<circle cx="{x(now):.1f}" cy="{y(prev):.1f}" r="2.5" fill="{color}"/>')
        ly = top + 6 + i * 20
        out.append(f'<rect x="{left + pw + 16}" y="{ly}" width="12" height="3" fill="{color}"/>')
        out.append(f'<text fill="#8b949e" x="{left + pw + 34}" y="{ly + 5}">{label} · {prev}</text>')
    out.append(f'<text fill="#8b949e" x="{left + pw}" y="{h - 6}" text-anchor="end" font-size="10">updated {now:%Y-%m-%d}</text>')
    out.append("</svg>")
    with open(path, "w") as f:
        f.write("\n".join(out) + "\n")


def main():
    outdir = sys.argv[1] if len(sys.argv) > 1 else "."
    now = dt.datetime.now(dt.timezone.utc)
    info = get(f"/repos/{OURS}")
    rows = log_csv(os.path.join(outdir, "stars.csv"), now.strftime("%Y-%m-%d"), {OURS: info})
    dates = star_dates(OURS)
    points = steps(dates) if dates is not None else logged(rows, OURS)
    svg_chart(os.path.join(outdir, "stars.svg"), "Nimbus Backup — GitHub stars", [("NimbusBackupClient", points)], now)
    print(f"{OURS}: {info['stargazers_count']} stars")


if __name__ == "__main__":
    main()
