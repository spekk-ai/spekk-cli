#!/usr/bin/env python3
"""
Spekk Agent Client

Runs inside a sandbox droplet. Maintains a WebSocket connection to Django,
receives messages, invokes Claude Code per-message, and streams output back.

Configuration via environment variables (loaded from /etc/spekk/agent.env):
  SPEKK_AGENT_TOKEN  — auth token for WebSocket connection
  SPEKK_HOST         — Spekk app host (e.g. spekk-staging.herokuapp.com)
  ANTHROPIC_API_KEY  — for Claude Code (read from env by Claude itself)
"""

import asyncio
import json
import logging
import os
import signal
import sys

logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s %(levelname)s %(message)s",
)
log = logging.getLogger("agent-client")

try:
    import websockets
except ImportError:
    log.error("websockets package not installed. Run: pip3 install websockets")
    sys.exit(1)

SPEKK_AGENT_TOKEN = os.environ.get("SPEKK_AGENT_TOKEN", "")
SPEKK_HOST = os.environ.get("SPEKK_HOST", "")
WORKSPACE = os.environ.get("WORKSPACE", "/opt/spekk/workspace")

shutdown_event = asyncio.Event()

HEARTBEAT_INTERVAL = 30
RECONNECT_BASE = 3
RECONNECT_MAX = 60


def build_ws_url():
    if not SPEKK_AGENT_TOKEN or not SPEKK_HOST:
        log.error("SPEKK_AGENT_TOKEN and SPEKK_HOST must be set")
        sys.exit(1)
    scheme = "ws" if "localhost" in SPEKK_HOST else "wss"
    return f"{scheme}://{SPEKK_HOST}/ws/agent/{SPEKK_AGENT_TOKEN}/"


class AgentClient:
    def __init__(self):
        self.ws = None
        self.current_process = None
        self.session_id = None

    async def run(self):
        """Main loop: connect, handle messages, reconnect on failure."""
        delay = RECONNECT_BASE
        while not shutdown_event.is_set():
            try:
                url = build_ws_url()
                log.info("Connecting to %s", url.replace(SPEKK_AGENT_TOKEN, "***"))
                async with websockets.connect(url, ping_interval=None) as ws:
                    self.ws = ws
                    delay = RECONNECT_BASE
                    log.info("Connected.")
                    heartbeat_task = asyncio.create_task(self._heartbeat_loop(ws))
                    try:
                        await self.handle_connection(ws)
                    finally:
                        heartbeat_task.cancel()
                        try:
                            await heartbeat_task
                        except asyncio.CancelledError:
                            pass
            except (websockets.ConnectionClosed, ConnectionRefusedError, OSError) as e:
                log.warning("Connection lost: %s. Reconnecting in %ds...", e, delay)
            except Exception:
                log.exception("Unexpected error. Reconnecting in %ds...", delay)

            self.ws = None
            if shutdown_event.is_set():
                break
            await asyncio.sleep(delay)
            delay = min(delay * 2, RECONNECT_MAX)

    async def _heartbeat_loop(self, ws):
        """Send application-level heartbeat messages periodically."""
        while True:
            await asyncio.sleep(HEARTBEAT_INTERVAL)
            try:
                await ws.send(json.dumps({"type": "heartbeat"}))
            except Exception:
                break

    async def handle_connection(self, ws):
        """Handle messages from Django."""
        async for raw in ws:
            try:
                msg = json.loads(raw)
            except json.JSONDecodeError:
                log.warning("Invalid JSON received: %s", raw[:200])
                continue

            msg_type = msg.get("type")
            if msg_type == "message":
                await self.handle_message(ws, msg)
            elif msg_type == "cancel":
                await self.handle_cancel()
            elif msg_type == "heartbeat_ack":
                pass
            else:
                log.warning("Unknown message type: %s", msg_type)

    async def handle_message(self, ws, msg):
        """Invoke Claude Code and stream output back."""
        text = msg.get("text", "")
        system_prompt = msg.get("system_prompt")
        # Only fall back to self.session_id for resumed sessions (no system_prompt).
        # New sessions (system_prompt present) must start fresh.
        session_id = msg.get("session_id") or (self.session_id if not system_prompt else None)
        agent_session_id = msg.get("agent_session_id")

        if not text:
            log.warning("Empty message received, ignoring")
            return

        if system_prompt:
            text = f"{system_prompt}\n\n---\n\nUser message:\n{text}"

        cmd = [
            "claude", "-p", "-",
            "--output-format", "stream-json",
            "--verbose",
            "--include-partial-messages",
            "--dangerously-skip-permissions",
        ]
        if session_id:
            cmd.extend(["--resume", session_id])
        log.info(
            "Invoking Claude: session=%s system_prompt_len=%s",
            session_id or "new",
            len(system_prompt) if system_prompt else 0,
        )

        try:
            self.current_process = await asyncio.create_subprocess_exec(
                *cmd,
                stdin=asyncio.subprocess.PIPE,
                stdout=asyncio.subprocess.PIPE,
                stderr=asyncio.subprocess.PIPE,
                cwd=WORKSPACE,
            )
            self.current_process.stdin.write(text.encode("utf-8"))
            self.current_process.stdin.close()

            last_result_text = ""

            async for line in self.current_process.stdout:
                decoded = line.decode("utf-8", errors="replace").strip()
                if not decoded:
                    continue

                # Forward raw NDJSON line to Django
                await ws.send(json.dumps({
                    "type": "stream",
                    "data": decoded,
                }))

                # Try to extract session ID and result text from result events
                try:
                    event = json.loads(decoded)
                    if event.get("type") == "result":
                        new_session_id = event.get("session_id")
                        if new_session_id:
                            self.session_id = new_session_id
                        last_result_text = event.get("result", "")
                except json.JSONDecodeError:
                    pass

            returncode = await self.current_process.wait()

            if returncode == 0:
                await ws.send(json.dumps({
                    "type": "result",
                    "session_id": self.session_id,
                    "agent_session_id": agent_session_id,
                    "output": last_result_text,
                }))
                log.info("Claude finished: session=%s", self.session_id)
            else:
                stderr = ""
                if self.current_process.stderr:
                    stderr = (await self.current_process.stderr.read()).decode("utf-8", errors="replace")
                await ws.send(json.dumps({
                    "type": "error",
                    "error": f"Claude exited with code {returncode}",
                    "detail": stderr[:2000],
                }))
                log.error("Claude failed: exit=%d stderr=%s", returncode, stderr[:500])

        except asyncio.CancelledError:
            log.info("Message handling cancelled")
            raise
        except Exception as e:
            await ws.send(json.dumps({
                "type": "error",
                "error": str(e),
            }))
            log.exception("Error invoking Claude")
        finally:
            self.current_process = None

    async def handle_cancel(self):
        """Cancel the running Claude process."""
        if self.current_process and self.current_process.returncode is None:
            log.info("Cancelling Claude process")
            self.current_process.send_signal(signal.SIGTERM)


def _handle_signal(sig, _frame):
    log.info("Received signal %s, shutting down...", sig)
    shutdown_event.set()


async def main():
    signal.signal(signal.SIGTERM, _handle_signal)
    signal.signal(signal.SIGINT, _handle_signal)
    os.makedirs(WORKSPACE, exist_ok=True)
    log.info("Starting agent client (host=%s, workspace=%s)", SPEKK_HOST, WORKSPACE)
    client = AgentClient()
    await client.run()
    log.info("Agent client stopped")


if __name__ == "__main__":
    asyncio.run(main())
