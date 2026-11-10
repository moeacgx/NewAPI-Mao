#!/usr/bin/env bash
set -euo pipefail
TAG='v1.0.0-rc.10.1.10.364'
EXPECT_SHA='7a07eb20d01808f23b0d6641d38885443ba2cc119e6c946d64e6cd31860c8ec2'
ASSET="new-api-${TAG}"
WORK="/apps/maolaoapi/releases/364"
STAMP=$(date -u +%Y%m%dT%H%M%SZ)
BACKUP_ROOT="/apps/maolaoapi/backups/pre-364-${STAMP}"

echo "## tools"
command -v gh || true
command -v curl
uname -m

echo "## create dirs"
sudo mkdir -p "$WORK" "$BACKUP_ROOT/binaries"
sudo chmod 700 "$WORK" "$BACKUP_ROOT" "$BACKUP_ROOT/binaries"

if [[ ! -f "$WORK/$ASSET.sha256ok" ]]; then
  echo "## need download of $ASSET"
  if command -v gh >/dev/null 2>&1; then
    echo "gh present; will download in next step with token from stdin/env"
  else
    echo "gh missing"
  fi
else
  echo "## already prepared"
  cat "$WORK/$ASSET.sha256ok"
fi

echo "## backup compose snapshot"
sudo cp -a /apps/maolaoapi/docker-compose.json "$BACKUP_ROOT/docker-compose.json"
sudo sha256sum /apps/maolaoapi/docker-compose.json | sudo tee "$BACKUP_ROOT/docker-compose.json.sha256" >/dev/null

for c in maolaoapi maolaoapi-slave-1 maolaoapi-slave-2; do
  echo "## backup binary $c"
  sudo docker exec "$c" cat /proc/1/exe | sudo tee "$BACKUP_ROOT/binaries/${c}.running" >/dev/null
  sudo chmod 0755 "$BACKUP_ROOT/binaries/${c}.running"
  sudo sha256sum "$BACKUP_ROOT/binaries/${c}.running" | sudo tee "$BACKUP_ROOT/binaries/${c}.running.sha256" >/dev/null
  (
    cd /tmp
    sudo env -u VERSION "$BACKUP_ROOT/binaries/${c}.running" --version
  ) | sudo tee "$BACKUP_ROOT/binaries/${c}.running.version" >/dev/null
done

echo "## backup postgres custom dump (may take minutes)"
DB_USER=$(sudo docker exec maolaoapi-postgres printenv POSTGRES_USER)
DB_NAME=$(sudo docker exec maolaoapi-postgres printenv POSTGRES_DB)
sudo docker exec maolaoapi-postgres pg_dump -Fc -U "$DB_USER" "$DB_NAME" | sudo tee "$BACKUP_ROOT/postgres.dump" >/dev/null
sudo docker exec -i maolaoapi-postgres pg_restore --list < "$BACKUP_ROOT/postgres.dump" | sudo tee "$BACKUP_ROOT/postgres.dump.toc" >/dev/null
sudo sha256sum "$BACKUP_ROOT/postgres.dump" | sudo tee "$BACKUP_ROOT/postgres.dump.sha256" >/dev/null
sudo sha256sum "$BACKUP_ROOT/postgres.dump.toc" | sudo tee "$BACKUP_ROOT/postgres.dump.toc.sha256" >/dev/null
sudo ls -la "$BACKUP_ROOT" "$BACKUP_ROOT/binaries"
echo "BACKUP_ROOT=$BACKUP_ROOT"
echo "WORK=$WORK"
