import importlib.util
import sys
from pathlib import Path

SCRIPT_DIR = Path(__file__).parent
sys.path.insert(0, str(SCRIPT_DIR))


def load_module(name: str, path: Path):
    spec = importlib.util.spec_from_file_location(name, path)
    module = importlib.util.module_from_spec(spec)
    assert spec.loader is not None
    sys.modules[name] = module
    spec.loader.exec_module(module)
    return module


fixtures = load_module(
    "block6_validator_fixtures", SCRIPT_DIR / "test_canonical_export_validator.py"
)
audit_module = load_module(
    "economy_stats_metadata_audit", SCRIPT_DIR / "economy_stats_metadata_audit.py"
)


def test_audit_reports_contracts_lineage_and_deterministic_tree(tmp_path: Path) -> None:
    match_dir = fixtures.write_valid_bundle(tmp_path)

    first = audit_module.audit(match_dir)
    second = audit_module.audit(match_dir)

    assert first["status"] == "pass"
    assert first["contracts"] == {
        "economy_round": "stratai.economy_round@1",
        "economy_player": "stratai.economy_player@1",
        "player_stats": "stratai.player_stats@1",
        "clutch_event": "stratai.clutch_event@1",
        "match_metadata": "stratai.match_metadata@1",
    }
    assert first["metadata"]["played_at"] is None
    assert first["stats"]["approximate_ratings"] == 2
    assert first["canonical_tree_sha256"] == second["canonical_tree_sha256"]
