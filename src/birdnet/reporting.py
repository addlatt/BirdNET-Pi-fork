"""Reporting and post-processing for BirdNET-Pi detections."""
import glob
import json
import logging
import os
import sqlite3
import subprocess
import tempfile
import io
import soundfile
from time import sleep
from typing import Any

import requests
from PIL import Image, ImageDraw, ImageFont

from .config import get_settings, get_birddb_path
from .helpers import get_font, DB_PATH
from .classes import Detection, ParseFileName
from .notifications import sendAppriseNotifications

log = logging.getLogger(__name__)

# BirdNET analysis constants
DETECTION_WINDOW_SECONDS = 3  # Duration of each BirdNET detection window (fixed by model)


def extract(in_file: str, out_file: str, start: float, stop: float) -> str:
    """Extract a segment from an audio file using sox."""
    result = subprocess.run(['sox', '-V1', f'{in_file}', f'{out_file}', 'trim', f'={start}', f'={stop}'],
                            check=True, capture_output=True)
    ret = result.stdout.decode('utf-8')
    err = result.stderr.decode('utf-8')
    if err:
        raise RuntimeError(f'{ret}:\n {err}')
    return ret


def extract_safe(in_file: str, out_file: str, start: float, stop: float) -> None:
    """Extract audio with configurable context padding around the detection."""
    conf = get_settings()
    # This section sets the SPACER that will be used to pad the audio clip with
    # context. If EXTRACTION_LENGTH is 10, for instance, DETECTION_WINDOW_SECONDS
    # are removed from that value and divided by 2, so that the detection window
    # is centered within equal audio context before and after.
    try:
        ex_len = int(conf.get('EXTRACTION_LENGTH', 6))
    except (ValueError, TypeError):
        ex_len = 6
    spacer = (ex_len - DETECTION_WINDOW_SECONDS) / 2
    safe_start = max(0, start - spacer)
    safe_stop = min(int(conf.get('RECORDING_LENGTH', 15)), stop + spacer)

    extract(in_file, out_file, safe_start, safe_stop)


def spectrogram(in_file: str, title: str, comment: str, raw: int = 0) -> None:
    """Generate a spectrogram image from an audio file with title and comment."""
    fd, tmp_file = tempfile.mkstemp(suffix='.png')
    os.close(fd)
    args = ['sox', '-V1', f'{in_file}', '-n', 'remix', '1', 'rate', '24k', 'spectrogram',
            '-t', '', '-c', '', '-o', tmp_file]
    args += ['-r'] if int(raw) else []

    result = subprocess.run(args, check=True, capture_output=True)
    ret = result.stdout.decode('utf-8')
    err = result.stderr.decode('utf-8')
    if err:
        raise RuntimeError(f'{ret}:\n {err}')
    img = Image.open(tmp_file)
    height = img.size[1]
    width = img.size[0]
    draw = ImageDraw.Draw(img)
    title_font = ImageFont.truetype(get_font()['path'], 13)
    _, _, w, _ = draw.textbbox((0, 0), title, font=title_font)
    draw.text(((width-w)/2, 6), title, fill="white", font=title_font)

    comment_font = ImageFont.truetype(get_font()['path'], 11)
    _, _, _, h = draw.textbbox((0, 0), comment, font=comment_font)
    draw.text((1, height - (h + 1)), comment, fill="white", font=comment_font)
    img.save(f'{in_file}.png')
    os.remove(tmp_file)


def extract_detection(file: ParseFileName, detection: Detection) -> str:
    """Extract audio clip for a detection and generate spectrogram."""
    conf = get_settings()
    new_file_name = f'{detection.common_name_safe}-{detection.confidence_pct}-{detection.date}-birdnet-{file.RTSP_id}{detection.time}.{conf["AUDIOFMT"]}'
    new_dir = os.path.join(conf['EXTRACTED'], 'By_Date', f'{detection.date}', f'{detection.common_name_safe}')
    new_file = os.path.join(new_dir, new_file_name)
    if os.path.isfile(new_file):
        log.warning('Extraction exists. Moving on: %s', new_file)
    else:
        os.makedirs(new_dir, exist_ok=True)
        extract_safe(file.file_name, new_file, detection.start, detection.stop)
        spectrogram(new_file, detection.common_name, new_file.replace(os.path.expanduser('~/'), ''), conf['RAW_SPECTROGRAM'])
    return new_file


def write_to_db(file: ParseFileName, detection: Detection) -> None:
    """Write a detection to the SQLite database."""
    conf = get_settings()
    # Connect to SQLite Database
    for attempt_number in range(3):
        try:
            con = sqlite3.connect(DB_PATH)
            cur = con.cursor()
            cur.execute("INSERT INTO detections VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
                        (detection.date, detection.time, detection.scientific_name, detection.common_name, detection.confidence,
                         conf['LATITUDE'], conf['LONGITUDE'], conf['CONFIDENCE'], str(detection.week), conf['SENSITIVITY'],
                         conf['OVERLAP'], os.path.basename(detection.file_name_extr)))
            # (Date, Time, Sci_Name, Com_Name, str(score),
            # Lat, Lon, Cutoff, Week, Sens,
            # Overlap, File_Name))

            con.commit()
            con.close()
            break
        except BaseException as e:
            log.warning("Database busy: %s", e)
            sleep(2)


def summary(file: ParseFileName, detection: Detection) -> str:
    """Generate a summary string for a detection."""
    # Date;Time;Sci_Name;Com_Name;Confidence;Lat;Lon;Cutoff;Week;Sens;Overlap
    # 2023-03-03;12:48:01;Phleocryptes melanops;Wren-like Rushbird;0.76950216;-1;-1;0.7;9;1.25;0.0
    conf = get_settings()
    s = (f'{detection.date};{detection.time};{detection.scientific_name};{detection.common_name};'
         f'{detection.confidence};'
         f'{conf["LATITUDE"]};{conf["LONGITUDE"]};{conf["CONFIDENCE"]};{detection.week};{conf["SENSITIVITY"]};'
         f'{conf["OVERLAP"]}')
    return s


def write_to_file(file: ParseFileName, detection: Detection) -> None:
    """Append a detection summary to BirdDB.txt."""
    with open(get_birddb_path(), 'a') as rfile:
        rfile.write(f'{summary(file, detection)}\n')


def update_json_file(file: ParseFileName, detections: list[Detection]) -> None:
    """Update JSON sidecar file, removing old ones first."""
    if file.RTSP_id is None:
        mask = f'{os.path.dirname(file.file_name)}/*.json'
    else:
        mask = f'{os.path.dirname(file.file_name)}/*{file.RTSP_id}*.json'
    for f in glob.glob(mask):
        log.debug(f'deleting {f}')
        os.remove(f)
    write_to_json_file(file, detections)


def write_to_json_file(file: ParseFileName, detections: list[Detection]) -> None:
    """Write detections to a JSON sidecar file."""
    conf = get_settings()
    json_file = f'{file.file_name}.json'
    log.debug(f'WRITING RESULTS TO {json_file}')
    dets: dict[str, Any] = {
        'file_name': os.path.basename(json_file),
        'timestamp': file.iso8601,
        'delay': conf['RECORDING_LENGTH'],
        'detections': [
            {"start": det.start, "common_name": det.common_name, "confidence": det.confidence}
            for det in detections
        ]
    }
    with open(json_file, 'w') as rfile:
        rfile.write(json.dumps(dets))
    log.debug(f'DONE! WROTE {len(detections)} RESULTS.')


def apprise(file: ParseFileName, detections: list[Detection]) -> None:
    """Send Apprise notifications for detections."""
    species_apprised_this_run: list[str] = []
    conf = get_settings()

    for detection in detections:
        # Apprise of detection if not already alerted this run.
        if detection.species not in species_apprised_this_run:
            try:
                sendAppriseNotifications(detection.scientific_name, detection.common_name, str(detection.confidence), str(detection.confidence_pct),
                                         os.path.basename(detection.file_name_extr), detection.date, detection.time, str(detection.week),
                                         conf['LATITUDE'], conf['LONGITUDE'], conf['CONFIDENCE'], conf['SENSITIVITY'], conf['OVERLAP'])

            except BaseException as e:
                log.exception('Error during Apprise:', exc_info=e)

            species_apprised_this_run.append(detection.species)


def bird_weather(file: ParseFileName, detections: list[Detection]) -> None:
    """Post detections to the BirdWeather API."""
    conf = get_settings()
    if conf['BIRDWEATHER_ID'] == "":
        return
    if detections:
        try:
            data, samplerate = soundfile.read(file.file_name)
            buf = io.BytesIO()
            soundfile.write(buf, data, samplerate, format='FLAC')
            flac_data = buf.getvalue()
        except Exception as e:
            log.error("Error during FLAC conversion: %s", e)
            return

        # POST soundscape to server
        soundscape_url = (f'https://app.birdweather.com/api/v1/stations/'
                          f'{conf["BIRDWEATHER_ID"]}/soundscapes?timestamp={file.iso8601}')

        try:
            response = requests.post(url=soundscape_url, data=flac_data, timeout=30,
                                     headers={'Content-Type': 'audio/flac'})
            log.info("Soundscape POST Response Status - %d", response.status_code)
            sdata = response.json()
        except BaseException as e:
            log.error("Cannot POST soundscape: %s", e)
            return
        if not sdata.get('success'):
            log.error(sdata.get('message'))
            return
        soundscape_id = sdata['soundscape']['id']

        for detection in detections:
            # POST detection to server
            detection_url = f'https://app.birdweather.com/api/v1/stations/{conf["BIRDWEATHER_ID"]}/detections'

            post_data: dict[str, Any] = {
                'timestamp': detection.iso8601,
                'lat': conf['LATITUDE'],
                'lon': conf['LONGITUDE'],
                'soundscapeId': soundscape_id,
                'soundscapeStartTime': detection.start,
                'soundscapeEndTime': detection.stop,
                'commonName': detection.common_name,
                'scientificName': detection.scientific_name,
                'algorithm': '2p4' if conf['MODEL'] == 'BirdNET_GLOBAL_6K_V2.4_Model_FP16' else 'alpha',
                'confidence': detection.confidence
            }

            log.debug(post_data)
            try:
                response = requests.post(detection_url, json=post_data, timeout=20)
                log.info("Detection POST Response Status - %d", response.status_code)
            except BaseException as e:
                log.error("Cannot POST detection: %s", e)


def heartbeat() -> None:
    """Send a heartbeat request to the configured URL."""
    conf = get_settings()
    if conf['HEARTBEAT_URL']:
        try:
            result = requests.get(url=conf['HEARTBEAT_URL'], timeout=10)
            log.info('Heartbeat: %s', result.text)
        except BaseException as e:
            log.error('Error during heartbeat: %s', e)
