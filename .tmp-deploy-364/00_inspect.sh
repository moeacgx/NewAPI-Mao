#!/usr/bin/env bash
set -euo pipefail
echo "## releases"
sudo ls -la /apps/maolaoapi/releases
echo "## backups"
sudo ls -la /apps/maolaoapi/backups | tail -n 30
echo "## compose images"
sudo python3 - <<'PY'
import json
p=json.load(open('/apps/maolaoapi/docker-compose.json'))
for n,s in p.get('services',{}).items():
    if n.startswith('maolaoapi'):
        print(n, 'image=', s.get('image'), 'ports=', s.get('ports'))
PY
echo "## containers"
docker ps --filter name=maolaoapi --format '{{.Names}} {{.Image}} {{.Status}} {{.Ports}}'
echo "## current image digests"
docker image inspect ghcr.io/moeacgx/newapi-mao:v1.0.0-rc.10.1.10.362 --format 'tags={{json .RepoTags}} digests={{json .RepoDigests}} created={{.Created}}'
echo "## release 362 files"
sudo ls -la /apps/maolaoapi/releases/362 || true
echo "## release 359 files"
sudo ls -la /apps/maolaoapi/releases/359 || true
