#!/usr/bin/env python3
"""Smoke test for the vaxis/ui demo app — multi-page widget showcase."""

import fcntl
import os
import pty
import select
import shlex
import signal
import struct
import subprocess
import sys
import termios
import time

ROOT = os.path.dirname(os.path.abspath(__file__))
BIN = os.path.join(ROOT, "main")
ARD_CMD = shlex.split(os.environ.get("ARD", "ard"))


# ── Screen emulator (same as other test files) ─────────────────────────

class Screen:
    def __init__(self, rows=24, cols=80):
        self.rows = rows
        self.cols = cols
        self.row = 0
        self.col = 0
        self.cells = [[" " for _ in range(cols)] for _ in range(rows)]

    def text(self):
        return "\n".join("".join(row).rstrip() for row in self.cells)

    def feed(self, data):
        text = data.decode("utf-8", errors="ignore").replace("\x00", "")
        i = 0
        while i < len(text):
            ch = text[i]
            if ch == "\x1b":
                i = self._escape(text, i + 1)
                continue
            if ch == "\r":
                self.col = 0
            elif ch == "\n":
                self.row = min(self.rows - 1, self.row + 1)
                self.col = 0
            elif ch == "\b":
                self.col = max(0, self.col - 1)
            elif ch >= " ":
                if 0 <= self.row < self.rows and 0 <= self.col < self.cols:
                    self.cells[self.row][self.col] = ch
                self.col += 1
                if self.col >= self.cols:
                    self.col = 0
                    self.row = min(self.rows - 1, self.row + 1)
            i += 1

    def _escape(self, text, i):
        if i >= len(text):
            return i
        kind = text[i]
        if kind == "[":
            j = i + 1
            while j < len(text) and not ("@" <= text[j] <= "~"):
                j += 1
            if j >= len(text):
                return j
            body = text[i + 1 : j]
            final = text[j]
            self._csi(body, final)
            return j + 1
        if kind in "]P_":
            end = text.find("\x1b\\", i + 1)
            if end == -1:
                bel = text.find("\a", i + 1)
                return len(text) if bel == -1 else bel + 1
            return end + 2
        return i + 1

    def _csi(self, body, final):
        clean = body.lstrip("?")
        parts = [p for p in clean.split(";") if p]
        nums = []
        for part in parts:
            try:
                nums.append(int(part))
            except ValueError:
                nums.append(0)
        if final in "Hf":
            row = nums[0] if len(nums) >= 1 and nums[0] else 1
            col = nums[1] if len(nums) >= 2 and nums[1] else 1
            self.row = max(0, min(self.rows - 1, row - 1))
            self.col = max(0, min(self.cols - 1, col - 1))
        elif final == "J" and (not nums or nums[0] == 2):
            self.cells = [[" " for _ in range(self.cols)] for _ in range(self.rows)]
        elif final == "m":
            pass


# ── Test helpers ───────────────────────────────────────────────────────

def build():
    subprocess.run(ARD_CMD + ["build", "main.ard"], cwd=ROOT, check=True)


def spawn(binary, rows=24, cols=80):
    pid, fd = pty.fork()
    if pid == 0:
        os.environ["TERM"] = "xterm-256color"
        os.environ["VAXIS_LOG_LEVEL"] = "error"
        os.execv(binary, [binary])
        sys.exit(1)
    time.sleep(0.1)
    ws = struct.pack("HHHH", rows, cols, 0, 0)
    fcntl.ioctl(fd, termios.TIOCSWINSZ, ws)
    return pid, fd


def read_for(fd, screen, duration=0.05):
    """Read available output from fd, update screen."""
    data = b""
    ready, _, _ = select.select([fd], [], [], duration)
    if ready:
        try:
            data = os.read(fd, 2048)
        except OSError:
            pass
    if data:
        respond_to_queries(fd, data)
        if screen is not None:
            screen.feed(data)
    return data


def respond_to_queries(fd, data):
    """Answer vaxis terminal capability queries."""
    responses = []
    # DA1 — primary device attributes
    if b"\x1b[c" in data or b"\x1b[0c" in data:
        responses.append(b"\x1b[?62;4;6c")
    # DA3 — tertiary attributes
    if b"\x1b[=c" in data:
        responses.append(b"\x1b[>0;0;0c")
    # DSRCPR — cursor position report
    if b"\x1b[6n" in data:
        responses.append(b"\x1b[1;1R")
    # XTGETTCAP responses
    if b"\x1bP+q" in data:
        # Send a generic 'not supported' response
        responses.append(b"\x1bP0+r\x1b\\")
    # Text area size
    if b"\x1b[14t" in data or b"\x1b[18t" in data:
        pass  # These are requests, we can't easily answer without knowing size
    for r in responses:
        os.write(fd, r)
        time.sleep(0.01)


def wait_for_screen(fd, screen, needle, timeout=6.0):
    """Read until needle appears in screen text, responding to queries."""
    deadline = time.time() + timeout
    while time.time() < deadline:
        data = read_for(fd, screen, 0.05)
        if data:
            respond_to_queries(fd, data)
        if needle in screen.text():
            return screen.text()
    raise AssertionError(
        f"did not see {needle!r} after {timeout}s\nscreen:\n{screen.text()}"
    )


def send(fd, text):
    os.write(fd, text.encode())


# ── Tests ──────────────────────────────────────────────────────────────

def test_demo():
    build()
    pid, fd = spawn(BIN, rows=30, cols=90)
    try:
        screen = Screen(30, 90)

        # Wait for initial render
        wait_for_screen(fd, screen, "Vaxis UI demo")
        text = screen.text()
        assert "Home" in text, f"home tab missing:\n{text}"
        assert "This demo exercises" in text, f"home content missing:\n{text}"
        print("  ✓ home page renders")

        # Navigate through pages
        for page_name in ["Text layout", "Controls", "Lists", "Table", "Theme colors"]:
            send(fd, "n")
            wait_for_screen(fd, screen, page_name)
            print(f"  ✓ {page_name} page")

        # Backward navigation
        send(fd, "p")
        wait_for_screen(fd, screen, "Table")
        print("  ✓ backward navigation")

        # Quit
        send(fd, "q")
        time.sleep(0.3)
        os.waitpid(pid, 0)
        print("  ✓ clean quit")

    finally:
        try:
            os.kill(pid, signal.SIGTERM)
        except OSError:
            pass


if __name__ == "__main__":
    try:
        test_demo()
        print("\n✓ all demo smoke tests passed")
    except AssertionError as e:
        print(f"\n✗ FAIL: {e}", file=sys.stderr)
        sys.exit(1)
