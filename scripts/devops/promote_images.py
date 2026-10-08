#!/usr/bin/env python3
"""Update only the staging image overlay after every CI gate has passed."""
import argparse
import os
from pathlib import Path
import re
import subprocess
import tempfile

def image_values(registry, revision):
    if not re.fullmatch(r"[0-9a-f]{40}", revision):
        raise ValueError("revision must be a full immutable git SHA")
    if not re.fullmatch(r"[a-zA-Z0-9][a-zA-Z0-9.:/_-]*", registry):
        raise ValueError("invalid registry path")
    return "".join(f'{role}:\n  image:\n    repository: {registry}/{image}\n    tag: "{revision}"\n' for role, image in [("api", "unihub-backend"), ("worker", "unihub-backend"), ("web", "unihub-web")])

def git(*args, cwd, env=None):
    return subprocess.check_output(["git", *args], cwd=cwd, env=env, text=True).strip()

def main():
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument("--registry", required=True)
    p.add_argument("--revision", required=True)
    p.add_argument("--repository", required=True)
    p.add_argument("--push", action="store_true")
    args = p.parse_args()
    values = image_values(args.registry, args.revision)
    # Credentials stay in an askpass process; never put a token into a Git URL.
    with tempfile.TemporaryDirectory() as directory:
        env = dict(os.environ)
        askpass = Path(directory) / "askpass.sh"
        askpass.write_text('#!/bin/sh\ncase "$1" in *Username*) echo x-access-token;; *) printf "%s" "$GIT_TOKEN";; esac\n')
        askpass.chmod(0o700)
        env.update(GIT_ASKPASS=str(askpass), GIT_TERMINAL_PROMPT="0")
        subprocess.run(["git", "clone", "--single-branch", "--branch", "main", args.repository, directory + "/repo"], check=True, env=env)
        checkout = Path(directory) / "repo"
        current = git("rev-parse", "HEAD", cwd=checkout)
        # Do not let an older successful workflow replace a newer commit.
        if current != args.revision:
            raise SystemExit("Stale workflow: main advanced; rerun CI for current HEAD")
        target = checkout / "deploy/helm/unihub/images-staging.yaml"
        target.write_text(values)
        git("config", "user.name", "UniHub CI", cwd=checkout)
        git("config", "user.email", "ci@unihub.example", cwd=checkout)
        git("add", "deploy/helm/unihub/images-staging.yaml", cwd=checkout)
        if not git("diff", "--cached", "--name-only", cwd=checkout):
            print("Image overlay is already current")
            return
        git("commit", "-m", f"ci: promote staging images {args.revision} [skip ci]", cwd=checkout)
        if args.push:
            if not env.get("GIT_TOKEN"):
                raise SystemExit("GIT_TOKEN is required for --push")
            # A branch race is rejected by normal fast-forward push.
            git("push", "origin", "HEAD:main", cwd=checkout, env=env)
            print("Published staging image overlay")
        else:
            print(values, end="")
if __name__ == "__main__":
    main()
