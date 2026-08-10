#!/usr/bin/env bash
# Fires N concurrent p-signup-request calls with DryRun=true to exercise the per-IP lock in
# PostSignUpRequest: one caller holds it, MaxWaiters=2 queue behind it, and every extra caller is
# refused with 429. All the requests leave this machine, so they share the source IP the lock is
# keyed by.
#
#   ./scripts/test_signup_lock.sh [base_url] [email] [count]
#   ./scripts/test_signup_lock.sh https://xxxx.lambda-url.us-east-1.on.aws lock-test@genix.dev 5
#
# DryRun is refused when the backend's `environment` starts with "prod".
set -u

base_url="${1:-http://localhost:14010}"
email="${2:-lock-test@genix.dev}"
request_count="${3:-5}"

endpoint="${base_url%/}/api/p-signup-request"
results_dir="$(mktemp -d)"
trap 'rm -rf "$results_dir"' EXIT

echo "POST $endpoint  x${request_count}  (DryRun, $email)"

for request_number in $(seq 1 "$request_count"); do
	# The same address on every call on purpose: the first one creates the request and the rest land
	# on the resend branch, so repeated runs neither burn the per-IP email quota nor leave new rows.
	curl -s -o "$results_dir/$request_number.body" -w '%{http_code} %{time_total}s' \
		-X POST "$endpoint" \
		-H 'Content-Type: application/json' \
		-d "{\"Email\":\"$email\",\"DryRun\":true}" \
		>"$results_dir/$request_number.status" &
done
wait

for request_number in $(seq 1 "$request_count"); do
	printf '#%s  %s  %s\n' \
		"$request_number" \
		"$(<"$results_dir/$request_number.status")" \
		"$(<"$results_dir/$request_number.body")"
done
