#!/usr/bin/env python3
"""
Spekk Agent Client

WebSocket client that connects to the spekk server, receives task
assignments, and executes them using Claude Code CLI in the sandbox.
"""

import asyncio
import json
import logging
import os
import signal
import subprocess
import sys

try:
    import websockets
except ImportError:
    print("ERROR: websockets package not installed. Run: pip3 install websockets", file=sys.stderr)
    sys.exit(1)

logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s [%(levelname)s] %(message)s",
)
logger = logging.getLogger("spekk-agent")

SERVER_URL = os.environ.get("SPEKK_SERVER_URL", "ws://localhost:8080")
AGENT_NAME = os.environ.get("SPEKK_AGENT_NAME", "sandbox-agent")
WORK_DIR = os.environ.get("SPEKK_WORK_DIR", "/opt/spekk/workspace")
RECONNECT_DELAY = int(os.environ.get("SPEKK_RECONNECT_DELAY", "5"))

shutdown_event = asyncio.Event()


def handle_signal(sig, frame):
    """Handle shutdown signals gracefully."""
    logger.info("Received signal %s, shutting down...", sig)
    shutdown_event.set()


signal.signal(signal.SIGTERM, handle_signal)
signal.signal(signal.SIGINT, handle_signal)


async def execute_task(task):
    """Execute a task using Claude Code CLI."""
    task_id = task.get("id", "unknown")
    prompt = task.get("prompt", "")
    repo = task.get("repo", "")

    logger.info("Executing task %s", task_id)

    work_dir = WORK_DIR
    if repo:
        repo_name = repo.split("/")[-1].replace(".git", "")
        work_dir = os.path.join(WORK_DIR, repo_name)
        if not os.path.exists(work_dir):
            logger.info("Cloning repo %s", repo)
            proc = await asyncio.create_subprocess_exec(
                "git", "clone", repo, work_dir,
                stdout=asyncio.subprocess.PIPE,
                stderr=asyncio.subprocess.PIPE,
            )
            await proc.wait()

    try:
        proc = await asyncio.create_subprocess_exec(
            "claude", "--print", "--dangerously-skip-permissions", "-p", prompt,
            cwd=work_dir,
            stdout=asyncio.subprocess.PIPE,
            stderr=asyncio.subprocess.PIPE,
        )
        stdout, stderr = await proc.communicate()

        return {
            "task_id": task_id,
            "status": "completed" if proc.returncode == 0 else "failed",
            "exit_code": proc.returncode,
            "stdout": stdout.decode("utf-8", errors="replace"),
            "stderr": stderr.decode("utf-8", errors="replace"),
        }
    except Exception as e:
        logger.error("Task %s failed: %s", task_id, e)
        return {
            "task_id": task_id,
            "status": "error",
            "error": str(e),
        }


async def agent_loop():
    """Main agent loop: connect to server and process tasks."""
    while not shutdown_event.is_set():
        try:
            logger.info("Connecting to %s as %s", SERVER_URL, AGENT_NAME)
            async with websockets.connect(SERVER_URL) as ws:
                # Register with the server
                await ws.send(json.dumps({
                    "type": "register",
                    "agent": AGENT_NAME,
                }))
                logger.info("Connected and registered")

                async for message in ws:
                    if shutdown_event.is_set():
                        break

                    data = json.loads(message)
                    msg_type = data.get("type", "")

                    if msg_type == "task":
                        result = await execute_task(data)
                        await ws.send(json.dumps({
                            "type": "result",
                            **result,
                        }))
                    elif msg_type == "ping":
                        await ws.send(json.dumps({"type": "pong"}))
                    else:
                        logger.warning("Unknown message type: %s", msg_type)

        except websockets.exceptions.ConnectionClosed:
            logger.warning("Connection closed, reconnecting in %ds...", RECONNECT_DELAY)
        except ConnectionRefusedError:
            logger.warning("Connection refused, retrying in %ds...", RECONNECT_DELAY)
        except Exception as e:
            logger.error("Unexpected error: %s, retrying in %ds...", e, RECONNECT_DELAY)

        if not shutdown_event.is_set():
            await asyncio.sleep(RECONNECT_DELAY)


def main():
    """Entry point."""
    os.makedirs(WORK_DIR, exist_ok=True)
    logger.info("Spekk agent client starting (server=%s, name=%s)", SERVER_URL, AGENT_NAME)
    asyncio.run(agent_loop())
    logger.info("Agent client stopped")


if __name__ == "__main__":
    main()
