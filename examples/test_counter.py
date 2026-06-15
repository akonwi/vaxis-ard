#!/usr/bin/env python3
"""Smoke test for the counter example — basic increment, decrement, reset, quit."""

import os, signal, sys, time
from test_harness import Screen, build, spawn, read_for, wait_for, send, wait_exit

ROOT = os.path.dirname(os.path.abspath(__file__))
BIN = os.path.join(ROOT, "counter")


def main():
    build("counter")
    pid, fd = spawn(BIN, rows=24, cols=80)
    screen = Screen(24, 80)
    try:
        # Initial render
        wait_for(fd, screen, "Count: 0")
        assert "Ard + Vaxis Counter" in screen.text()
        assert "q quits" in screen.text()

        # Increment via +
        send(fd, "+")
        wait_for(fd, screen, "Count: 1")
        assert "Incremented" in screen.text()

        # Increment via up arrow
        send(fd, "\x1b[A")
        wait_for(fd, screen, "Count: 2")

        # Increment via right arrow
        send(fd, "\x1b[C")
        wait_for(fd, screen, "Count: 3")

        # Decrement via -
        send(fd, "-")
        wait_for(fd, screen, "Count: 2")
        assert "Decremented" in screen.text()

        # Decrement via down arrow
        send(fd, "\x1b[B")
        wait_for(fd, screen, "Count: 1")

        # Decrement via left arrow
        send(fd, "\x1b[D")
        wait_for(fd, screen, "Count: 0")

        # Vim keys: k = up (increment), j = down (decrement)
        send(fd, "k")
        wait_for(fd, screen, "Count: 1")
        send(fd, "j")
        wait_for(fd, screen, "Count: 0")

        # Reset
        send(fd, "+")
        wait_for(fd, screen, "Count: 1")
        send(fd, "r")
        wait_for(fd, screen, "Count: 0")
        assert "Reset" in screen.text()

        # Quit
        send(fd, "q")
        status = wait_exit(pid, fd, screen, timeout=2.0)
        if status is None:
            send(fd, "\x03")
            raise AssertionError("counter did not exit after q")
        assert status == 0, f"exit status {status}"

        print("✓ counter smoke test passed")
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
