<?php
/**
 * BirdNET-Pi Web Interface - Front Controller
 * All requests are routed through this file.
 */
define('APP_STARTED', true);

// Initialize application
require_once __DIR__ . '/../app/bootstrap.php';

// Start session if not already started
if (session_status() !== PHP_SESSION_ACTIVE) {
    session_start();
}

// Get request info
$requestUri = parse_url($_SERVER['REQUEST_URI'], PHP_URL_PATH);

// API routing (handle before normal pages)
if (strpos($requestUri, '/api/v1/') === 0) {
    require_once PAGES_PATH . '/api.php';
    exit;
}

// Sanitize input
$_GET = filter_input_array(INPUT_GET, FILTER_SANITIZE_FULL_SPECIAL_CHARS) ?: [];
$_POST = filter_input_array(INPUT_POST, FILTER_SANITIZE_FULL_SPECIAL_CHARS) ?: [];

// Get configuration
$config = get_config();
$site_name = get_sitename();
$color_scheme_css = get_color_scheme_css();
set_timezone();

// Determine if this is an iframe/view request or main page
$isViewRequest = isset($_GET['view']) || isset($_GET['filename']);

if ($isViewRequest) {
    // Route to views (inner content for iframe)
    require_once APP_PATH . '/router.php';
    exit;
}

// Main page - render the outer frame with iframe
?>
<!DOCTYPE html>
<html lang="en">
<head>
    <title><?php echo htmlspecialchars($site_name); ?></title>
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <link id="iconLink" rel="shortcut icon" sizes="85x85" href="/assets/images/bird.png" />
    <link rel="stylesheet" href="<?php echo $color_scheme_css . '?v=' . time(); ?>">
    <link rel="stylesheet" type="text/css" href="/assets/css/dialog-polyfill.css" />
</head>
<body>
<div class="banner">
    <div class="logo">
<?php if (isset($_GET['logo'])): ?>
        <a href="https://github.com/Nachtzuster/BirdNET-Pi.git" target="_blank"><img style="width:60;height:60;" src="/assets/images/bird.png"></a>
<?php else: ?>
        <a href="https://github.com/Nachtzuster/BirdNET-Pi.git" target="_blank"><img src="/assets/images/bird.png"></a>
<?php endif; ?>
    </div>

    <div class="stream">
<?php if (isset($_GET['stream'])): ?>
<?php ensure_authenticated('You cannot listen to the live audio stream'); ?>
        <audio controls autoplay><source src="/stream"></audio>
    </div>
    <h1><a href="/"><img class="topimage" src="/assets/images/bnp.png"></a></h1>
</div>
<div class="centered"><h3><?php echo htmlspecialchars($site_name); ?></h3></div>
<?php else: ?>
        <form action="" method="GET">
            <button type="submit" name="stream" value="play">Live Audio</button>
        </form>
    </div>
    <h1><a href="/"><img class="topimage" src="/assets/images/bnp.png"></a></h1>
</div>
<div class="centered"><h3><?php echo htmlspecialchars($site_name); ?></h3></div>
<?php endif; ?>

<?php if (isset($_GET['filename'])): ?>
<iframe src="?view=Recordings&filename=<?php echo htmlspecialchars($_GET['filename']); ?>"></iframe>
<?php else: ?>
<iframe src="?view=Overview"></iframe>
<?php endif; ?>

</body>
</html>
