# /// script
# requires-python = ">=3.11"
# dependencies = ["psutil"]
# ///
"""Runs the load generator for several session counts while sampling resources; see docs/design.md."""

import argparse
import csv
import statistics
import subprocess
import threading
import time
from pathlib import Path

import psutil

INTERVAL_SECS = 0.5
MB = 1024 * 1024

parser = argparse.ArgumentParser()
parser.add_argument("--start", type=int, default=100, help="first session count")
parser.add_argument("--stop", type=int, default=2000, help="last session count, included")
parser.add_argument("--step", type=int, default=100, help="difference between session counts")
parser.add_argument("--single", action="store_true", help="pass -single to the load generator")
parser.add_argument("--out", default="sweep", help="output folder")
args = parser.parse_args()

services = ["singlesession" if args.single else "router", "dispatcher", "fakemodel"]
cpu_columns = ["cpu_" + s for s in services]
ram_columns = ["ram_mb_" + s for s in services]
metrics = cpu_columns + ram_columns + ["disk_mb_s", "net_mb_s"]


def sample(stop, rows):
    start = time.time()
    disk, net, last = psutil.disk_io_counters(), psutil.net_io_counters(), start
    while not stop.wait(INTERVAL_SECS):
        now = time.time()
        row = dict.fromkeys(metrics, 0.0)
        for p in psutil.process_iter(["name"]):
            service = p.info["name"].removesuffix(".exe")
            if service in services:
                try:
                    row["cpu_" + service] += p.cpu_percent()
                    row["ram_mb_" + service] += p.memory_info().rss / MB
                except psutil.NoSuchProcess:
                    pass
        new_disk, new_net = psutil.disk_io_counters(), psutil.net_io_counters()
        secs = now - last
        row["disk_mb_s"] = (new_disk.read_bytes + new_disk.write_bytes - disk.read_bytes - disk.write_bytes) / MB / secs
        row["net_mb_s"] = (new_net.bytes_sent + new_net.bytes_recv - net.bytes_sent - net.bytes_recv) / MB / secs
        disk, net, last = new_disk, new_net, now
        rows.append({"time": round(now - start, 2)} | row)


out = Path(args.out)
out.mkdir(exist_ok=True)
running = {p.info["name"].removesuffix(".exe") for p in psutil.process_iter(["name"])}
for service in services:
    if service not in running:
        print(f"{service} is not running: its columns stay 0")


def write_outputs(summary, raw):
    for name, table in [("sweep.csv", summary), ("metrics.csv", raw)]:
        with open(out / name, "w", newline="") as file:
            w = csv.DictWriter(file, fieldnames=table[0].keys())
            w.writeheader()
            w.writerows(table)


summary = []
raw = []
for n in range(args.start, args.stop + 1, args.step):
    command = ["./loadgenerator", "-n", str(n)] + (["-single"] if args.single else [])
    rows = []
    stop = threading.Event()
    sampler = threading.Thread(target=sample, args=(stop, rows), daemon=True)
    sampler.start()
    start = time.time()
    subprocess.run(command, check=True)
    wall = time.time() - start
    stop.set()
    sampler.join()

    with open("latency.csv", newline="") as file:
        results = list(csv.DictReader(file))
    Path("latency.csv").unlink()
    overheads = sorted(float(r["overhead_ms"]) for r in results)
    row = {
        "n": n,
        "failed": sum(r["status"] == "failed" for r in results),
        "error": sum(r["status"] == "error" for r in results),
        "min_ms": overheads[0],
        "median_ms": statistics.median(overheads),
        "p99_ms": overheads[n * 99 // 100],
        "max_ms": overheads[-1],
        "wall_s": round(wall, 1),
    }
    for m in metrics:
        values = [r[m] for r in rows] or [0.0]
        row["mean_" + m] = round(statistics.mean(values), 2)
        row["peak_" + m] = round(max(values), 2)
    summary.append(row)
    raw += [{"n": n} | r for r in rows]
    write_outputs(summary, raw)
