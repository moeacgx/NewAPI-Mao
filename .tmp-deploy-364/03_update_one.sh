#!/usr/bin/env bash
set -euo pipefail
CONTAINER="${1:?container name required}"
PORT="${2:?local port required}"
TAG='v1.0.0-rc.10.1.10.364'
EXPECT_SHA='7a07eb20d01808f23b0d6641d38885443ba2cc119e6c946d64e6cd31860c8ec2'
WORK='/apps/maolaoapi/releases/364'
ASSET="$WORK/new-api-${TAG}"

[[ -x "$ASSET" ]] || { echo "missing asset $ASSET" >&2; exit 1; }
ACTUAL=$(sudo sha256sum "$ASSET" | awk '{print $1}')
[[ "$ACTUAL" == "$EXPECT_SHA" ]] || { echo "asset sha mismatch" >&2; exit 1; }

BEFORE_ID=$(sudo docker inspect "$CONTAINER" --format '{{.Id}}')
BEFORE_IMAGE=$(sudo docker inspect "$CONTAINER" --format '{{.Image}}')
BEFORE_PID=$(sudo docker inspect "$CONTAINER" --format '{{.State.Pid}}')
BEFORE_RESTART=$(sudo docker inspect "$CONTAINER" --format '{{.RestartCount}}')
echo "before id=$BEFORE_ID pid=$BEFORE_PID restart=$BEFORE_RESTART"

STAGE="new-api.update.${TAG}.$$"
sudo docker cp "$ASSET" "${CONTAINER}:/tmp/${STAGE}"
sudo docker exec "$CONTAINER" chmod 0755 "/tmp/${STAGE}"
STAGE_SHA=$(sudo docker exec "$CONTAINER" sha256sum "/tmp/${STAGE}" | awk '{print $1}')
[[ "$STAGE_SHA" == "$EXPECT_SHA" ]] || { echo "staged sha mismatch $STAGE_SHA" >&2; exit 1; }
STAGE_VER=$(sudo docker exec -w /tmp "$CONTAINER" env -u VERSION "/tmp/${STAGE}" --version)
[[ "$STAGE_VER" == "$TAG" ]] || { echo "staged version mismatch $STAGE_VER" >&2; exit 1; }

sudo docker exec "$CONTAINER" sh -ceu '
  src=$1
  dst=/new-api
  tmp="${dst}.update-new.$$"
  cp "$src" "$tmp"
  chmod --reference="$dst" "$tmp" 2>/dev/null || chmod 0755 "$tmp"
  chown --reference="$dst" "$tmp" 2>/dev/null || true
  sync "$tmp" || sync
  mv -f "$tmp" "$dst"
  rm -f "$src"
' sh "/tmp/${STAGE}"

NEW_PATH_SHA=$(sudo docker exec "$CONTAINER" sha256sum /new-api | awk '{print $1}')
[[ "$NEW_PATH_SHA" == "$EXPECT_SHA" ]] || { echo "path sha mismatch after replace" >&2; exit 1; }

echo "## restart $CONTAINER"
sudo docker restart --time 20 "$CONTAINER"

for i in $(seq 1 60); do
  STATUS=$(sudo docker inspect "$CONTAINER" --format '{{.State.Status}}|{{if .State.Health}}{{.State.Health.Status}}{{end}}')
  echo "wait $i $STATUS"
  if [[ "$STATUS" == "running|healthy" ]]; then
    break
  fi
  sleep 2
done

AFTER_ID=$(sudo docker inspect "$CONTAINER" --format '{{.Id}}')
AFTER_IMAGE=$(sudo docker inspect "$CONTAINER" --format '{{.Image}}')
AFTER_PID=$(sudo docker inspect "$CONTAINER" --format '{{.State.Pid}}')
AFTER_RESTART=$(sudo docker inspect "$CONTAINER" --format '{{.RestartCount}}')
AFTER_HEALTH=$(sudo docker inspect "$CONTAINER" --format '{{if .State.Health}}{{.State.Health.Status}}{{end}}')
AFTER_VER=$(sudo docker exec -w /tmp "$CONTAINER" env -u VERSION /proc/1/exe --version)
AFTER_SHA=$(sudo docker exec "$CONTAINER" sha256sum /proc/1/exe | awk '{print $1}')
STATUS_JSON=$(curl -fsS --max-time 8 "http://127.0.0.1:${PORT}/api/status" || true)
STATUS_VER=$(python3 -c 'import json,sys; s=sys.stdin.read();
import re
try:
 d=json.loads(s); print(d.get("data",{}).get("version",""))
except Exception:
 print("")' <<<"$STATUS_JSON")

echo "after id=$AFTER_ID pid=$AFTER_PID restart=$AFTER_RESTART health=$AFTER_HEALTH ver=$AFTER_VER sha=$AFTER_SHA status_ver=$STATUS_VER"

[[ "$AFTER_ID" == "$BEFORE_ID" ]] || { echo "container id changed" >&2; exit 1; }
[[ "$AFTER_IMAGE" == "$BEFORE_IMAGE" ]] || { echo "image id changed" >&2; exit 1; }
[[ "$AFTER_PID" != "$BEFORE_PID" ]] || { echo "pid did not change" >&2; exit 1; }
[[ "$AFTER_HEALTH" == "healthy" ]] || { echo "not healthy" >&2; exit 1; }
[[ "$AFTER_VER" == "$TAG" ]] || { echo "runtime version mismatch" >&2; exit 1; }
[[ "$AFTER_SHA" == "$EXPECT_SHA" ]] || { echo "runtime sha mismatch" >&2; exit 1; }
[[ "$STATUS_VER" == "$TAG" ]] || { echo "status version mismatch: $STATUS_VER" >&2; exit 1; }

# basic startup error scan (last 200 lines)
if sudo docker logs --tail 200 "$CONTAINER" 2>&1 | grep -Eiq 'panic|fatal|migration failed|failed to connect to database|FATAL'; then
  echo "WARN: suspicious startup log markers found" >&2
  sudo docker logs --tail 80 "$CONTAINER" 2>&1 | tail -n 80 >&2
  exit 1
fi

echo "OK $CONTAINER -> $TAG"
