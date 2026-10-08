#!/usr/bin/env bash
set -euo pipefail
namespace=${1:?Usage: create-secrets.sh NAMESPACE [SECRET_NAME]}
secret_name=${2:-unihub-secret}
if kubectl -n "$namespace" get secret "$secret_name" >/dev/null 2>&1; then
  echo "Using existing $namespace/$secret_name"
  exit 0
fi
work_dir=$(mktemp -d)
trap 'rm -rf "$work_dir"' EXIT
chmod 700 "$work_dir"
umask 077
openssl rand -hex 24 > "$work_dir/DB_PASSWORD"
openssl rand -hex 32 > "$work_dir/AUTH_SECRET"
openssl rand -hex 24 > "$work_dir/RABBITMQ_PASSWORD"
openssl rand -hex 24 > "$work_dir/REDIS_PASSWORD"
openssl genpkey -algorithm RSA -pkeyopt rsa_keygen_bits:2048 -out "$work_dir/RSA_PRIVATE_KEY" 2>/dev/null
# Hex credentials are safe in the AMQP URL; arbitrary passwords need URL encoding.
printf 'amqp://unihub:%s@rabbitmq:5672/' "$(cat "$work_dir/RABBITMQ_PASSWORD")" > "$work_dir/RABBITMQ_URL"
# Remove newlines: password secrets must match server values byte for byte.
for key in DB_PASSWORD AUTH_SECRET RABBITMQ_PASSWORD REDIS_PASSWORD; do
  tr -d '\n' < "$work_dir/$key" > "$work_dir/trimmed"
  mv "$work_dir/trimmed" "$work_dir/$key"
done
kubectl -n "$namespace" create secret generic "$secret_name" --from-file="$work_dir" >/dev/null
echo "Created $namespace/$secret_name"
