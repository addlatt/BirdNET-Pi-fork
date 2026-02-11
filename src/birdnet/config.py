"""
BirdNET-Pi Configuration Management

Provides centralized configuration loading with:
- Schema-based validation and defaults
- Environment variable overrides (BIRDNET_<KEY>)
- Type coercion
- Path resolution
"""
import json
import logging
import os
from configparser import ConfigParser
from pathlib import Path
from typing import Any, Dict, List, Optional, Union

log = logging.getLogger(__name__)

# Configuration cache
_config_cache: Optional[Dict[str, Any]] = None
_schema_cache: Optional[Dict[str, Any]] = None

# Base paths computed relative to this module
# src/birdnet/config.py -> src/birdnet -> src -> project root
BASE_PATH = Path(__file__).parent.parent.parent.resolve()
CONFIG_FILE = Path('/etc/birdnet/birdnet.conf')
SCHEMA_FILE = BASE_PATH / 'data' / 'config_schema.json'


def get_base_path() -> Path:
    """Get the BirdNET-Pi installation root directory."""
    return BASE_PATH


def get_db_path() -> Path:
    """Get the path to the SQLite database."""
    return BASE_PATH / 'data' / 'db' / 'birds.db'


def get_model_path() -> Path:
    """Get the path to the model directory."""
    return BASE_PATH / 'model'


def get_font_dir() -> Path:
    """Get the path to the font directory."""
    return BASE_PATH / 'data' / 'fonts'


def get_analyzing_now() -> Path:
    """Get the path to the analyzing_now.txt lock file.

    Uses RECS_DIR from config instead of hardcoding ~/BirdSongs.
    """
    config = get_config()
    recs_dir = config.get('RECS_DIR', os.path.expanduser('~/BirdSongs'))
    return Path(recs_dir) / 'StreamData' / 'analyzing_now.txt'


def get_install_dir() -> Path:
    """Get the BirdNET-Pi installation directory.

    Uses INSTALL_DIR from config, falling back to ~/BirdNET-Pi.
    """
    config = get_config()
    install_dir = config.get('INSTALL_DIR', '')
    if install_dir:
        return Path(install_dir)
    return Path(os.path.expanduser('~/BirdNET-Pi'))


def get_species_list_path(list_type: str) -> Path:
    """Get path to a species list file.

    Args:
        list_type: One of 'include', 'exclude', or 'whitelist'

    Returns:
        Path to the species list file
    """
    install_dir = get_install_dir()
    return install_dir / f'{list_type}_species_list.txt'


def get_include_species_list() -> Path:
    """Get path to the include species list file."""
    return get_species_list_path('include')


def get_exclude_species_list() -> Path:
    """Get path to the exclude species list file."""
    return get_species_list_path('exclude')


def get_whitelist_species_list() -> Path:
    """Get path to the whitelist species list file."""
    return get_species_list_path('whitelist')


def get_birddb_path() -> Path:
    """Get path to the BirdDB.txt log file."""
    return get_install_dir() / 'BirdDB.txt'


def get_apprise_config_path() -> Path:
    """Get path to the Apprise configuration file."""
    return get_install_dir() / 'apprise.txt'


def get_apprise_body_path() -> Path:
    """Get path to the Apprise notification body template."""
    return get_install_dir() / 'body.txt'


def _load_schema() -> Dict[str, Any]:
    """Load configuration schema for validation and defaults."""
    global _schema_cache
    if _schema_cache is None:
        if SCHEMA_FILE.exists():
            with open(SCHEMA_FILE) as f:
                _schema_cache = json.load(f)
        else:
            log.warning(f"Schema file not found: {SCHEMA_FILE}")
            _schema_cache = {}
    return _schema_cache


def _get_schema_type(key: str, schema: Dict) -> str:
    """Get the expected type for a config key from schema."""
    props = schema.get('properties', {})
    if key in props:
        return props[key].get('type', 'string')
    return 'string'


def _get_schema_default(key: str, schema: Dict) -> Any:
    """Get the default value for a config key from schema."""
    props = schema.get('properties', {})
    if key in props:
        return props[key].get('default')
    return None


def _coerce_type(key: str, value: str, schema: Dict) -> Any:
    """Coerce string value to appropriate type based on schema."""
    if value == '' or value is None:
        return _get_schema_default(key, schema)

    prop_type = _get_schema_type(key, schema)

    try:
        if prop_type == 'number':
            return float(value)
        elif prop_type == 'integer':
            return int(float(value))  # Handle "1.0" -> 1
        elif prop_type == 'boolean':
            return str(value).lower() in ('true', '1', 'yes', 'on')
        return value
    except (ValueError, TypeError):
        log.warning(f"Could not coerce {key}={value!r} to {prop_type}")
        return value


def _apply_env_overrides(config: Dict[str, Any], schema: Dict) -> Dict[str, Any]:
    """Apply BIRDNET_* environment variable overrides."""
    result = dict(config)

    # Check for overrides of existing keys
    for key in list(config.keys()):
        env_key = f'BIRDNET_{key}'
        env_value = os.environ.get(env_key)
        if env_value is not None:
            result[key] = _coerce_type(key, env_value, schema)
            log.debug(f"Config override: {key} from env {env_key}")

    # Check for any BIRDNET_* env vars that might set new keys
    for env_key, env_value in os.environ.items():
        if env_key.startswith('BIRDNET_'):
            key = env_key[8:]  # Remove 'BIRDNET_' prefix
            if key not in result:
                result[key] = _coerce_type(key, env_value, schema)
                log.debug(f"Config added: {key} from env {env_key}")

    return result


def _validate_config(config: Dict[str, Any], schema: Dict) -> List[str]:
    """Validate config against schema. Returns list of errors."""
    errors = []
    props = schema.get('properties', {})
    required = schema.get('required', [])

    # Check required fields
    for req in required:
        if req not in config or config[req] in (None, ''):
            errors.append(f"Missing required config: {req}")

    # Check type constraints
    for key, value in config.items():
        if key not in props:
            continue

        prop = props[key]
        prop_type = prop.get('type', 'string')

        # Skip validation for empty optional fields
        if value in (None, '') and key not in required:
            continue

        # Check numeric constraints
        if prop_type in ('number', 'integer') and isinstance(value, (int, float)):
            if 'minimum' in prop and value < prop['minimum']:
                errors.append(f"{key}={value} must be >= {prop['minimum']}")
            if 'maximum' in prop and value > prop['maximum']:
                errors.append(f"{key}={value} must be <= {prop['maximum']}")

        # Check enum constraints
        if 'enum' in prop and value not in prop['enum']:
            errors.append(f"{key}={value!r} must be one of {prop['enum']}")

    return errors


class QuotedINIParser(ConfigParser):
    """ConfigParser that strips surrounding quotes from values."""

    def get(self, section: str, option: str, *, raw: bool = False,
            vars: Optional[Dict] = None, fallback: Any = None) -> Any:
        value = super().get(section, option, raw=raw, vars=vars, fallback=fallback)
        if raw or value is None:
            return value
        # Strip surrounding quotes
        if isinstance(value, str):
            return value.strip('"\'')
        return value


def get_config(force_reload: bool = False) -> Dict[str, Any]:
    """
    Load configuration with schema defaults, file values, and environment overrides.

    Priority (highest to lowest):
    1. Environment variables (BIRDNET_<KEY>)
    2. Config file (/etc/birdnet/birdnet.conf)
    3. Schema defaults

    Args:
        force_reload: If True, bypass cache and reload from disk

    Returns:
        Dictionary of configuration values with proper types
    """
    global _config_cache

    if _config_cache is not None and not force_reload:
        return _config_cache

    schema = _load_schema()
    config: Dict[str, Any] = {}

    # 1. Apply schema defaults
    for key, prop in schema.get('properties', {}).items():
        if 'default' in prop:
            config[key] = prop['default']

    # 2. Load from config file
    if CONFIG_FILE.exists():
        parser = QuotedINIParser(interpolation=None)
        parser.optionxform = lambda x: x  # Preserve case

        try:
            with open(CONFIG_FILE) as f:
                # Add fake section header for INI parsing
                content = f.read()
                # Remove comment lines (shell comments)
                lines = [line for line in content.splitlines()
                        if not line.strip().startswith('#')]
                parser.read_string('[birdnet]\n' + '\n'.join(lines))

            for key, value in parser['birdnet'].items():
                config[key] = value
        except Exception as e:
            log.error(f"Failed to parse config file: {e}")
    else:
        log.warning(f"Config file not found: {CONFIG_FILE}")

    # 3. Coerce types based on schema
    for key in list(config.keys()):
        config[key] = _coerce_type(key, str(config[key]) if config[key] is not None else '', schema)

    # 4. Apply environment overrides
    config = _apply_env_overrides(config, schema)

    # 5. Validate and log warnings
    errors = _validate_config(config, schema)
    for error in errors:
        log.warning(f"Config validation: {error}")

    _config_cache = config
    return config


def get_settings(settings_path: str = '/etc/birdnet/birdnet.conf',
                 force_reload: bool = False) -> Dict[str, Any]:
    """
    Backwards-compatible wrapper for get_config().

    Note: settings_path parameter is ignored; always uses CONFIG_FILE.
    """
    return get_config(force_reload)


def reload_config() -> Dict[str, Any]:
    """Force reload configuration from disk."""
    return get_config(force_reload=True)


def get_schema() -> Dict[str, Any]:
    """Get the configuration schema."""
    return _load_schema()


def validate_value(key: str, value: Any) -> Union[bool, str]:
    """
    Validate a single config value against schema.

    Args:
        key: Configuration key name
        value: Value to validate

    Returns:
        True if valid, or error message string if invalid
    """
    schema = _load_schema()
    props = schema.get('properties', {})

    if key not in props:
        return True  # Unknown key, allow

    prop = props[key]
    prop_type = prop.get('type', 'string')

    # Type checking
    if prop_type == 'number':
        try:
            num_value = float(value)
        except (ValueError, TypeError):
            return f"{key} must be a number"

        if 'minimum' in prop and num_value < prop['minimum']:
            return f"{key} must be >= {prop['minimum']}"
        if 'maximum' in prop and num_value > prop['maximum']:
            return f"{key} must be <= {prop['maximum']}"

    elif prop_type == 'integer':
        try:
            int_value = int(float(value))
        except (ValueError, TypeError):
            return f"{key} must be an integer"

        if 'minimum' in prop and int_value < prop['minimum']:
            return f"{key} must be >= {prop['minimum']}"
        if 'maximum' in prop and int_value > prop['maximum']:
            return f"{key} must be <= {prop['maximum']}"

    # Enum checking
    if 'enum' in prop:
        # Handle both string and numeric enums
        if value not in prop['enum'] and str(value) not in map(str, prop['enum']):
            return f"{key} must be one of {prop['enum']}"

    return True


# Module-level constants for backwards compatibility
# These are evaluated at import time
DB_PATH = str(get_db_path())
MODEL_PATH = str(get_model_path())
FONT_DIR = str(get_font_dir())
