#!/usr/bin/env python3
"""Shared PTY test harness for vaxis-ard example smoke tests."""

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

ARD_CMD = shlex.split(os.environ.get("ARD", "ard"))


# ── Screen emulator ────────────────────────────────────────────────────

class Screen:
    """Minimal terminal screen emulator for smoke-test assertions."""

    def __init__(self, rows=24, cols=80):
        self.rows = rows
        self.cols = cols
        self.row = 0
        self.col = 0
        self.cells = [[" " for _ in range(cols)] for _ in range(rows)]

    def text(self):
        return "\n".join("".join(row).rstrip() for row in self.cells)

    def line(self, n):
        """Return the n-th row (0-indexed) as a string."""
        return "".join(self.cells[n]).rstrip()

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
        # SGR (Select Graphic Rendition) — just consume it.
        if final == "m":
            return
        clean = body.lstrip("?")
        parts = [p for p in clean.split(";") if p]
        nums = []
        for part in parts:
            # Colons denote sub-parameters (e.g. 38:5:89 for indexed fg color).
            # Take only the first sub-param for cursor/clear operations.
            first = part.split(":")[0]
            try:
                nums.append(int(first))
            except ValueError:
                nums.append(0)
        if final in "Hf":
            row = nums[0] if len(nums) >= 1 and nums[0] else 1
            col = nums[1] if len(nums) >= 2 and nums[1] else 1
            self.row = max(0, min(self.rows - 1, row - 1))
            self.col = max(0, min(self.cols - 1, col - 1))
        elif final == "J" and (not nums or nums[0] in (2, 3)):
            self.cells = [[" " for _ in range(self.cols)] for _ in range(self.rows)]
            self.row = 0
            self.col = 0
        elif final == "m":
            pass


# ── Build / spawn / IO ─────────────────────────────────────────────────

def build(name):
    """Build an example binary."""
    subprocess.run(
        [*ARD_CMD, "build", "--out", name, f"{name}.ard"],
        cwd=os.path.dirname(os.path.abspath(__file__)),
        check=True,
    )


def spawn(binary, rows=24, cols=80):
    """Fork a PTY and exec the binary. Returns (pid, fd)."""
    pid, fd = pty.fork()
    if pid == 0:
        os.environ.setdefault("TERM", "xterm-256color")
        os.environ.setdefault("VAXIS_LOG_LEVEL", "error")
        os.execv(binary, [binary])
        sys.exit(1)
    _set_winsize(fd, rows, cols)
    return pid, fd


def _set_winsize(fd, rows, cols):
    fcntl.ioctl(fd, termios.TIOCSWINSZ, struct.pack("HHHH", rows, cols, 0, 0))


def read_for(fd, screen, seconds=0.05):
    """Read available output, update screen, answer terminal queries."""
    data = b""
    ready, _, _ = select.select([fd], [], [], seconds)
    if ready:
        try:
            data = os.read(fd, 65536)
        except OSError:
            pass
    if data:
        _respond_to_queries(fd, data)
        if screen is not None:
            screen.feed(data)
    return data


def _respond_to_queries(fd, data):
    """Answer vaxis terminal capability queries so startup doesn't hang."""
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
    # CSIu — kitty keyboard protocol ack
    if b"\x1b[?u" in data or b"\x1b[?1u" in data:
        responses.append(b"\x1b[?1u")
    for r in responses:
        os.write(fd, r)


def wait_for(fd, screen, needle, timeout=4.0):
    """Read until `needle` appears in the screen text."""
    deadline = time.time() + timeout
    while time.time() < deadline:
        read_for(fd, screen, 0.05)
        if needle in screen.text():
            return screen.text()
    raise AssertionError(
        f"did not see {needle!r} after {timeout}s\nscreen:\n{screen.text()}"
    )


def send(fd, text):
    os.write(fd, text.encode())


def wait_exit(pid, fd, screen=None, timeout=2.0):
    """Wait for the child process to exit after sending 'q'. Returns exit status."""
    deadline = time.time() + timeout
    while time.time() < deadline:
        done, status = os.waitpid(pid, os.WNOHANG)
        if done == pid:
            return status
        if screen:
            read_for(fd, screen, 0.05)
    return None


def drain(fd, screen, seconds=0.3):
    """Read and discard output for `seconds` to let rendering settle."""
    deadline = time.time() + seconds
    while time.time() < deadline:
        read_for(fd, screen, 0.03)
