# server.py — Python rule fixtures (VX-SEC-004/003/013).
import os
import pickle
import subprocess
import yaml


def run_query(expr):
    # flagged (high): eval of tool input
    return eval(expr)


def run_block(src):
    # flagged (high): exec
    exec(src)


def disk_usage():
    # flagged (high): interpreter shell
    return os.system("df -h")


def shell_run(cmd):
    # flagged (high): shell=True
    return subprocess.run(cmd, shell=True)


def load_state(blob):
    # flagged (critical): pickle deserialization
    return pickle.loads(blob)


def load_cfg(text):
    # flagged (critical): yaml.load without SafeLoader
    return yaml.load(text)
