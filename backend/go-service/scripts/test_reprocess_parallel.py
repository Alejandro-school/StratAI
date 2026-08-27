import importlib.util
import json
import sys
from pathlib import Path
from types import SimpleNamespace

SCRIPT_PATH = Path(__file__).with_name("reprocess_parallel.py")
SPEC = importlib.util.spec_from_file_location("reprocess_parallel", SCRIPT_PATH)
reprocess_parallel = importlib.util.module_from_spec(SPEC)
assert SPEC.loader is not None
sys.modules[SPEC.name] = reprocess_parallel
SPEC.loader.exec_module(reprocess_parallel)


def write_json(path: Path, payload: object) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(json.dumps(payload), encoding="utf-8")


def write_valid_export(
    exports_dir: Path,
    match_id: str,
    demo_checksum: str,
    played_at: str = "",
) -> Path:
    export_dir = exports_dir / f"match_{match_id}"
    quality = {
        "schema_version": reprocess_parallel.QUALITY_SCHEMA_VERSION,
        "status": "pass",
        "usable_for_training": True,
        "parse_completed": True,
        "expected_rounds": 2,
        "checks": [
            {"name": name, "status": "pass"}
            for name in sorted(reprocess_parallel.REQUIRED_QUALITY_CHECKS)
        ],
        **{name: 0 for name in reprocess_parallel.REQUIRED_ZERO_QUALITY_METRICS},
        "domains": [
            {
                "name": name,
                "status": "pass",
                "severity": "hard",
                "expected": "0 hard violations",
                "actual": "0 hard violations",
                "coverage": 1.0,
                "unavailable_count": 0,
                "inferred_count": 0,
                "warning_details": [],
                "hard_failure_details": [],
                "source_artifacts": ["canonical/manifest.json"],
                "schema_versions": ["stratai.canonical_manifest@3"],
            }
            for name in sorted(reprocess_parallel.REQUIRED_BLOCK7_DOMAINS)
        ],
    }
    write_json(
        export_dir / reprocess_parallel.CANONICAL_MATCH_PATH,
        {
            "schema_id": "stratai.match@1",
            "match_id": match_id,
            "played_at": played_at,
            "duration_ms": 1_800_000,
            "round_count": 2,
        },
    )
    write_json(
        export_dir / reprocess_parallel.CANONICAL_QUALITY_PATH,
        {
            "schema_id": "stratai.quality_report@1",
            "match_id": match_id,
            "report": quality,
        },
    )
    write_json(
        export_dir / reprocess_parallel.CANONICAL_PLAYER_STATS_PATH,
        {
            "schema_id": "stratai.player_stats@1",
            "match_id": match_id,
            "players": [],
        },
    )
    write_json(
        export_dir / "canonical/manifest.json",
        {
            "schema_id": "stratai.canonical_manifest@3",
            "export_format_version": reprocess_parallel.EXPORT_FORMAT_VERSION,
            "validator_version": reprocess_parallel.VALIDATOR_VERSION,
            "validation_status": "passed",
            "match_id": match_id,
            "artifacts": [],
        },
    )
    write_json(
        export_dir / reprocess_parallel.ROOT_MANIFEST_PATH,
        {
            "match_id": match_id,
            "checksum": demo_checksum,
            "parser_schema_version": reprocess_parallel.PARSER_SCHEMA_VERSION,
            "committed_at": "2026-08-13T00:00:00Z",
            "export_format_version": reprocess_parallel.EXPORT_FORMAT_VERSION,
            "artifacts": [],
        },
    )
    return export_dir


def allow_structural_validation(monkeypatch) -> None:
    monkeypatch.setattr(reprocess_parallel, "validate_match_export", lambda *_args: [])


def test_extract_match_id() -> None:
    assert reprocess_parallel.extract_match_id(Path("match_123.dem")) == "123"
    assert reprocess_parallel.extract_match_id(Path("demo_123.dem")) is None


def test_parse_arguments_accepts_limit_and_quality_only(monkeypatch) -> None:
    monkeypatch.setattr(
        sys, "argv", ["reprocess_parallel.py", "--limit", "2", "--quality-only"]
    )

    arguments = reprocess_parallel.parse_arguments()

    assert arguments.limit == 2
    assert arguments.quality_only is True


def test_parse_arguments_accepts_repeated_match_ids(monkeypatch) -> None:
    match_ids = [f"match-{index:02d}" for index in range(10)]
    arguments_list = [
        "reprocess_parallel.py",
        "--expected-count",
        "10",
    ]
    for match_id in match_ids:
        arguments_list.extend(("--match-id", match_id))
    monkeypatch.setattr(
        sys,
        "argv",
        arguments_list,
    )

    arguments = reprocess_parallel.parse_arguments()

    assert arguments.match_id == match_ids
    assert arguments.expected_count == 10


def test_select_demo_files_requires_every_requested_match() -> None:
    demos = [Path("match_3GJOG.dem"), Path("match_7D4W7.dem")]

    assert reprocess_parallel.select_demo_files(demos, ["7D4W7"]) == [
        Path("match_7D4W7.dem")
    ]

    try:
        reprocess_parallel.select_demo_files(demos, ["missing"])
    except ValueError as error:
        assert str(error) == "No existen demos para estos match IDs: missing"
    else:
        raise AssertionError("Debía rechazar un match ID sin demo")


def test_require_expected_count_accepts_ten_and_rejects_other_sizes() -> None:
    demos = [Path(f"match_{index:02d}.dem") for index in range(10)]

    assert reprocess_parallel.require_expected_count(demos, 10) == demos
    assert reprocess_parallel.require_expected_count(demos, None) == demos

    try:
        reprocess_parallel.require_expected_count(demos[:9], 10)
    except ValueError as error:
        assert str(error) == (
            "La selección contiene 9 demos; se esperaban exactamente 10."
        )
    else:
        raise AssertionError("Debía rechazar un lote con tamaño inesperado")


def test_get_demo_files_filters_on_canonical_quality_report(
    tmp_path, monkeypatch
) -> None:
    demos_dir = tmp_path / "demos"
    exports_dir = tmp_path / "exports"
    demos_dir.mkdir()
    for match_id in ("123", "456"):
        (demos_dir / f"match_{match_id}.dem").write_bytes(b"demo")
    write_json(
        exports_dir / "match_456" / reprocess_parallel.CANONICAL_QUALITY_PATH,
        {},
    )
    monkeypatch.setattr(reprocess_parallel, "DEMOS_DIR", demos_dir)
    monkeypatch.setattr(reprocess_parallel, "EXPORTS_DIR", exports_dir)

    assert [
        path.name for path in reprocess_parallel.get_demo_files(quality_only=True)
    ] == ["match_456.dem"]


def test_get_demo_checksum(tmp_path) -> None:
    demo_path = tmp_path / "match_123.dem"
    demo_path.write_bytes(b"demo")

    assert reprocess_parallel.get_demo_checksum(demo_path) == (
        "2a97516c354b68848cdbd8f54a226a0a55b21ed138e207ad6c5cbb9c00aa5aea"
    )


def test_verify_demo_sidecar_accepts_exact_sha_and_rejects_mismatch(tmp_path) -> None:
    demo_path = tmp_path / "match_123.dem"
    demo_path.write_bytes(b"demo")
    sidecar = demo_path.with_suffix(".dem.sha256")
    sidecar.write_text(reprocess_parallel.get_demo_checksum(demo_path), encoding="ascii")

    valid, detail = reprocess_parallel.verify_demo_sidecar(demo_path)
    assert valid is True
    assert detail == reprocess_parallel.get_demo_checksum(demo_path)

    sidecar.write_text("0" * 64, encoding="ascii")
    assert reprocess_parallel.verify_demo_sidecar(demo_path) == (
        False,
        "checksum de demo no coincide: match_123.dem",
    )


def test_create_verified_backup_copies_exact_tree_and_records_hashes(
    tmp_path, monkeypatch
) -> None:
    demos_dir = tmp_path / "demos"
    exports_dir = tmp_path / "exports"
    backup_dir = tmp_path / "backups" / "run"
    demos_dir.mkdir()
    demo_path = demos_dir / "match_123.dem"
    demo_path.write_bytes(b"demo")
    export_dir = exports_dir / "match_123"
    write_json(export_dir / "manifest.json", {"match_id": "123"})
    write_json(export_dir / "canonical/manifest.json", {"schema_id": "test"})
    monkeypatch.setattr(reprocess_parallel, "EXPORTS_DIR", exports_dir)

    result = reprocess_parallel.create_verified_backup([demo_path], backup_dir)

    manifest = json.loads((result / "backup_manifest.json").read_text(encoding="utf-8"))
    assert manifest["exports"]["123"] == reprocess_parallel.tree_hashes(export_dir)
    assert reprocess_parallel.tree_hashes(result / "match_123") == manifest["exports"]["123"]
    assert manifest["demo_checksums"]["123"] == reprocess_parallel.get_demo_checksum(
        demo_path
    )


def test_existing_match_uses_canonical_core(tmp_path, monkeypatch) -> None:
    exports_dir = tmp_path / "exports"
    write_json(
        exports_dir / "match_123" / reprocess_parallel.CANONICAL_MATCH_PATH,
        {"played_at": "2026-07-24T12:00:00Z", "duration_ms": 1_800_000},
    )
    monkeypatch.setattr(reprocess_parallel, "EXPORTS_DIR", exports_dir)

    assert (
        reprocess_parallel.get_existing_match("123")["played_at"]
        == "2026-07-24T12:00:00Z"
    )


def test_affected_steam_ids_are_read_from_canonical_stats(
    tmp_path, monkeypatch
) -> None:
    exports_dir = tmp_path / "exports"
    write_json(
        exports_dir / "match_123" / reprocess_parallel.CANONICAL_PLAYER_STATS_PATH,
        {
            "players": [
                {"player_id": "steam:76561198000000000", "metrics": {}},
                {"player_id": "invalid", "metrics": {}},
            ]
        },
    )
    monkeypatch.setattr(reprocess_parallel, "EXPORTS_DIR", exports_dir)

    assert reprocess_parallel.get_affected_steam_ids(["123"]) == {"76561198000000000"}


def test_process_demo_preserves_existing_canonical_match(tmp_path, monkeypatch) -> None:
    exports_dir = tmp_path / "exports"
    write_valid_export(
        exports_dir,
        "123",
        "a" * 64,
        played_at="2026-07-24T12:00:00Z",
    )
    demo_path = tmp_path / "match_123.dem"
    demo_path.write_bytes(b"demo")
    captured_request = {}

    class Response:
        def raise_for_status(self) -> None:
            return None

    def post(_url, data, headers, timeout):
        request_payload = json.loads(data)
        captured_request.update(request_payload)
        assert headers["X-Service-Version"] == "v1"
        assert len(headers["X-Service-Nonce"]) == 32
        write_valid_export(
            exports_dir,
            "123",
            request_payload["checksum"],
            played_at=request_payload["match_date"],
        )
        return Response()

    monkeypatch.setattr(reprocess_parallel, "EXPORTS_DIR", exports_dir)
    monkeypatch.setattr(reprocess_parallel.requests, "post", post)
    monkeypatch.setenv("INTERNAL_SERVICE_SECRET", "s" * 64)
    allow_structural_validation(monkeypatch)

    assert reprocess_parallel.process_demo(demo_path, timeout=10).status == "processed"
    assert captured_request["match_date"] == "2026-07-24T12:00:00Z"
    assert captured_request["match_duration"] == 1800
    assert captured_request["checksum"] == (
        "2a97516c354b68848cdbd8f54a226a0a55b21ed138e207ad6c5cbb9c00aa5aea"
    )
    assert captured_request["parser_schema_version"] == "v16"
    assert captured_request["source"] == "demo"
    assert captured_request["source_endpoint"] == "demo:123"
    assert captured_request["source_version"] == "v16"
    assert captured_request["force"] is True


def test_process_demo_does_not_retry_client_error(tmp_path, monkeypatch) -> None:
    exports_dir = tmp_path / "exports"
    demo_path = tmp_path / "match_123.dem"
    demo_path.write_bytes(b"demo")
    calls = []

    class Response:
        status_code = 403
        text = "forbidden"

        def raise_for_status(self) -> None:
            raise reprocess_parallel.requests.HTTPError("403", response=self)

    def post(*_args, **_kwargs):
        calls.append(1)
        return Response()

    monkeypatch.setattr(reprocess_parallel, "EXPORTS_DIR", exports_dir)
    monkeypatch.setattr(reprocess_parallel.requests, "post", post)
    monkeypatch.setenv("INTERNAL_SERVICE_SECRET", "s" * 64)

    result = reprocess_parallel.process_demo(demo_path, timeout=10, retries=2)

    assert result.status == "failed"
    assert len(calls) == 1


def test_process_demo_retries_transient_error_with_fresh_nonce(
    tmp_path, monkeypatch
) -> None:
    exports_dir = tmp_path / "exports"
    demo_path = tmp_path / "match_123.dem"
    demo_path.write_bytes(b"demo")
    nonces = []

    class Response:
        text = "temporary"

        def __init__(self, status_code: int):
            self.status_code = status_code

        def raise_for_status(self) -> None:
            if self.status_code >= 400:
                raise reprocess_parallel.requests.HTTPError(
                    str(self.status_code), response=self
                )

    def post(_url, data, headers, timeout):
        del timeout
        nonces.append(headers["X-Service-Nonce"])
        if len(nonces) == 1:
            return Response(503)
        payload = json.loads(data)
        write_valid_export(exports_dir, "123", payload["checksum"])
        return Response(200)

    monkeypatch.setattr(reprocess_parallel, "EXPORTS_DIR", exports_dir)
    monkeypatch.setattr(reprocess_parallel.requests, "post", post)
    monkeypatch.setattr(reprocess_parallel.time, "sleep", lambda *_args: None)
    monkeypatch.setenv("INTERNAL_SERVICE_SECRET", "s" * 64)
    allow_structural_validation(monkeypatch)

    result = reprocess_parallel.process_demo(demo_path, timeout=10, retries=2)

    assert result.status == "processed"
    assert len(nonces) == 2
    assert nonces[0] != nonces[1]


def test_validate_export_accepts_canonical_only_layout(tmp_path, monkeypatch) -> None:
    exports_dir = tmp_path / "exports"
    monkeypatch.setattr(reprocess_parallel, "EXPORTS_DIR", exports_dir)
    allow_structural_validation(monkeypatch)
    export_dir = write_valid_export(exports_dir, "123", "a" * 64)

    legacy_data_files = {
        "metadata.json",
        "quality.json",
        "players_summary.json",
        "combat.json",
        "economy.json",
        "grenades.json",
        "tracking.json",
        "replay.json",
    }
    assert not any((export_dir / name).exists() for name in legacy_data_files)
    assert reprocess_parallel.validate_export("123", "a" * 64) is None


def test_validate_export_rejects_unusable_canonical_quality(
    tmp_path, monkeypatch
) -> None:
    exports_dir = tmp_path / "exports"
    monkeypatch.setattr(reprocess_parallel, "EXPORTS_DIR", exports_dir)
    allow_structural_validation(monkeypatch)
    export_dir = write_valid_export(exports_dir, "123", "a" * 64)
    quality_path = export_dir / reprocess_parallel.CANONICAL_QUALITY_PATH
    quality = json.loads(quality_path.read_text(encoding="utf-8"))
    quality["report"]["status"] = "fail"
    quality["report"]["usable_for_training"] = False
    write_json(quality_path, quality)

    assert reprocess_parallel.validate_export("123", "a" * 64) == (
        "canonical/diagnostics/quality_report.json marca el export como no utilizable"
    )


def test_validate_export_requires_zero_utility_replay_projection_mismatches(
    tmp_path, monkeypatch
) -> None:
    exports_dir = tmp_path / "exports"
    monkeypatch.setattr(reprocess_parallel, "EXPORTS_DIR", exports_dir)
    allow_structural_validation(monkeypatch)
    export_dir = write_valid_export(exports_dir, "123", "a" * 64)
    quality_path = export_dir / reprocess_parallel.CANONICAL_QUALITY_PATH
    quality = json.loads(quality_path.read_text(encoding="utf-8"))
    quality["report"]["utility_replay_projection_mismatches"] = 1
    write_json(quality_path, quality)

    assert reprocess_parallel.validate_export("123", "a" * 64) == (
        "canonical/diagnostics/quality_report.json no supera metricas hard: "
        "utility_replay_projection_mismatches"
    )


def test_validate_export_requires_player_state_quality_checks(
    tmp_path, monkeypatch
) -> None:
    exports_dir = tmp_path / "exports"
    monkeypatch.setattr(reprocess_parallel, "EXPORTS_DIR", exports_dir)
    allow_structural_validation(monkeypatch)
    export_dir = write_valid_export(exports_dir, "123", "a" * 64)
    quality_path = export_dir / reprocess_parallel.CANONICAL_QUALITY_PATH
    quality = json.loads(quality_path.read_text(encoding="utf-8"))
    quality["report"]["checks"] = [
        check
        for check in quality["report"]["checks"]
        if check["name"] != "player_state_contract"
    ]
    write_json(quality_path, quality)

    assert reprocess_parallel.validate_export("123", "a" * 64) == (
        "canonical/diagnostics/quality_report.json no contiene checks requeridos: "
        "player_state_contract"
    )


def test_validate_export_requires_objective_quality_checks(
    tmp_path, monkeypatch
) -> None:
    exports_dir = tmp_path / "exports"
    monkeypatch.setattr(reprocess_parallel, "EXPORTS_DIR", exports_dir)
    allow_structural_validation(monkeypatch)
    export_dir = write_valid_export(exports_dir, "123", "a" * 64)
    quality_path = export_dir / reprocess_parallel.CANONICAL_QUALITY_PATH
    quality = json.loads(quality_path.read_text(encoding="utf-8"))
    quality["report"]["checks"] = [
        check
        for check in quality["report"]["checks"]
        if check["name"] != "objective_lifecycle"
    ]
    write_json(quality_path, quality)

    assert reprocess_parallel.validate_export("123", "a" * 64) == (
        "canonical/diagnostics/quality_report.json no contiene checks requeridos: "
        "objective_lifecycle"
    )


def test_validate_export_requires_utility_quality_checks(tmp_path, monkeypatch) -> None:
    exports_dir = tmp_path / "exports"
    monkeypatch.setattr(reprocess_parallel, "EXPORTS_DIR", exports_dir)
    allow_structural_validation(monkeypatch)
    export_dir = write_valid_export(exports_dir, "123", "a" * 64)
    quality_path = export_dir / reprocess_parallel.CANONICAL_QUALITY_PATH
    quality = json.loads(quality_path.read_text(encoding="utf-8"))
    quality["report"]["checks"] = [
        check
        for check in quality["report"]["checks"]
        if check["name"] != "utility_determinism"
    ]
    write_json(quality_path, quality)

    assert reprocess_parallel.validate_export("123", "a" * 64) == (
        "canonical/diagnostics/quality_report.json no contiene checks requeridos: "
        "utility_determinism"
    )


def test_validate_export_requires_v16_and_export_format_3_8(
    tmp_path, monkeypatch
) -> None:
    exports_dir = tmp_path / "exports"
    monkeypatch.setattr(reprocess_parallel, "EXPORTS_DIR", exports_dir)
    allow_structural_validation(monkeypatch)
    export_dir = write_valid_export(exports_dir, "123", "a" * 64)
    manifest_path = export_dir / reprocess_parallel.ROOT_MANIFEST_PATH
    manifest = json.loads(manifest_path.read_text(encoding="utf-8"))
    manifest["parser_schema_version"] = "v8"
    write_json(manifest_path, manifest)

    assert reprocess_parallel.validate_export("123", "a" * 64) == (
        "manifest.json no usa el schema esperado v16"
    )
    manifest["parser_schema_version"] = "v16"
    manifest["export_format_version"] = "2.0.0"
    write_json(manifest_path, manifest)

    assert reprocess_parallel.validate_export("123", "a" * 64) == (
        "manifest.json no usa el formato esperado 3.8.0"
    )


def test_validate_export_rejects_wrong_demo_checksum(tmp_path, monkeypatch) -> None:
    exports_dir = tmp_path / "exports"
    monkeypatch.setattr(reprocess_parallel, "EXPORTS_DIR", exports_dir)
    allow_structural_validation(monkeypatch)
    write_valid_export(exports_dir, "123", "a" * 64)

    assert reprocess_parallel.validate_export("123", "b" * 64) == (
        "manifest.json no corresponde al checksum de la demo"
    )


def test_validate_export_rejects_invalid_canonical_bundle(
    tmp_path, monkeypatch
) -> None:
    exports_dir = tmp_path / "exports"
    monkeypatch.setattr(reprocess_parallel, "EXPORTS_DIR", exports_dir)
    write_valid_export(exports_dir, "123", "a" * 64)
    monkeypatch.setattr(
        reprocess_parallel,
        "validate_match_export",
        lambda *_args: ["canonical/manifest.json: fichero requerido inexistente"],
    )

    assert reprocess_parallel.validate_export("123", "a" * 64) == (
        "bundle canónico inválido: canonical/manifest.json: fichero requerido inexistente"
    )


def test_duration_ms_to_seconds_handles_invalid_values() -> None:
    assert reprocess_parallel.duration_ms_to_seconds(1_821_968.695296) == 1821
    assert reprocess_parallel.duration_ms_to_seconds("2109340") == 2109
    assert reprocess_parallel.duration_ms_to_seconds(None) == 0


def test_health_rejects_non_object_json(monkeypatch) -> None:
    response = SimpleNamespace(json=lambda: [], ok=True, status_code=200)
    monkeypatch.setattr(reprocess_parallel.requests, "get", lambda *_args, **_kwargs: response)

    assert reprocess_parallel.get_go_service_health() == (
        False,
        "health incompatible: la respuesta JSON no es un objeto",
    )


def test_main_stops_batch_on_first_failure(tmp_path, monkeypatch) -> None:
    demos = [tmp_path / "match_1.dem", tmp_path / "match_2.dem"]
    for demo in demos:
        demo.write_bytes(b"demo")
    args = SimpleNamespace(
        workers=1,
        timeout=10,
        retries=2,
        limit=None,
        expected_count=2,
        quality_only=False,
        match_id=["1", "2"],
        skip_aggregate_rebuild=True,
        dry_run=False,
        backup_dir=tmp_path / "backup",
    )
    calls = []
    monkeypatch.setattr(reprocess_parallel, "parse_arguments", lambda: args)
    monkeypatch.setattr(reprocess_parallel, "get_demo_files", lambda *_args: demos)
    monkeypatch.setattr(reprocess_parallel, "verify_demo_sidecar", lambda *_args: (True, "a" * 64))
    monkeypatch.setattr(reprocess_parallel, "staging_is_clean", lambda: (True, "staging limpio"))
    monkeypatch.setattr(reprocess_parallel, "check_go_service", lambda: True)
    monkeypatch.setattr(reprocess_parallel, "create_verified_backup", lambda *_args: tmp_path / "backup")
    monkeypatch.setattr(
        reprocess_parallel,
        "process_demo",
        lambda path, *_args: calls.append(path.name)
        or reprocess_parallel.ProcessResult("failed", path.stem, "hard gate"),
    )
    monkeypatch.setenv("INTERNAL_SERVICE_SECRET", "s" * 64)

    assert reprocess_parallel.main() == 1
    assert calls == ["match_1.dem"]
