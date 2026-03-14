#!/usr/bin/env python3

"""Send a direct POST request to the local almacenes endpoint for manual testing."""

from __future__ import annotations

import argparse
import json
import sys
import urllib.error
import urllib.request


DEFAULT_ENDPOINT_URL = "http://localhost:3589/api/almacenes?empresa-id=1"
DEFAULT_AUTHORIZATION_TOKEN = "pQEBAgUDGhcNDAAEG6V2bbdWIB3YBWh1c3VhcmlvMg~~"
DEFAULT_PAYLOAD = {
    "EmpresaID": 1,
    "ID": 1,
    "SedeID": 1,
    "Nombre": "Tienda 1",
    "Descripcion": "",
    "Layout": None,
    "ss": 1,
    "upd": 386600975,
    "Created": 386600975,
    "CreatedBy": 1,
}


def build_argument_parser() -> argparse.ArgumentParser:
    """Expose the minimum knobs needed to retarget the request without editing the file."""
    argument_parser = argparse.ArgumentParser(
        description="Send a POST /api/almacenes request to a Genix backend."
    )
    argument_parser.add_argument(
        "--url",
        default=DEFAULT_ENDPOINT_URL,
        help="Target request URL.",
    )
    argument_parser.add_argument(
        "--token",
        default=DEFAULT_AUTHORIZATION_TOKEN,
        help="Authorization token value to send in the Authorization header.",
    )
    argument_parser.add_argument(
        "--use-bearer",
        action="store_true",
        help="Prefix the Authorization header with 'Bearer '.",
    )
    argument_parser.add_argument(
        "--timeout-seconds",
        type=float,
        default=15.0,
        help="HTTP timeout in seconds.",
    )
    return argument_parser


def send_almacenes_request(
    endpoint_url: str,
    authorization_token: str,
    use_bearer_prefix: bool,
    timeout_seconds: float,
) -> int:
    """Send the JSON payload and print both the request/response details for debugging."""
    serialized_payload = json.dumps(DEFAULT_PAYLOAD).encode("utf-8")
    authorization_header_value = (
        f"Bearer {authorization_token}" if use_bearer_prefix else authorization_token
    )

    request_headers = {
        # Ask the server to avoid transport compression so the script can print the body directly.
        "Accept": "application/json",
        "Accept-Encoding": "identity",
        "Authorization": authorization_header_value,
        "Content-Type": "application/json",
    }

    request = urllib.request.Request(
        endpoint_url,
        data=serialized_payload,
        headers=request_headers,
        method="POST",
    )

    print("Sending POST request...")
    print(f"URL: {endpoint_url}")
    print(f"Authorization header length: {len(authorization_header_value)}")
    print(f"Payload: {json.dumps(DEFAULT_PAYLOAD, ensure_ascii=True)}")

    try:
        with urllib.request.urlopen(request, timeout=timeout_seconds) as response:
            response_body_bytes = response.read()
            response_body_text = response_body_bytes.decode("utf-8", errors="replace")

            print(f"Status: {response.status}")
            print("Response headers:")
            for header_name, header_value in response.headers.items():
                print(f"  {header_name}: {header_value}")
            print("Response body:")
            print(response_body_text)
            return 0

    except urllib.error.HTTPError as http_error:
        error_body_text = http_error.read().decode("utf-8", errors="replace")
        print(f"HTTP error status: {http_error.code}", file=sys.stderr)
        print("HTTP error body:", file=sys.stderr)
        print(error_body_text, file=sys.stderr)
        return 1

    except urllib.error.URLError as url_error:
        print(f"Request failed: {url_error}", file=sys.stderr)
        return 1


def main() -> int:
    """Parse arguments and execute the single request flow."""
    parsed_arguments = build_argument_parser().parse_args()
    return send_almacenes_request(
        endpoint_url=parsed_arguments.url,
        authorization_token=parsed_arguments.token,
        use_bearer_prefix=parsed_arguments.use_bearer,
        timeout_seconds=parsed_arguments.timeout_seconds,
    )


if __name__ == "__main__":
    raise SystemExit(main())
