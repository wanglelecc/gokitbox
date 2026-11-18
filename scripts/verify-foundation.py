#!/usr/bin/env python3
"""Run the versioned acceptance artifact in a disposable copy, never altering local tests."""
import hashlib, json, os, pathlib, shutil, subprocess, tarfile, tempfile
root = pathlib.Path(__file__).resolve().parents[1]
manifest = json.loads((root / 'verification/foundation-suite-v1.json').read_text())
archive = root / 'verification/foundation-suite-v1.tar.gz'
if hashlib.sha256(archive.read_bytes()).hexdigest() != manifest['archive_sha256']:
    raise SystemExit('acceptance archive checksum mismatch')
def fingerprint(base):
    paths = [p for p in base.rglob('*.go') if not p.name.endswith('_test.go') and '.git' not in p.parts]
    paths += [base / 'go.mod', base / 'go.sum']
    h = hashlib.sha256()
    for p in sorted(paths, key=lambda p: p.relative_to(base).as_posix()):
        h.update(p.relative_to(base).as_posix().encode() + b'\0' + p.read_bytes() + b'\0')
    return h.hexdigest()
if fingerprint(root) != manifest['production_sha256']:
    raise SystemExit('production sources differ from the accepted suite; regenerate artifact after review')
if shutil.which('redis-server') is None:
    raise SystemExit('redis-server is required for isolated failure tests')
with tempfile.TemporaryDirectory(prefix='gokitbox-acceptance-') as temporary:
    work = pathlib.Path(temporary) / 'module'
    shutil.copytree(root, work, ignore=shutil.ignore_patterns('.git', '*_test.go', 'logs', 'log', 'bin', 'verification'))
    with tarfile.open(archive) as suite:
        suite.extractall(work, filter='data')
    env = dict(os.environ, GOSUMDB='sum.golang.org')
    for args in [['go', 'build', './...'], ['go', 'build', '-tags=redis,jsoniter', './...'], ['go', 'vet', './...'], ['go', 'test', '-race', './...', '-count=1', '-timeout=120s']]:
        subprocess.run(args, cwd=work, env=env, check=True)
