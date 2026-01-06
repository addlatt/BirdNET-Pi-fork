<?php
/**
 * Application Bootstrap
 * Sets up paths, constants, and includes core functionality.
 * This file must be included by all entry points.
 */

// Prevent direct access
if (!defined('APP_STARTED')) {
    die('Direct access not allowed');
}

// Application paths (absolute, no symlinks needed)
define('APP_ROOT', dirname(__DIR__));                    // src/web
define('PROJECT_ROOT', dirname(dirname(APP_ROOT)));      // BirdNET-Pi root
define('PUBLIC_PATH', APP_ROOT . '/public');
define('APP_PATH', APP_ROOT . '/app');
define('LIB_PATH', APP_PATH . '/lib');
define('PAGES_PATH', APP_PATH . '/pages');
define('VENDOR_PATH', APP_ROOT . '/vendor');

// Configuration file paths
define('CONFIG_FILE', '/etc/birdnet/birdnet.conf');
define('SCHEMA_FILE', PROJECT_ROOT . '/data/config_schema.json');

/**
 * Load configuration schema for defaults and type info
 */
function bootstrap_load_schema() {
    static $schema = null;
    if ($schema === null) {
        if (file_exists(SCHEMA_FILE)) {
            $content = file_get_contents(SCHEMA_FILE);
            $schema = json_decode($content, true) ?: [];
        } else {
            $schema = [];
        }
    }
    return $schema;
}

/**
 * Get schema default value for a key
 */
function bootstrap_get_schema_default($key) {
    $schema = bootstrap_load_schema();
    $props = $schema['properties'] ?? [];
    if (isset($props[$key]['default'])) {
        return $props[$key]['default'];
    }
    return null;
}

/**
 * Get schema type for a key
 */
function bootstrap_get_schema_type($key) {
    $schema = bootstrap_load_schema();
    $props = $schema['properties'] ?? [];
    if (isset($props[$key]['type'])) {
        return $props[$key]['type'];
    }
    return 'string';
}

/**
 * Coerce value to appropriate type based on schema
 */
function bootstrap_coerce_type($key, $value) {
    if ($value === '' || $value === null) {
        return bootstrap_get_schema_default($key);
    }

    $type = bootstrap_get_schema_type($key);

    switch ($type) {
        case 'number':
            return floatval($value);
        case 'integer':
            return intval(floatval($value));
        case 'boolean':
            return in_array(strtolower($value), ['true', '1', 'yes', 'on'], true);
        default:
            return $value;
    }
}

/**
 * Load configuration from birdnet.conf with environment override support
 *
 * Priority (highest to lowest):
 * 1. Environment variables (BIRDNET_<KEY>)
 * 2. Config file (/etc/birdnet/birdnet.conf)
 * 3. Schema defaults
 *
 * @param bool $force_reload Force reload from disk
 * @return array Configuration values with proper types
 */
function bootstrap_get_config($force_reload = false) {
    static $config = null;

    if ($config !== null && !$force_reload) {
        return $config;
    }

    $config = [];

    // 1. Load schema defaults first
    $schema = bootstrap_load_schema();
    foreach (($schema['properties'] ?? []) as $key => $prop) {
        if (isset($prop['default'])) {
            $config[$key] = $prop['default'];
        }
    }

    // 2. Load from config file
    if (file_exists(CONFIG_FILE)) {
        // Remove shell-style comments
        $source = preg_replace("~^#+.*$~m", "", file_get_contents(CONFIG_FILE));
        $file_config = parse_ini_string($source) ?: [];

        foreach ($file_config as $key => $value) {
            $config[$key] = $value;
        }
    }

    // 3. Coerce types based on schema
    foreach ($config as $key => $value) {
        $config[$key] = bootstrap_coerce_type($key, is_string($value) ? $value : strval($value));
    }

    // 4. Apply environment variable overrides (BIRDNET_<KEY>)
    foreach ($config as $key => $value) {
        $env_key = 'BIRDNET_' . $key;
        $env_value = getenv($env_key);
        if ($env_value !== false) {
            $config[$key] = bootstrap_coerce_type($key, $env_value);
        }
    }

    // Also check for any BIRDNET_* env vars that might set new keys
    foreach ($_SERVER as $env_key => $env_value) {
        if (strpos($env_key, 'BIRDNET_') === 0) {
            $key = substr($env_key, 8); // Remove 'BIRDNET_' prefix
            if (!isset($config[$key])) {
                $config[$key] = bootstrap_coerce_type($key, $env_value);
            }
        }
    }

    return $config;
}

/**
 * Get a single config value with environment override support
 *
 * @param string $key Configuration key
 * @param mixed $default Default value if not found
 * @return mixed Configuration value
 */
function get_config_value($key, $default = null) {
    $config = bootstrap_get_config();
    return $config[$key] ?? $default;
}

/**
 * Reload configuration from disk
 */
function reload_config() {
    return bootstrap_get_config(true);
}

// Load config for derived paths
$_bootstrap_config = bootstrap_get_config();

// User and home paths
define('BIRDNET_USER', $_bootstrap_config['BIRDNET_USER'] ?? 'pi');
define('HOME_PATH', '/home/' . BIRDNET_USER);

// Recording directories - use config values if set, otherwise compute from RECS_DIR
define('RECS_DIR', !empty($_bootstrap_config['RECS_DIR'])
    ? $_bootstrap_config['RECS_DIR']
    : HOME_PATH . '/BirdSongs');
define('EXTRACTED_PATH', !empty($_bootstrap_config['EXTRACTED'])
    ? $_bootstrap_config['EXTRACTED']
    : RECS_DIR . '/Extracted');
define('STREAMDATA_PATH', RECS_DIR . '/StreamData');

// Database path - NOW ABSOLUTE, not relative!
define('DB_PATH', PROJECT_ROOT . '/data/db/birds.db');

// Model and font paths
define('MODEL_PATH', PROJECT_ROOT . '/model');
define('FONT_PATH', PUBLIC_PATH . '/assets/fonts');

// Include core library
require_once LIB_PATH . '/common.php';
