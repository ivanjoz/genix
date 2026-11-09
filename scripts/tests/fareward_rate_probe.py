#!/usr/bin/env python3
"""Probe the deployed Fareward edge: log in as a user, then send GETs at a fixed rate and report
what comes back — status codes, the x-fareward-usage header, Retry-After and the refusal bodies.

    python3 scripts/tests/fareward_rate_probe.py --company 5 --user admin --password '...' \
        --route products --rate 30 --seconds 60

Every request carries the user's session token (Authorization: Bearer). One persistent HTTPS
connection per worker thread, so the rate measures Fareward and the backend, not TLS handshakes.
Prints one line per second and, at the end, every distinct refusal body seen.
"""

import argparse
import collections
import http.client
import json
import re
import secrets
import threading
import time
import urllib.parse

LOGIN_PATH = "/api/p-user-login"


def log_in(host, company_id, user_name, password):
    """POST p-user-login and return the session token. A CipherKey of 32+ characters is required:
    the backend encrypts the user info with it (the token itself travels in clear)."""
    connection = http.client.HTTPSConnection(host, timeout=15)
    request_body = json.dumps({
        "CompanyID": company_id, "User": user_name, "Password": password,
        "CipherKey": secrets.token_hex(16),
    })
    connection.request("POST", LOGIN_PATH, request_body, {"Content-Type": "application/json"})
    response = connection.getresponse()
    response_body = json.loads(response.read() or b"{}")
    if response.status != 200 or not response_body.get("UserToken"):
        raise SystemExit(f"[!] Login failed ({response.status}): {response_body}")
    print(f"[*] Logged in: company {response_body['CompanyID']}, user {response_body['UserID']}")
    return response_body["UserToken"]


class ProbeResults:
    """Thread-safe tallies: per-second status counts, the last usage header, refusal bodies."""

    def __init__(self):
        self.lock = threading.Lock()
        self.status_by_second = collections.defaultdict(collections.Counter)
        self.last_usage_by_second = {}
        self.retry_after_by_second = {}
        self.refusal_bodies = collections.Counter()
        self.refusal_examples = {}
        self.latencies_ms = []

    def record(self, second_index, status, usage_header, retry_after, body_text, latency_ms):
        with self.lock:
            self.status_by_second[second_index][status] += 1
            if usage_header:
                self.last_usage_by_second[second_index] = usage_header
            if retry_after:
                self.retry_after_by_second[second_index] = retry_after
            if status != 200:
                # Refusal pages differ only in their counters: group them with the numbers masked,
                # keeping the first body seen as the example.
                body_shape = re.sub(r"\d+", "N", body_text[:600])
                self.refusal_examples.setdefault((status, body_shape), body_text[:600])
                self.refusal_bodies[(status, body_shape)] += 1
            self.latencies_ms.append(latency_ms)


def run_worker(host, request_path, token, send_times, started_at, results):
    """Send one request at each of send_times (seconds since started_at) over one connection."""
    connection = http.client.HTTPSConnection(host, timeout=15)
    headers = {"Authorization": f"Bearer {token}", "Accept-Encoding": "identity"}
    for send_offset in send_times:
        delay = started_at + send_offset - time.monotonic()
        if delay > 0:
            time.sleep(delay)
        request_started = time.monotonic()
        try:
            connection.request("GET", request_path, headers=headers)
            response = connection.getresponse()
            body_bytes = response.read()
            status = response.status
            usage_header = response.getheader("x-fareward-usage", "")
            retry_after = response.getheader("retry-after", "")
            body_text = body_bytes.decode("utf-8", "replace") if status != 200 else ""
        except (OSError, http.client.HTTPException) as request_error:
            # A refused request may close the connection: reconnect for the next one.
            connection.close()
            connection = http.client.HTTPSConnection(host, timeout=15)
            status, usage_header, retry_after, body_text = 0, "", "", f"{type(request_error).__name__}: {request_error}"
        latency_ms = (time.monotonic() - request_started) * 1000
        results.record(int(send_offset), status, usage_header, retry_after, body_text, latency_ms)
    connection.close()


def main():
    argument_parser = argparse.ArgumentParser(description="Rate-probe the Fareward edge with a user's token.")
    argument_parser.add_argument("--host", default="genix-api-5.un.pe")
    argument_parser.add_argument("--company", type=int, required=True)
    argument_parser.add_argument("--user", required=True)
    argument_parser.add_argument("--password", required=True)
    argument_parser.add_argument("--route", default="products", help="API route without the api/ prefix")
    argument_parser.add_argument("--query", default="", help="query string, e.g. upd=0")
    argument_parser.add_argument("--rate", type=int, default=30, help="requests per second")
    argument_parser.add_argument("--seconds", type=int, default=60)
    argument_parser.add_argument("--workers", type=int, default=10)
    command_arguments = argument_parser.parse_args()

    token = log_in(command_arguments.host, command_arguments.company, command_arguments.user, command_arguments.password)
    request_path = f"/api/{command_arguments.route}"
    if command_arguments.query:
        request_path += "?" + urllib.parse.quote(command_arguments.query, safe="=&")

    # The schedule is evenly spaced and dealt round-robin to the workers, so the aggregate rate is
    # exact even though each connection carries one request at a time.
    total_requests = command_arguments.rate * command_arguments.seconds
    schedule = [index / command_arguments.rate for index in range(total_requests)]
    results = ProbeResults()
    started_at = time.monotonic() + 0.5
    workers = [
        threading.Thread(target=run_worker, args=(
            command_arguments.host, request_path, token, schedule[worker_index::command_arguments.workers],
            started_at, results,
        ))
        for worker_index in range(command_arguments.workers)
    ]
    print(f"[*] GET {request_path}: {command_arguments.rate}/s for {command_arguments.seconds}s, "
          f"{total_requests} requests over {command_arguments.workers} connections")
    for worker in workers:
        worker.start()

    # One line per finished second, while the probe runs.
    reported_second = 0
    while any(worker.is_alive() for worker in workers) or reported_second < command_arguments.seconds:
        if time.monotonic() - started_at >= reported_second + 1.5 or not any(w.is_alive() for w in workers):
            with results.lock:
                status_counts = dict(sorted(results.status_by_second[reported_second].items()))
                usage_header = results.last_usage_by_second.get(reported_second, "-")
                retry_after = results.retry_after_by_second.get(reported_second, "")
            print(f"  t={reported_second:>3}s  {status_counts}  usage={usage_header}"
                  + (f"  retry-after={retry_after}" if retry_after else ""))
            reported_second += 1
        else:
            time.sleep(0.05)
    for worker in workers:
        worker.join()

    total_by_status = collections.Counter()
    for status_counts in results.status_by_second.values():
        total_by_status.update(status_counts)
    latencies = sorted(results.latencies_ms)
    print(f"[*] Totals: {dict(sorted(total_by_status.items()))}")
    if latencies:
        print(f"[*] Latency ms: p50={latencies[len(latencies) // 2]:.0f} "
              f"p99={latencies[int(len(latencies) * 0.99) - 1]:.0f} max={latencies[-1]:.0f}")
    for (status, body_shape), occurrences in results.refusal_bodies.most_common():
        print(f"\n[*] {occurrences} x status {status}, first one:\n{results.refusal_examples[(status, body_shape)]}")


if __name__ == "__main__":
    main()
