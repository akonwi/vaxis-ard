#!/usr/bin/env python3
"""Smoke test for the tic-tac-toe example — select, navigate, restart, quit."""

import os, signal, sys, time
from test_harness import Screen, build, spawn, read_for, wait_for, send, drain, wait_exit

ROOT = os.path.dirname(os.path.abspath(__file__))
BIN = os.path.join(ROOT, "tic_tac_toe")


def main():
    build("tic_tac_toe")
    pid, fd = spawn(BIN, rows=24, cols=100)
    screen = Screen(24, 100)
    try:
        # Initial render
        wait_for(fd, screen, "Player X to move")
        text = screen.text()
        assert "[ ]" in text, f"empty board expected:\n{text}"
        assert "Player X" in text

        # Select top-left (already focused)
        send(fd, "\r")
        drain(fd, screen, 0.3)
        text = screen.text()
        assert "Player O" in text, f"turn didn't change:\n{text}"
        assert "[X]" in text, f"X not placed:\n{text}"

        # Navigate right, place O
        send(fd, "\x1b[C")
        send(fd, "\r")
        drain(fd, screen, 0.3)
        text = screen.text()
        assert "[O]" in text, f"O not placed:\n{text}"

        # Restart
        send(fd, "r")
        drain(fd, screen, 0.3)
        text = screen.text()
        assert "New game" in text, f"no new game:\n{text}"
        assert "Player X to move" in text

        # Navigate with vim keys
        send(fd, "l")  # right
        send(fd, "\r")
        drain(fd, screen, 0.3)
        text = screen.text()
        assert "Player O" in text

        # Quit
        send(fd, "q")
        status = wait_exit(pid, fd, screen, timeout=2.0)
        if status is None:
            send(fd, "\x03")
            raise AssertionError("did not exit after q")
        assert status == 0, f"exit status {status}"

        print("✓ tic-tac-toe smoke test passed")
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
