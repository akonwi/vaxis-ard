#!/usr/bin/env python3
"""Smoke test for the vaxis/ui todo app — start, navigate, toggle, quit."""

import os, signal, sys, time
from test_harness import Screen, build, spawn, read_for, wait_for, send, drain, wait_exit

ROOT = os.path.dirname(os.path.abspath(__file__))
BIN = os.path.join(ROOT, "todo")


def main():
    build("todo")
    pid, fd = spawn(BIN, rows=30, cols=80)
    screen = Screen(30, 80)
    try:
        # App starts and renders
        wait_for(fd, screen, "Vaxis/ui Todo")
        drain(fd, screen, 0.5)
        text = screen.text()
        assert "Wire project Go FFI" in text, f"first todo missing:\n{text}"
        assert "Done:" in text, f"done count missing:\n{text}"

        # Toggle with Enter
        send(fd, "\r")
        drain(fd, screen, 0.3)

        # Navigate down with j
        send(fd, "j")
        drain(fd, screen, 0.2)

        # Navigate down with arrow
        send(fd, "\x1b[B")
        drain(fd, screen, 0.2)

        # Navigate up with k
        send(fd, "k")
        drain(fd, screen, 0.2)

        # Toggle with Space
        send(fd, " ")
        drain(fd, screen, 0.3)

        # Reset
        send(fd, "r")
        drain(fd, screen, 0.3)
        assert "Done:" in screen.text()

        # Quit
        send(fd, "q")
        status = wait_exit(pid, fd, screen, timeout=2.0)
        if status is None:
            send(fd, "\x03")
            raise AssertionError("did not exit after q")
        assert status == 0, f"exit status {status}"

        print("✓ todo smoke test passed")
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
    except Exception as err:
        print(f"FAIL: {err}", file=sys.stderr)
        sys.exit(1)
