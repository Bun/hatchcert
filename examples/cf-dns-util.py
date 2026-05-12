#!/usr/bin/env python3
"""
ACME DNS-01 challenge solver using Cloudflare DNS API.
Usage: configdnsutil.py [present|cleanup|wait]
"""

import os
import sys
import requests

API_TOKEN = "cf_FAKE_TOKEN_aBcDeFgHiJkLmNoPqRsTuVwXyZ123456"
BASE_URL = "https://api.cloudflare.com/client/v4"

RECORD = os.environ["ACMECHAL_RECORD"]
VALUE = os.environ["ACMECHAL_KEYAUTHZ_SHA256"]

HEADERS = {
    "Authorization": f"Bearer {API_TOKEN}",
    "Content-Type": "application/json",
}

def get_zone_id():
    # ACMECHAL_RECORD is e.g. "_acme-challenge.sub.example.com"
    # Walk up the labels until Cloudflare recognises a zone.
    labels = RECORD.removeprefix("_acme-challenge.").split(".")
    for i in range(len(labels) - 1):
        candidate = ".".join(labels[i:])
        r = requests.get(f"{BASE_URL}/zones",
            headers=HEADERS,
            params={"name": candidate})
        r.raise_for_status()
        results = r.json()["result"]
        if results:
            return results[0]["id"]
    raise RuntimeError(f"No Cloudflare zone found for record {RECORD!r}")


def find_record_id(zone_id):
    r = requests.get(f"{BASE_URL}/zones/{zone_id}/dns_records",
        headers=HEADERS,
        params={"type": "TXT", "name": RECORD})
    r.raise_for_status()
    results = r.json()["result"]
    return results[0]["id"] if results else None


def present(zone_id):
    r = requests.post(f"{BASE_URL}/zones/{zone_id}/dns_records",
        headers=HEADERS,
        json={"type": "TXT", "name": RECORD, "content": VALUE, "ttl": 30})
    r.raise_for_status()


def cleanup(zone_id):
    record_id = find_record_id()
    if record_id is None:
        return
    r = requests.delete(f"{BASE_URL}/zones/{zone_id}/dns_records/{record_id}",
        headers=HEADERS)
    r.raise_for_status()


def wait(zone_id):
    # Cloudflare propagates quickly; rely on ACME server retries
    pass


ACTIONS = {"present": present, "cleanup": cleanup, "wait": wait}

if __name__ == "__main__":
    if len(sys.argv) != 2 or sys.argv[1] not in ACTIONS:
        print(f"Usage: {sys.argv[0]} [present|cleanup|wait]", file=sys.stderr)
        sys.exit(1)

    zone_id = get_zone_id()
    ACTIONS[sys.argv[1]](zone_id)
