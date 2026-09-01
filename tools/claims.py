#!/usr/bin/env python3
"""Validate the claims ledger and generate its capability-status table."""

from __future__ import annotations

import argparse
import datetime
import json
import re
import sys
from pathlib import Path
from typing import Any


REPOSITORY_ROOT = Path(__file__).resolve().parents[1]
DEFAULT_CLAIMS_PATH = REPOSITORY_ROOT / "state" / "CLAIMS.json"
DEFAULT_DOCUMENT_PATH = REPOSITORY_ROOT / "docs" / "CAPABILITY_STATUS.md"
GENERATED_TABLE_BEGIN = (
    "<!-- BEGIN GENERATED CLAIMS TABLE: "
    "do not edit; run python3 tools/claims.py generate -->"
)
GENERATED_TABLE_END = "<!-- END GENERATED CLAIMS TABLE -->"
LEGACY_TABLE_HEADER = "| Capability | Status | Notes |"
ALLOWED_STATUSES = {
    "designed",
    "implemented",
    "evidenced",
    "evidenced_failing",
    "verified",
    "rejected",
}
REQUIRED_FIELDS = {
    "id",
    "statement",
    "status",
    "version_introduced",
    "last_checked",
}
ALLOWED_FIELDS = REQUIRED_FIELDS | {"evidence", "notes"}
EVIDENCE_REQUIRED_STATUSES = {
    "implemented",
    "evidenced",
    "evidenced_failing",
    "verified",
}
TEST_REFERENCE_PATTERN = re.compile(
    r"^test:(?P<package_path>[^:]+):(?P<test_name>Test[A-Za-z0-9_]+)$"
)
CLAIM_ID_PATTERN = re.compile(r"^[A-Z0-9][A-Z0-9._-]*$")


def load_claims(claims_path: Path) -> list[dict[str, Any]]:
    try:
        claims_value = json.loads(claims_path.read_text(encoding="utf-8"))
    except (OSError, json.JSONDecodeError) as error:
        raise ValueError(f"cannot load {claims_path}: {error}") from error
    if not isinstance(claims_value, list):
        raise ValueError("claims ledger must be a JSON array")
    return claims_value


def validate_test_reference(reference: str) -> list[str]:
    reference_match = TEST_REFERENCE_PATTERN.fullmatch(reference)
    if reference_match is None:
        return [f"invalid test evidence reference: {reference}"]

    package_path = (REPOSITORY_ROOT / reference_match["package_path"]).resolve()
    try:
        package_path.relative_to(REPOSITORY_ROOT)
    except ValueError:
        return [f"test evidence escapes repository root: {reference}"]
    if not package_path.is_dir():
        return [f"test evidence package does not exist: {reference}"]

    test_declaration = re.compile(
        rf"^func\s+{re.escape(reference_match['test_name'])}\s*\(",
        re.MULTILINE,
    )
    for test_path in sorted(package_path.glob("*_test.go")):
        if test_declaration.search(test_path.read_text(encoding="utf-8")):
            return []
    return [f"test evidence function does not exist: {reference}"]


def validate_claims(claims: list[dict[str, Any]]) -> list[str]:
    errors: list[str] = []
    if not claims:
        return ["claims ledger must not be empty"]

    observed_ids: set[str] = set()
    for claim_index, claim in enumerate(claims):
        claim_location = f"claim[{claim_index}]"
        if not isinstance(claim, dict):
            errors.append(f"{claim_location} must be an object")
            continue

        missing_fields = sorted(REQUIRED_FIELDS - claim.keys())
        extra_fields = sorted(claim.keys() - ALLOWED_FIELDS)
        if missing_fields:
            errors.append(f"{claim_location} missing fields: {', '.join(missing_fields)}")
        if extra_fields:
            errors.append(f"{claim_location} has unknown fields: {', '.join(extra_fields)}")

        claim_id = claim.get("id")
        if not isinstance(claim_id, str) or CLAIM_ID_PATTERN.fullmatch(claim_id) is None:
            errors.append(f"{claim_location}.id must be a stable uppercase identifier")
            display_id = claim_location
        else:
            display_id = claim_id
            if claim_id in observed_ids:
                errors.append(f"duplicate claim id: {claim_id}")
            observed_ids.add(claim_id)

        for field_name in ("statement", "version_introduced"):
            field_value = claim.get(field_name)
            if not isinstance(field_value, str) or not field_value.strip():
                errors.append(f"{display_id}.{field_name} must be a non-empty string")

        claim_status = claim.get("status")
        if claim_status not in ALLOWED_STATUSES:
            errors.append(f"{display_id}.status is not allowed: {claim_status!r}")

        last_checked = claim.get("last_checked")
        if not isinstance(last_checked, str):
            errors.append(f"{display_id}.last_checked must be an ISO date")
        else:
            try:
                datetime.date.fromisoformat(last_checked)
            except ValueError:
                errors.append(f"{display_id}.last_checked must be an ISO date")

        evidence = claim.get("evidence", [])
        if not isinstance(evidence, list) or not all(
            isinstance(reference, str) and reference for reference in evidence
        ):
            errors.append(f"{display_id}.evidence must be an array of non-empty strings")
            evidence = []
        if claim_status in EVIDENCE_REQUIRED_STATUSES and not evidence:
            errors.append(f"{display_id} requires evidence for status {claim_status}")
        if claim_status == "implemented" and not any(
            reference.startswith("test:") for reference in evidence
        ):
            errors.append(f"{display_id} requires test evidence for implemented status")
        if claim_status in {"evidenced", "evidenced_failing"} and not any(
            reference.startswith("run:") for reference in evidence
        ):
            errors.append(f"{display_id} requires run evidence for status {claim_status}")
        if claim_status == "verified" and not any(
            reference.startswith(("hash:", "invariant:", "bytes:"))
            for reference in evidence
        ):
            errors.append(f"{display_id} requires deterministic verification evidence")

        for evidence_reference in evidence:
            if evidence_reference.startswith("test:"):
                errors.extend(validate_test_reference(evidence_reference))
            elif ":" not in evidence_reference:
                errors.append(f"invalid evidence reference: {evidence_reference}")

    return errors


def markdown_cell(value: Any) -> str:
    if isinstance(value, list):
        rendered_value = "<br>".join(f"`{item}`" for item in value) or "—"
    else:
        rendered_value = str(value or "—")
    return rendered_value.replace("|", "\\|").replace("\n", " ")


def render_table(claims: list[dict[str, Any]]) -> str:
    table_lines = [
        GENERATED_TABLE_BEGIN,
        "| Claim | Statement | Status | Evidence | Version introduced | Last checked | Notes |",
        "|---|---|---|---|---|---|---|",
    ]
    for claim in sorted(claims, key=lambda item: item["id"]):
        row_values = (
            f"`{claim['id']}`",
            claim["statement"],
            f"`{claim['status']}`",
            claim.get("evidence", []),
            f"`{claim['version_introduced']}`",
            claim["last_checked"],
            claim.get("notes", ""),
        )
        table_lines.append("| " + " | ".join(markdown_cell(value) for value in row_values) + " |")
    table_lines.append(GENERATED_TABLE_END)
    return "\n".join(table_lines)


def generated_document(document_text: str, table_text: str) -> str:
    if GENERATED_TABLE_BEGIN in document_text and GENERATED_TABLE_END in document_text:
        prefix, remaining_text = document_text.split(GENERATED_TABLE_BEGIN, 1)
        _, suffix = remaining_text.split(GENERATED_TABLE_END, 1)
        return prefix + table_text + suffix

    legacy_start = document_text.find(LEGACY_TABLE_HEADER)
    if legacy_start < 0:
        raise ValueError("capability document has no generated markers or legacy table")
    next_section = document_text.find("\n\n## ", legacy_start)
    if next_section < 0:
        raise ValueError("cannot find the section after the legacy capability table")
    return document_text[:legacy_start] + table_text + document_text[next_section:]


def emit_result(command: str, claims: list[dict[str, Any]], **details: Any) -> None:
    status_counts = {
        status: sum(1 for claim in claims if claim["status"] == status)
        for status in sorted(ALLOWED_STATUSES)
        if any(claim["status"] == status for claim in claims)
    }
    result = {
        "ok": True,
        "command": command,
        "claims": len(claims),
        "status_counts": status_counts,
        **details,
    }
    print(json.dumps(result, sort_keys=True))


def fail(errors: list[str]) -> None:
    print(json.dumps({"ok": False, "errors": errors}, sort_keys=True), file=sys.stderr)
    raise SystemExit(1)


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("command", choices=("validate", "generate", "check"))
    parser.add_argument("--claims", type=Path, default=DEFAULT_CLAIMS_PATH)
    parser.add_argument("--document", type=Path, default=DEFAULT_DOCUMENT_PATH)
    arguments = parser.parse_args()

    try:
        claims = load_claims(arguments.claims.resolve())
    except ValueError as error:
        fail([str(error)])
    validation_errors = validate_claims(claims)
    if validation_errors:
        fail(validation_errors)
    if arguments.command == "validate":
        emit_result("validate", claims)
        return

    table_text = render_table(claims)
    try:
        current_document = arguments.document.read_text(encoding="utf-8")
        expected_document = generated_document(current_document, table_text)
    except (OSError, ValueError) as error:
        fail([str(error)])

    if arguments.command == "generate":
        arguments.document.write_text(expected_document, encoding="utf-8")
        emit_result("generate", claims, document=str(arguments.document))
        return
    if current_document != expected_document:
        fail(
            [
                f"generated claims table is stale: {arguments.document}; "
                "run python3 tools/claims.py generate"
            ]
        )
    emit_result("check", claims, document=str(arguments.document))


if __name__ == "__main__":
    main()
