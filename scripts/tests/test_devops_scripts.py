"""Exercise deployment scripts without writing to a Kubernetes cluster."""
import json
import os
from pathlib import Path
import shutil
import subprocess
import tempfile
import unittest


ROOT = Path(__file__).resolve().parents[2]


class DevOpsScriptsTest(unittest.TestCase):
    def setUp(self):
        temporary = tempfile.TemporaryDirectory()
        self.addCleanup(temporary.cleanup)
        self.work = Path(temporary.name)
        self.real_helm = shutil.which("helm")
        self.bin = self.work / "bin"
        self.bin.mkdir()
        self.env = dict(
            os.environ,
            PATH=str(self.bin) + os.pathsep + os.environ["PATH"],
            HELM_ARGUMENTS_FILE=str(self.work / "helm-arguments.json"),
            KUBECTL_JOB_FILE=str(self.work / "job.json"),
            CI_NAMESPACE="unihub-ci",
            IMAGE_REGISTRY="registry.example/team",
            GIT_SHA="a" * 40,
        )
        self.executable("helm", """#!/usr/bin/env python3
import json, os, sys
from pathlib import Path
Path(os.environ['HELM_ARGUMENTS_FILE']).write_text(json.dumps(sys.argv[1:]))
""")
        self.executable("kubectl", """#!/usr/bin/env python3
import os, sys
from pathlib import Path
args = sys.argv[1:]
if args[:2] == ['create', '-f']:
    Path(os.environ['KUBECTL_JOB_FILE']).write_text(Path(args[2]).read_text())
    print('unihub-k6-test')
elif 'configmap' in args:
    print('{}')
elif args[:2] == ['apply', '-f']:
    sys.stdin.read()
""")

    def executable(self, name, contents):
        path = self.bin / name
        path.write_text(contents)
        path.chmod(0o755)

    def run_script(self, name, *args, **environment):
        return subprocess.run(
            ["bash", str(ROOT / "scripts/devops" / name), *args],
            cwd=ROOT,
            env=dict(self.env, **environment),
            text=True,
            capture_output=True,
            timeout=30,
        )

    def test_actual_candidate_arguments_render_with_helm_schema(self):
        if not self.real_helm:
            self.skipTest("Helm is required; this test also runs in the manifests gate")
        result = self.run_script("deploy-candidate.sh")
        self.assertEqual(result.returncode, 0, result.stderr)
        arguments = json.loads((self.work / "helm-arguments.json").read_text())
        # Render the exact options used by the deploy script, omitting wait flags.
        arguments = arguments[2:]
        arguments.remove("--wait")
        timeout = arguments.index("--timeout")
        del arguments[timeout:timeout + 2]
        rendered = subprocess.run(
            [self.real_helm, "template", *arguments],
            cwd=ROOT, text=True, capture_output=True, timeout=30,
        )
        self.assertEqual(rendered.returncode, 0, rendered.stderr)
        self.assertIn('RUN_SEED: "false"', rendered.stdout)
        self.assertIn('DB_NAME: "unihub_ci_test"', rendered.stdout)
        self.assertIn("registry-credentials", rendered.stdout)
        self.assertIn("registry.example/team/unihub-backend:" + "a" * 40, rendered.stdout)

    def test_loadtest_job_can_pull_private_ci_fixture(self):
        result = self.run_script("run-loadtest.sh", "unihub-ci", "registry.example/team/backend:sha")
        self.assertEqual(result.returncode, 0, result.stderr)
        job = json.loads((self.work / "job.json").read_text())
        pod = job["spec"]["template"]["spec"]
        self.assertEqual(pod["imagePullSecrets"], [{"name": "registry-credentials"}])
        self.assertEqual(job["metadata"]["namespace"], "unihub-ci")
        self.assertEqual(pod["initContainers"][0]["envFrom"][1], {"secretRef": {"name": "unihub-ci-secret"}})

    def test_demo_fixture_uses_loaded_image_without_registry_credentials(self):
        result = self.run_script("run-loadtest.sh", "unihub-demo", "unihub-backend:sha")
        self.assertEqual(result.returncode, 0, result.stderr)
        job = json.loads((self.work / "job.json").read_text())
        pod = job["spec"]["template"]["spec"]
        self.assertNotIn("imagePullSecrets", pod)
        self.assertEqual(pod["initContainers"][0]["image"], "unihub-backend:sha")

    def test_scripts_reject_production_namespace(self):
        for name, args in [
            ("deploy-candidate.sh", []),
            ("cleanup-candidate.sh", []),
            ("run-loadtest.sh", ["unihub-prod", "backend:sha"]),
        ]:
            with self.subTest(script=name):
                result = self.run_script(name, *args, CI_NAMESPACE="unihub-prod")
                self.assertNotEqual(result.returncode, 0)
                self.assertFalse((self.work / "helm-arguments.json").exists())
                self.assertFalse((self.work / "job.json").exists())


if __name__ == "__main__":
    unittest.main()
