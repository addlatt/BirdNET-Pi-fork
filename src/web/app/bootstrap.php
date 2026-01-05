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

// Configuration file path
define('CONFIG_FILE', '/etc/birdnet/birdnet.conf');

/**
 * Load configuration from birdnet.conf
 * Used early in bootstrap before common.php is loaded
 */
function bootstrap_get_config() {
    static $config = null;
    if ($config === null) {
        if (!file_exists(CONFIG_FILE)) {
            // Fallback for development/testing
            $config = [];
        } else {
            $source = preg_replace("~^#+.*$~m", "", file_get_contents(CONFIG_FILE));
            $config = parse_ini_string($source) ?: [];
        }
    }
    return $config;
}

// Load config for derived paths
$_bootstrap_config = bootstrap_get_config();

// User and home paths
define('BIRDNET_USER', $_bootstrap_config['BIRDNET_USER'] ?? 'pi');
define('HOME_PATH', '/home/' . BIRDNET_USER);

// Recording directories
define('RECS_DIR', $_bootstrap_config['RECS_DIR'] ?? HOME_PATH . '/BirdSongs');
define('EXTRACTED_PATH', RECS_DIR . '/Extracted');
define('STREAMDATA_PATH', RECS_DIR . '/StreamData');

// Database path - NOW ABSOLUTE, not relative!
define('DB_PATH', PROJECT_ROOT . '/data/db/birds.db');

// Model and font paths
define('MODEL_PATH', PROJECT_ROOT . '/model');
define('FONT_PATH', PUBLIC_PATH . '/assets/fonts');

// Include core library
require_once LIB_PATH . '/common.php';
