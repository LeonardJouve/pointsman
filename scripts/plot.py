# /// script
# requires-python = ">=3.11"
# dependencies = ["matplotlib"]
# ///
"""Plots the sweep.csv of router mode and single mode together; see docs/design.md."""

import argparse
import csv

import matplotlib

matplotlib.use("Agg")
import matplotlib.pyplot as plt

SURFACE = "#fcfcfb"
INK = "#0b0b0b"
MUTED = "#898781"
GRID = "#e1e0d9"
BLUE = "#2a78d6"
ORANGE = "#eb6834"

parser = argparse.ArgumentParser()
parser.add_argument("--router", required=True, help="sweep.csv of router mode")
parser.add_argument("--single", required=True, help="sweep.csv of single mode")
parser.add_argument("--out", default="compare.png", help="output image")
args = parser.parse_args()


def load(path):
    with open(path, newline="") as file:
        rows = list(csv.DictReader(file))
    columns = rows[0].keys()
    return {
        "n": [int(r["n"]) for r in rows],
        "median": [float(r["median_ms"]) for r in rows],
        "p99": [float(r["p99_ms"]) for r in rows],
        "cpu": [sum(float(r[c]) for c in columns if c.startswith("peak_cpu_")) for r in rows],
        "ram": [sum(float(r[c]) for c in columns if c.startswith("peak_ram_mb_")) for r in rows],
    }


modes = [("router", load(args.router), BLUE), ("single", load(args.single), ORANGE)]
panels = [
    ("Latency overhead (ms)", ["median", "p99"], True),
    ("Peak CPU (% of one core)", ["cpu"], False),
    ("Peak RAM (MB)", ["ram"], True),
]

fig, axes = plt.subplots(1, 3, figsize=(15, 4.5), facecolor=SURFACE)
for ax, (title, keys, log) in zip(axes, panels):
    ax.set_facecolor(SURFACE)
    for name, data, color in modes:
        for key in keys:
            p99 = key == "p99"
            ax.plot(data["n"], data[key], color=color, linestyle="--" if p99 else "-", linewidth=2,
                    label=None if p99 else name)
    if log:
        ax.set_yscale("log")
    ax.set_title(title, color=INK, loc="left")
    ax.set_xlabel("concurrent sessions", color=MUTED)
    ax.grid(axis="y", color=GRID, linewidth=0.8)
    ax.tick_params(colors=MUTED)
    for side in ["top", "right", "left"]:
        ax.spines[side].set_visible(False)
    ax.spines["bottom"].set_color(GRID)
axes[0].plot([], [], color=MUTED, linestyle="-", label="median")
axes[0].plot([], [], color=MUTED, linestyle="--", label="p99")
axes[0].legend(frameon=False, labelcolor=INK)
fig.tight_layout()
fig.savefig(args.out, dpi=150)
