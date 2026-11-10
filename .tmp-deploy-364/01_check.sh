#!/usr/bin/env bash
set -euo pipefail
for c in maolaoapi maolaoapi-slave-1 maolaoapi-slave-2; do
  echo "## $c"
  docker inspect "$c" --format 'id={{.Id}} health={{if .State.Health}}{{.State.Health.Status}}{{end}} restart={{.RestartCount}} readonly={{.HostConfig.ReadonlyRootfs}} policy={{.HostConfig.RestartPolicy.Name}} image={{.Config.Image}}'
  docker exec -w /tmp "$c" env -u VERSION /proc/1/exe --version
  docker exec "$c" sh -ceu 'printf "pid1="; readlink /proc/1/exe; sha256sum /proc/1/exe /new-api'
done
for p in 18095 18100 18101; do
  echo "## status $p"
  curl -fsS --max-time 5 "http://127.0.0.1:${p}/api/status" | python3 -c 'import sys,json; d=json.load(sys.stdin)["data"]; print(d.get("version"), "theme=", d.get("theme"))'
done
