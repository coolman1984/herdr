#!/usr/bin/env python3
"""Random goal plan for the soak run: N tasks, random deps, faults and shared-file conflicts.
   genplan.py N SEED [secret]  -> plan JSON on stdout"""
import json, random, sys
n, seed = int(sys.argv[1]), int(sys.argv[2])
secret = len(sys.argv) > 3
rnd = random.Random(seed)
faults = ["FAULT:trust", "FAULT:crash", "FAULT:idle", "FAULT:limit", "FAULT:capacity", "FAULT:menu", "REVIEW-FAIL-ONCE", "", ""]
tasks = []
for i in range(1, n + 1):
    tid = f"T{i}"
    deps = sorted(rnd.sample([t["id"] for t in tasks], k=min(len(tasks), rnd.choice([0, 0, 1, 2]))))
    f = " ".join(sorted(set(rnd.sample(faults, 2)) - {""}))
    if secret and i == n:
        f = "FAULT:secret"
    sh = (f"mkdir -p notes tests\necho '{tid} done' > notes/{tid}.txt\ncat notes/*.txt > NOTES.md\n"
          f"cat > tests/test_{tid.lower()}.py <<'PY'\nimport os, unittest\nclass T(unittest.TestCase):\n"
          f"    def test_{tid.lower()}(self):\n        self.assertTrue(os.path.exists('notes/{tid}.txt'))\nPY\n")
    tasks.append({"id": tid, "title": f"note {tid}", "deps": deps, "writer": rnd.choice(["alpha", "beta"]),
                  "spec": f"Add note {tid} and its test. {f}\n```sh\n{sh}```\n```resolve\ncat notes/*.txt > NOTES.md\n```",
                  "accept": ["python3 -m unittest discover -s tests -q"]})
print(json.dumps({"tasks": tasks}, indent=1))
