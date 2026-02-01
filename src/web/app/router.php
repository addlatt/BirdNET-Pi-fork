<?php
/**
 * View Router
 * Routes requests to appropriate page controllers.
 * This file is included from the front controller (public/index.php).
 */

// Ensure we're called from front controller
if (!defined('APP_STARTED')) {
    die('Direct access not allowed');
}

$user = get_user();
$home = get_home();
$config = get_config();
$color_scheme_css = get_color_scheme_css();
set_timezone();

$restore = "cat $home/BirdSongs/restore.log";

// Check for updates (cached for 24 hours)
if (is_authenticated() && (!isset($_SESSION['behind']) || !isset($_SESSION['behind_time']) || time() > $_SESSION['behind_time'] + 86400)) {
    shell_exec("sudo -u" . $user . " git -C " . PROJECT_ROOT . " fetch > /dev/null 2>/dev/null &");
    $str = trim(shell_exec("sudo -u" . $user . " git -C " . PROJECT_ROOT . " status"));
    if (preg_match("/behind '.*?' by (\d+) commit(s?)\b/", $str, $matches)) {
        $num_commits_behind = $matches[1];
    }
    if (preg_match('/\b(\d+)\b and \b(\d+)\b different commits each/', $str, $matches)) {
        $num1 = (int) $matches[1];
        $num2 = (int) $matches[2];
        $num_commits_behind = $num1 + $num2;
    }
    if (stripos($str, "Your branch is up to date") !== false) {
        $num_commits_behind = '0';
    }
    $_SESSION['behind'] = $num_commits_behind ?? null;
    $_SESSION['behind_time'] = time();
}

$updatediv = "";
if (isset($_SESSION['behind']) && intval($_SESSION['behind']) >= 50 && ($config['SILENCE_UPDATE_INDICATOR'] ?? 0) != 1) {
    $updatediv = ' <div class="updatenumber">' . $_SESSION["behind"] . '</div>';
}

// Configuration warnings
$warnings = [];
if (($config["LATITUDE"] ?? "0.000") == "0.000" && ($config["LONGITUDE"] ?? "0.000") == "0.000") {
    $warnings[] = "WARNING: Your latitude and longitude are not set properly. Please do so now in Tools -> Settings.";
} elseif (($config["LATITUDE"] ?? "0.000") == "0.000") {
    $warnings[] = "WARNING: Your latitude is not set properly. Please do so now in Tools -> Settings.";
} elseif (($config["LONGITUDE"] ?? "0.000") == "0.000") {
    $warnings[] = "WARNING: Your longitude is not set properly. Please do so now in Tools -> Settings.";
}

// AJAX Request Detection - Handle AJAX requests before ANY HTML output
// These requests expect raw data/JSON/HTML fragments, not a full HTML page
$ajax_params = [
    'ajax_csv',           // spectrogram.php
    'ajax_detections',    // overview.php, todays_detections.php
    'ajax_left_chart',    // overview.php
    'ajax_center_chart',  // overview.php
    'fetch_chart_string', // overview.php
    'custom_image',       // overview.php
    'blacklistimage',     // overview.php
    'today_stats',        // todays_detections.php
];

$is_ajax_request = false;
foreach ($ajax_params as $param) {
    if (isset($_GET[$param])) {
        $is_ajax_request = true;
        break;
    }
}

if ($is_ajax_request && isset($_GET['view'])) {
    // Route directly to the view file without HTML wrapper
    $view = $_GET['view'];
    switch ($view) {
        case 'Overview':
            include(PAGES_PATH . '/overview.php');
            break;
        case 'Todays Detections':
            include(PAGES_PATH . '/todays_detections.php');
            break;
        case 'Spectrogram':
            include(PAGES_PATH . '/spectrogram.php');
            break;
        case 'Species Stats':
            include(PAGES_PATH . '/stats.php');
            break;
        default:
            // Unknown AJAX view, return empty
            break;
    }
    exit; // Stop processing - AJAX request handled
}
?>
<?php if (isset($_SESSION['behind']) && intval($_SESSION['behind']) >= 99): ?>
<style>
.updatenumber {
    width:30px !important;
}
</style>
<?php endif; ?>
<?php foreach ($warnings as $warning): ?>
<center style='color:red'><b><?php echo $warning; ?></b></center>
<?php endforeach; ?>
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <title>BirdNET-Pi DB</title>
    <link rel="stylesheet" href="<?php echo $color_scheme_css . '?v=' . time(); ?>">
</head>
<body>
<form action="" method="GET" id="views">
<div class="topnav" id="myTopnav">
    <button type="submit" name="view" value="Overview" form="views">Overview</button>
    <button type="submit" name="view" value="Todays Detections" form="views">Today's Detections</button>
    <button type="submit" name="view" value="Spectrogram" form="views">Spectrogram</button>
    <button type="submit" name="view" value="Species Stats" form="views">Best Recordings</button>
    <button type="submit" name="view" value="Streamlit" form="views">Species Stats</button>
    <button type="submit" name="view" value="Daily Charts" form="views">Daily Charts</button>
    <button type="submit" name="view" value="Weekly Report" form="views">Weekly Report</button>
    <button type="submit" name="view" value="Recordings" form="views">Recordings</button>
    <button type="submit" name="view" value="View Log" form="views">View Log</button>
    <button type="submit" name="view" value="Tools" form="views">Tools<?php echo $updatediv; ?></button>
    <button type="button" href="javascript:void(0);" class="icon" onclick="myFunction()"><img src="/assets/images/menu.png"></button>
</div>
</form>
<script type="text/javascript" src="/assets/js/plupload.full.min.js"></script>
<script>
window.onload = function() {
    var elements = document.querySelectorAll("button[name=view]");
    var setViewsOpacity = function() {
        document.getElementsByClassName("views")[0].style.opacity = "0.5";
    };
    for (var i = 0; i < elements.length; i++) {
        elements[i].addEventListener('click', setViewsOpacity, false);
    }
};
var topbuttons = document.querySelectorAll("button[form='views']");
if (window.location.search.substr(1) != '') {
    for (var i = 0; i < topbuttons.length; i++) {
        if (topbuttons[i].value == decodeURIComponent(window.location.search.substr(1)).replace(/\+/g, ' ').split('=').pop()) {
            topbuttons[i].classList.add("button-hover");
        }
    }
} else {
    topbuttons[0].classList.add("button-hover");
}
function copyOutput(elem) {
    elem.innerHTML = 'Copied!';
    const copyText = document.getElementsByTagName("pre")[0].textContent;
    const textArea = document.createElement('textarea');
    textArea.style.position = 'absolute';
    textArea.style.left = '-100%';
    textArea.textContent = copyText;
    document.body.append(textArea);
    textArea.select();
    document.execCommand("copy");
}
</script>

<div class="views">
<?php
/**
 * Helper function to update species list files
 */
function update_species_list($filename, $species, $add) {
    if ($add) {
        $str = file_get_contents($filename);
        $str = preg_replace("/(^[\r\n]*|[\r\n]+)[\s\t]*[\r\n]+/", "\n", $str);
        file_put_contents("$filename", "$str");
        foreach ($species as $selectedOption) {
            if (strpos($str, $selectedOption) === false) {
                file_put_contents($filename, htmlspecialchars_decode($selectedOption, ENT_QUOTES) . "\n", FILE_APPEND);
            }
        }
    } else {
        $str = file_get_contents($filename);
        $str = preg_replace('/^\h*\v+/m', '', $str);
        file_put_contents($filename, "$str");
        foreach ($species as $selectedOption) {
            $content = file_get_contents($filename);
            $newcontent = str_replace($selectedOption, "", "$content");
            $newcontent = str_replace(htmlspecialchars_decode($selectedOption, ENT_QUOTES), "", "$newcontent");
            file_put_contents($filename, "$newcontent");
        }
        $str = file_get_contents($filename);
        $str = preg_replace('/^\h*\v+/m', '', $str);
        file_put_contents($filename, "$str");
    }
}

// Species list file paths (in scripts directory)
$scripts_dir = PROJECT_ROOT . '/scripts';

if (isset($_GET['view'])) {
    $view = $_GET['view'];

    switch ($view) {
        case 'System Info':
            echo "<div style='padding: 2rem; text-align: center;'>";
            echo "<h2>System Info Deprecated</h2>";
            echo "<p>The phpsysinfo tool has been removed. Use the new <a href='/app'>Go API</a> for system information, or access the system via SSH.</p>";
            echo "</div>";
            break;

        case 'System Controls':
            ensure_authenticated();
            include(PAGES_PATH . '/system_controls.php');
            break;

        case 'Services':
            ensure_authenticated();
            include(PAGES_PATH . '/service_controls.php');
            break;

        case 'Spectrogram':
            include(PAGES_PATH . '/spectrogram.php');
            break;

        case 'View Log':
            echo "<body style=\"scroll:no;overflow-x:hidden;\"><iframe style=\"width:calc( 100% + 1em);\" src=\"/log\"></iframe></body>";
            break;

        case 'Overview':
            include(PAGES_PATH . '/overview.php');
            break;

        case 'Todays Detections':
            include(PAGES_PATH . '/todays_detections.php');
            break;

        case 'Kiosk':
            $kiosk = true;
            include(PAGES_PATH . '/todays_detections.php');
            break;

        case 'Species Stats':
            include(PAGES_PATH . '/stats.php');
            break;

        case 'Weekly Report':
            include(PAGES_PATH . '/weekly_report.php');
            break;

        case 'Streamlit':
            echo "<iframe src=\"/stats\"></iframe>";
            break;

        case 'Daily Charts':
            include(PAGES_PATH . '/history.php');
            break;

        case 'Tools':
            ensure_authenticated();
            echo "<div class=\"centered\">
                <form action=\"\" method=\"GET\" id=\"views\">
                <button type=\"submit\" name=\"view\" value=\"Settings\" form=\"views\">Settings</button>
                <button type=\"submit\" name=\"view\" value=\"System Info\" form=\"views\">System Info</button>
                <button type=\"submit\" name=\"view\" value=\"System Controls\" form=\"views\">System Controls" . $updatediv . "</button>
                <button type=\"submit\" name=\"view\" value=\"Services\" form=\"views\">Services</button>
                <button type=\"submit\" name=\"view\" value=\"File\" form=\"views\">File Manager</button>
                <button type=\"submit\" name=\"view\" value=\"Adminer\" form=\"views\">Database Maintenance</button>
                <button type=\"submit\" name=\"view\" value=\"Webterm\" form=\"views\">Web Terminal</button>
                <button type=\"submit\" name=\"view\" value=\"Included\" form=\"views\">Custom Species List</button>
                <button type=\"submit\" name=\"view\" value=\"Excluded\" form=\"views\">Excluded Species List</button>
                <button type=\"submit\" name=\"view\" value=\"Whitelisted\" form=\"views\">Whitelist Species List</button>
                <button type=\"submit\" name=\"view\" value=\"Species Management\" form=\"views\">Species Management</button>
                </form>
                </div>";
            break;

        case 'Recordings':
            include(PAGES_PATH . '/play.php');
            break;

        case 'Settings':
            include(PAGES_PATH . '/config.php');
            break;

        case 'Advanced':
            include(PAGES_PATH . '/advanced.php');
            break;

        case 'Included':
            ensure_authenticated();
            if (isset($_GET['species']) && (isset($_GET['add']) || isset($_GET['del']))) {
                update_species_list("$scripts_dir/include_species_list.txt", $_GET['species'], isset($_GET['add']));
            }
            $species_list = "include";
            include(PAGES_PATH . '/species_list.php');
            break;

        case 'Excluded':
            ensure_authenticated();
            if (isset($_GET['species']) && (isset($_GET['add']) || isset($_GET['del']))) {
                update_species_list("$scripts_dir/exclude_species_list.txt", $_GET['species'], isset($_GET['add']));
            }
            $species_list = "exclude";
            include(PAGES_PATH . '/species_list.php');
            break;

        case 'Whitelisted':
            ensure_authenticated();
            if (isset($_GET['species']) && (isset($_GET['add']) || isset($_GET['del']))) {
                update_species_list("$scripts_dir/whitelist_species_list.txt", $_GET['species'], isset($_GET['add']));
            }
            $species_list = "whitelist";
            include(PAGES_PATH . '/species_list.php');
            break;

        case 'Species Management':
            ensure_authenticated();
            include(PAGES_PATH . '/species_tools.php');
            break;

        case 'File':
            echo "<div style='padding: 2rem; text-align: center;'>";
            echo "<h2>File Manager Deprecated</h2>";
            echo "<p>The PHP file manager has been removed. Use the new <a href='/app'>Recordings browser</a> in the Go app, or access files via SSH/SFTP.</p>";
            echo "</div>";
            break;

        case 'Adminer':
            echo "<iframe src='/adminer/adminer.php'></iframe>";
            break;

        case 'Webterm':
            ensure_authenticated('You cannot access the web terminal');
            echo "<iframe src='/terminal'></iframe>";
            break;

        default:
            include(PAGES_PATH . '/overview.php');
            break;
    }
} elseif (isset($_GET['submit'])) {
    ensure_authenticated();
    $allowedCommands = array(
        'sudo systemctl stop livestream.service && sudo systemctl stop icecast2.service',
        'sudo systemctl restart livestream.service && sudo systemctl restart icecast2.service',
        'sudo systemctl disable --now livestream.service && sudo systemctl disable icecast2 && sudo systemctl stop icecast2.service',
        'sudo systemctl enable icecast2 && sudo systemctl start icecast2.service && sudo systemctl enable --now livestream.service',
        'sudo systemctl stop web_terminal.service',
        'sudo systemctl restart web_terminal.service',
        'sudo systemctl disable --now web_terminal.service',
        'sudo systemctl enable --now web_terminal.service',
        'sudo systemctl stop birdnet_log.service',
        'sudo systemctl restart birdnet_log.service',
        'sudo systemctl disable --now birdnet_log.service',
        'sudo systemctl enable --now birdnet_log.service',
        'sudo systemctl stop birdnet_analysis.service',
        'sudo systemctl restart birdnet_analysis.service',
        'sudo systemctl disable --now birdnet_analysis.service',
        'sudo systemctl enable --now birdnet_analysis.service',
        'sudo systemctl stop birdnet_stats.service',
        'sudo systemctl restart birdnet_stats.service',
        'sudo systemctl disable --now birdnet_stats.service',
        'sudo systemctl enable --now birdnet_stats.service',
        'sudo systemctl stop birdnet_recording.service',
        'sudo systemctl restart birdnet_recording.service',
        'sudo systemctl disable --now birdnet_recording.service',
        'sudo systemctl enable --now birdnet_recording.service',
        'sudo systemctl stop chart_viewer.service',
        'sudo systemctl restart chart_viewer.service',
        'sudo systemctl disable --now chart_viewer.service',
        'sudo systemctl enable --now chart_viewer.service',
        'sudo systemctl stop spectrogram_viewer.service',
        'sudo systemctl restart spectrogram_viewer.service',
        'sudo systemctl disable --now spectrogram_viewer.service',
        'sudo systemctl enable --now spectrogram_viewer.service',
        'sudo systemctl enable ' . get_service_mount_name() . ' && sudo reboot',
        'sudo systemctl disable ' . get_service_mount_name() . ' && sudo reboot',
        'stop_core_services.sh',
        'restart_services.sh',
        'sudo reboot',
        'update_birdnet.sh',
        'sudo shutdown now',
        'sudo clear_all_data.sh',
        "$restore"
    );
    $command = $_GET['submit'];
    if (in_array($command, $allowedCommands)) {
        $initcommand = $command;
        if (strpos($command, "systemctl") !== false) {
            if (strpos($command, " && ") !== false) {
                $separate_commands = explode("&&", trim($command));
                $new_multiservice_status_command = "";
                foreach ($separate_commands as $indiv_service_command) {
                    $separate_command_tmp = explode(" ", trim($indiv_service_command));
                    $new_multiservice_status_command .= " " . trim(end($separate_command_tmp));
                }
                $service_names = $new_multiservice_status_command;
            } else {
                $tmp = explode(" ", trim($command));
                $service_names = end($tmp);
            }
            $command .= " & sleep 3;sudo systemctl status " . $service_names;
        }
        if ($initcommand == "update_birdnet.sh") {
            session_unset();
        }
        $results = shell_exec("$command 2>&1");
        $results = str_replace("FAILURE", "<span style='color:red'>FAILURE</span>", $results);
        $results = str_replace("failed", "<span style='color:red'>failed</span>", $results);
        $results = str_replace("active (running)", "<span style='color:green'><b>active (running)</b></span>", $results);
        $results = str_replace("Your branch is up to date", "<span style='color:limegreen'><b>Your branch is up to date</b></span>", $results);
        $results = str_replace("(+)", "(<span style='color:lime;font-weight:bold'>+</span>)", $results);
        $results = str_replace("(-)", "(<span style='color:red;font-weight:bold'>-</span>)", $results);

        // Split and format lines
        $lines = explode("\n", $results);
        foreach ($lines as &$line) {
            if (preg_match('/^(.+?)\s*\|\s*(\d+)\s*([\+\- ]+)(\d+)?$/', $line, $matches)) {
                $filename = $matches[1];
                $count = $matches[2];
                $diff = $matches[3];
                $delta = $matches[4] ?? '';
                $diff_array = str_split($diff);
                $indicators = array_map(function ($d) use ($delta) {
                    if ($d === '+') {
                        return "<span style='color:lime;'><b>+</b></span>";
                    } elseif ($d === '-') {
                        return "<span style='color:red;'><b>-</b></span>";
                    } elseif ($d === ' ') {
                        if ($delta !== '') {
                            return 'A';
                        } else {
                            return ' ';
                        }
                    }
                }, $diff_array);
                $line = sprintf('%-35s|%3d %s%s', $filename, $count, implode('', $indicators), $delta);
            }
        }

        $output = implode("\n", $lines);
        $results = $output;

        // Remove script tags (XSS protection)
        $results = preg_replace('#<script(.*?)>(.*?)</script>#is', '', $results);
        if (strlen($results) == 0) {
            $results = "This command has no output.";
        }
        echo "<table style='min-width:70%;'><tr class='relative'><th>Output of command:`" . $initcommand . "`<button class='copyimage' style='right:40px' onclick='copyOutput(this);'>Copy</button></th></tr><tr><td style='padding-left: 0px;padding-right: 0px;padding-bottom: 0px;padding-top: 0px;'><pre class='bash' style='text-align:left;margin:0px'>$results</pre></td></tr></table>";
    }
    if (function_exists('ob_end_flush')) {
        @ob_end_flush();
    }
} else {
    include(PAGES_PATH . '/overview.php');
}
?>
<script>
function myFunction() {
    var x = document.getElementById("myTopnav");
    if (x.className === "topnav") {
        x.className += " responsive";
    } else {
        x.className = "topnav";
    }
}
function setLiveStreamVolume(vol) {
    var audioElements = document.querySelectorAll(".custom-audio-player audio");
    audioElements.forEach(audioEl => {
        if (audioEl) {
            audioEl.volume = vol;
        }
    });
}
window.onbeforeunload = function(event) {
    var audioElements = document.querySelectorAll(".custom-audio-player audio");
    audioElements.forEach(audioEl => {
        if (audioEl) {
            audioEl.volume = 1;
        }
    });
}

function getTheDate(increment) {
    var theDate = "<?php if (isset($theDate)) echo $theDate; ?>";
    d = new Date(theDate);
    d.setDate(d.getDate(theDate) + increment);
    yyyy = d.getFullYear();
    mm = d.getMonth() + 1; if (mm < 10) mm = "0" + mm;
    dd = d.getDate(); if (dd < 10) dd = "0" + dd;
    document.getElementById("SwipeSpinner").hidden = false;
    window.location = "?date=" + yyyy + "-" + mm + "-" + dd + "&view=Daily+Charts";
}

function installKeyAndSwipeEventHandler() {
    for (var i = 0; i < topbuttons.length; i++) {
        if (topbuttons[i].textContent == "Daily Charts" &&
            topbuttons[i].className == "button-hover") {

            document.onkeydown = function(event) {
                switch (event.keyCode) {
                    case 37: // Left key
                        getTheDate(-1);
                        break;
                    case 39: // Right key
                        getTheDate(+1);
                        break;
                }
            };

            let touchstartX = 0;
            let diffX = 0;
            let touchstartY = 0;
            let diffY = 0;
            let startTime = 0;
            let diffTime = 0;

            function checkDirection() {
                if (Math.abs(diffX) > Math.abs(diffY) && diffTime < 350) {
                    if (diffX > 20) getTheDate(+1);
                    if (diffX < -20) getTheDate(-1);
                }
            }

            document.addEventListener('touchstart', e => {
                touchstartX = e.changedTouches[0].screenX;
                touchstartY = e.changedTouches[0].screenY;
                startTime = Date.now();
            });

            document.addEventListener('touchend', e => {
                diffX = touchstartX - e.changedTouches[0].screenX;
                diffY = touchstartY - e.changedTouches[0].screenY;
                diffTime = Date.now() - startTime;
                checkDirection();
            });
        }
    }
}

installKeyAndSwipeEventHandler();
</script>
</div>
</body>
</html>
