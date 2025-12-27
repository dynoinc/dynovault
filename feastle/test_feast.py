#!/usr/bin/env python3
# /// script
# requires-python = ">=3.11"
# dependencies = [
#     "feast[aws]>=0.47.0",
#     "pandas>=1.5.3",
# ]
# ///
"""
Feast integration test for dynovault.

Builds and starts the Go server, initializes a Feast repo, runs the
generated test workflow, then tears down.

Usage: uv run feastle/test_feast.py
"""

import atexit
import os
import shutil
import signal
import socket
import subprocess
import sys
import tempfile
import time
from pathlib import Path

# Paths
ROOT_DIR = Path(__file__).parent.parent
FEASTLE_DIR = Path(__file__).parent
SERVER_BINARY = ROOT_DIR / "feastle" / "server"
FEATURE_STORE_YAML = FEASTLE_DIR / "feature_store.yaml"

# Server config
SERVER_ADDR = "127.0.0.1:8779"
SERVER_URL = f"http://{SERVER_ADDR}"

# Global state for cleanup
_server_process = None
_temp_dir = None


def cleanup():
    """Clean up server and temp files."""
    global _server_process, _temp_dir

    if _server_process:
        print("\n--- Stopping server ---")
        _server_process.terminate()
        try:
            _server_process.wait(timeout=5)
        except subprocess.TimeoutExpired:
            _server_process.kill()
        _server_process = None

    if _temp_dir and Path(_temp_dir).exists():
        shutil.rmtree(_temp_dir, ignore_errors=True)
        _temp_dir = None

    if SERVER_BINARY.exists():
        SERVER_BINARY.unlink()


def signal_handler(signum, frame):
    """Handle Ctrl+C gracefully."""
    print("\nInterrupted, cleaning up...")
    cleanup()
    sys.exit(1)


def build_server():
    """Build the Go server."""
    print("--- Building server ---")
    subprocess.run(
        ["go", "build", "-o", str(SERVER_BINARY), "./server"],
        cwd=ROOT_DIR,
        check=True,
    )


def start_server():
    """Start the server and wait for it to be ready."""
    global _server_process

    print("--- Starting server ---")
    env = os.environ.copy()
    env["AWS_ACCESS_KEY_ID"] = "id"
    env["AWS_SECRET_ACCESS_KEY"] = "key"

    _server_process = subprocess.Popen(
        [str(SERVER_BINARY), "-addr", SERVER_ADDR],
        env=env,
        stdout=subprocess.DEVNULL,
        stderr=subprocess.DEVNULL,
    )

    # Wait for server to be ready
    for _ in range(30):
        try:
            with socket.create_connection(
                (SERVER_ADDR.split(":")[0], int(SERVER_ADDR.split(":")[1])), timeout=1
            ):
                print(f"Server ready at {SERVER_URL}")
                return
        except (ConnectionRefusedError, socket.timeout, OSError):
            time.sleep(0.1)

    raise RuntimeError("Server failed to start")


def setup_feast_repo():
    """Initialize Feast repo in temp directory."""
    global _temp_dir

    print("--- Setting up Feast repo ---")
    _temp_dir = tempfile.mkdtemp(prefix="feastle_")

    # Initialize feast repo
    subprocess.run(
        ["feast", "init", "feature_repo"],
        cwd=_temp_dir,
        check=True,
        capture_output=True,
    )

    # feast init creates nested structure: _temp_dir/feature_repo/feature_repo/
    repo_dir = Path(_temp_dir) / "feature_repo" / "feature_repo"

    # Copy our feature_store.yaml (overwrite the generated one)
    shutil.copy(FEATURE_STORE_YAML, repo_dir / "feature_store.yaml")

    return repo_dir


def run_feast_workflow(repo_dir: Path):
    """Run the Feast test workflow."""
    os.chdir(repo_dir)

    # Set AWS credentials for Feast
    env = os.environ.copy()
    env["AWS_ACCESS_KEY_ID"] = "id"
    env["AWS_SECRET_ACCESS_KEY"] = "key"
    env["AWS_DEFAULT_REGION"] = "us-west-2"

    print("\n--- Running test_workflow.py ---")
    result = subprocess.run(
        [sys.executable, "test_workflow.py"],
        capture_output=True,
        text=True,
        env=env,
    )
    print(result.stdout)
    if result.stderr:
        print(result.stderr)
    if result.returncode != 0:
        raise RuntimeError(f"test_workflow.py failed with exit code {result.returncode}")


def main():
    # Register cleanup handlers
    atexit.register(cleanup)
    signal.signal(signal.SIGINT, signal_handler)
    signal.signal(signal.SIGTERM, signal_handler)

    try:
        build_server()
        start_server()
        repo_dir = setup_feast_repo()
        run_feast_workflow(repo_dir)
        print("\n--- Success! ---")
    except Exception as e:
        print(f"\n--- Failed: {e} ---")
        sys.exit(1)
    finally:
        cleanup()


if __name__ == "__main__":
    main()
