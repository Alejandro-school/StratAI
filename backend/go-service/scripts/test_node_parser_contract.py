"""Node-produced requests must satisfy the actual Python root-catalog contract."""
import json
import subprocess
from pathlib import Path

import canonical_export_validator as validator


def test_node_new_default_and_recovered_requests_match_root_validator(tmp_path):
    helper = Path(__file__).resolve().parents[2] / "node-service/test/helpers/emitAnalysisRequest.js"
    requests = json.loads(subprocess.check_output(["node", str(helper)], text=True))
    assert len(requests) == 3
    for index, request in enumerate(requests):
        match_dir = tmp_path / str(index)
        match_dir.mkdir()
        manifest = {
            "match_id": request["match_id"], "checksum": request["checksum"],
            "parser_schema_version": request["parser_schema_version"],
            "export_format_version": validator.EXPORT_FORMAT_VERSION,
            "validator_version": validator.VALIDATOR_VERSION,
            "validation_status": "passed", "committed_at": "contract-test", "artifacts": [],
        }
        (match_dir / "manifest.json").write_text(json.dumps(manifest), encoding="utf-8")
        errors = []
        validator.validate_root_catalog(match_dir, request["match_id"], errors, request["checksum"])
        assert errors == []
        # This is a version-contract regression, not certification of an empty bundle.
        manifest["parser_schema_version"] = "v11"
        (match_dir / "manifest.json").write_text(json.dumps(manifest), encoding="utf-8")
        errors = []
        validator.validate_root_catalog(match_dir, request["match_id"], errors, request["checksum"])
        assert errors == [f"manifest.json: parser_schema_version debe ser {validator.PARSER_SCHEMA_VERSION}"]
