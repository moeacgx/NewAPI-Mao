#!/usr/bin/env bash
set -euo pipefail
TAG='v1.0.0-rc.10.1.10.364'
IMAGE="ghcr.io/moeacgx/newapi-mao:${TAG}"
EXPECT_SHA='7a07eb20d01808f23b0d6641d38885443ba2cc119e6c946d64e6cd31860c8ec2'
EXPECT_DIGEST='sha256:438bfb834602316940f67eee9edfddc89d8ba686b92f0fbc58962a9ea5535a87'
WORK='/apps/maolaoapi/releases/364'
STAMP=$(date -u +%Y%m%dT%H%M%SZ)
BACKUP_ROOT="/apps/maolaoapi/backups/pre-364-${STAMP}"
ASSET="new-api-${TAG}"

sudo mkdir -p "$WORK" "$BACKUP_ROOT/binaries"
sudo chmod 700 "$WORK" "$BACKUP_ROOT" "$BACKUP_ROOT/binaries"
echo "BACKUP_ROOT=$BACKUP_ROOT" | sudo tee "$WORK/backup_root.txt" >/dev/null

echo "## pull image"
sudo docker pull "$IMAGE"
echo "## resolve local digest"
sudo docker image inspect "$IMAGE" --format 'Id={{.Id}} RepoDigests={{json .RepoDigests}} Created={{.Created}}'
# Prefer matching digest if present
if sudo docker image inspect "$IMAGE" --format '{{json .RepoDigests}}' | grep -q "$EXPECT_DIGEST"; then
  echo "digest match ok"
else
  echo "WARN: expected digest $EXPECT_DIGEST not listed in RepoDigests; continue with binary sha256 gate"
fi

TMP_CTR="newapi-mao-extract-364-$$"
sudo docker rm -f "$TMP_CTR" >/dev/null 2>&1 || true
sudo docker create --name "$TMP_CTR" "$IMAGE" >/dev/null
sudo docker cp "${TMP_CTR}:/new-api" "$WORK/${ASSET}"
sudo docker rm -f "$TMP_CTR" >/dev/null
sudo chmod 0755 "$WORK/${ASSET}"

ACTUAL=$(sudo sha256sum "$WORK/${ASSET}" | awk '{print $1}')
echo "actual_sha=$ACTUAL"
if [[ "$ACTUAL" != "$EXPECT_SHA" ]]; then
  echo "SHA256 mismatch: got $ACTUAL want $EXPECT_SHA" >&2
  exit 1
fi
VER=$(cd /tmp && sudo env -u VERSION "$WORK/${ASSET}" --version)
echo "binary_version=$VER"
if [[ "$VER" != "$TAG" ]]; then
  echo "version mismatch: got $VER want $TAG" >&2
  exit 1
fi
echo "$ACTUAL  $ASSET" | sudo tee "$WORK/${ASSET}.sha256ok" >/dev/null
echo "$VER" | sudo tee "$WORK/${ASSET}.version" >/dev/null

echo "## backup compose"
sudo cp -a /apps/maolaoapi/docker-compose.json "$BACKUP_ROOT/docker-compose.json"
sudo sha256sum "$BACKUP_ROOT/docker-compose.json" | sudo tee "$BACKUP_ROOT/docker-compose.json.sha256" >/dev/null

echo "## backup running binaries"
for c in maolaoapi maolaoapi-slave-1 maolaoapi-slave-2; do
  sudo docker exec "$c" cat /proc/1/exe | sudo tee "$BACKUP_ROOT/binaries/${c}.running" >/dev/null
  sudo chmod 0755 "$BACKUP_ROOT/binaries/${c}.running"
  sudo sha256sum "$BACKUP_ROOT/binaries/${c}.running" | sudo tee "$BACKUP_ROOT/binaries/${c}.running.sha256" >/dev/null
  (cd /tmp && sudo env -u VERSION "$BACKUP_ROOT/binaries/${c}.running" --version) | sudo tee "$BACKUP_ROOT/binaries/${c}.running.version" >/dev/null
done

echo "## backup postgres"
DB_USER=$(sudo docker exec maolaoapi-postgres printenv POSTGRES_USER)
DB_NAME=$(sudo docker exec maolaoapi-postgres printenv POSTGRES_DB)
sudo docker exec maolaoapi-postgres pg_dump -Fc -U "$DB_USER" "$DB_NAME" | sudo tee "$BACKUP_ROOT/postgres.dump" >/dev/null
sudo docker exec -i maolaoapi-postgres pg_restore --list < "$BACKUP_ROOT/postgres.dump" | sudo tee "$BACKUP_ROOT/postgres.dump.toc" >/dev/null
sudo sha256sum "$BACKUP_ROOT/postgres.dump" | sudo tee "$BACKUP_ROOT/postgres.dump.sha256" >/dev/null
sudo ls -lah "$WORK" "$BACKUP_ROOT" "$BACKUP_ROOT/binaries"
echo "READY backup=$BACKUP_ROOT asset=$WORK/$ASSET sha=$ACTUAL"
