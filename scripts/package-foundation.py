#!/usr/bin/env python3
import gzip, hashlib, io, json, pathlib, tarfile
root=pathlib.Path(__file__).resolve().parents[1]
files=sorted(root.rglob('*_test.go'))
if not files: raise SystemExit('no local test sources to package')
raw=io.BytesIO()
with tarfile.open(fileobj=raw,mode='w') as archive:
    for p in files:
        entry=tarfile.TarInfo(p.relative_to(root).as_posix());content=p.read_bytes()
        entry.size=len(content);entry.mode=0o644;entry.mtime=0;archive.addfile(entry,io.BytesIO(content))
packed=gzip.compress(raw.getvalue(),mtime=0)
(root/'verification/foundation-suite-v1.tar.gz').write_bytes(packed)
h=hashlib.sha256()
paths=[p for p in root.rglob('*.go') if not p.name.endswith('_test.go') and '.git' not in p.parts]+[root/'go.mod',root/'go.sum']
for p in sorted(paths,key=lambda p:p.relative_to(root).as_posix()):h.update(p.relative_to(root).as_posix().encode()+b'\0'+p.read_bytes()+b'\0')
(root/'verification/foundation-suite-v1.json').write_text(json.dumps({'schema':1,'test_files':len(files),'production_sha256':h.hexdigest(),'archive_sha256':hashlib.sha256(packed).hexdigest()},indent=2)+'\n')
