#!/usr/bin/env python3
"""Smoke test for the vaxis/ui demo app — start, navigate pages, quit.

NOTE: The Screen emulator only tracks sequential rendering. vaxis/ui uses
diff-based partial updates with cursor positioning that this emulator
cannot fully reproduce. Assertions are limited to text that appears in
sequential order (page titles, static content)."""

import os, signal, sys, time
from test_harness import Screen, build, spawn, read_for, wait_for, send, drain, wait_exit

ROOT = os.path.dirname(os.path.abspath(__file__))
BIN = os.path.join(ROOT, "demo")


def main():
    build("demo")
    pid, fd = spawn(BIN, rows=30, cols=90)
    screen = Screen(30, 90)
    try:
        # Home page renders
        wait_for(fd, screen, "Vaxis UI demo")
        drain(fd, screen, 0.5)
        text = screen.text()
        assert "Home" in text, f"home tab missing:\n{text}"
        assert "This example is intentionally larger" in text, f"home content missing:\n{text}"

        # Navigate through all pages
        pages = ["Text layout", "Controls", "Lists", "Table", "Animation", "Theme"]
        for name in pages:
            send(fd, "n")
            wait_for(fd, screen, name)
            print(f"  ✓ {name}")

        # Backward
        send(fd, "p")
        wait_for(fd, screen, "Animation")
        print("  ✓ backward navigation")

        # Quit
        send(fd, "q")
        status = wait_exit(pid, fd, screen, timeout=3.0)
        if status is None:
            send(fd, "\x03")
            raise AssertionError("did not exit after q")
        assert status == 0, f"exit status {status}"
        print("  ✓ clean quit")

    finally:
        _cleanup(fd, pid)


def _cleanup(fd, pid):
    try:
        os.close(fd)
    except OSError:
        pass
    try:
        os.kill(pid, signal.SIGTERM)
    except OSError:
        pass


if __name__ == "__main__":
    try:
        main()
        print("\n✓ all demo smoke tests passed")
    except AssertionError as e:
        print(f"\n✗ FAIL: {e}", file=sys.stderr)
        sys.exit(1)
