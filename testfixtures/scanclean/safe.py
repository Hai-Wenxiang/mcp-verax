# safe.py — negative fixtures.
import subprocess
import yaml


def read_cfg(text):
    return yaml.safe_load(text)  # safe loader: not flagged


def git_status(cwd):
    return subprocess.run(["git", "status"], cwd=cwd)  # list args, no shell
