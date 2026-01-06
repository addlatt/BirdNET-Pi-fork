"""
BirdNET-Pi Helper Functions

Provides utility functions for file handling, language support, and model labels.
Configuration is loaded from the centralized config module.
"""
import glob
import json
import os
import re
import subprocess
from collections import OrderedDict

# Import configuration from centralized module
from .config import (
    get_config,
    get_settings,
    get_base_path,
    get_db_path,
    get_model_path,
    get_font_dir,
    get_analyzing_now,
    BASE_PATH,
    DB_PATH,
    MODEL_PATH,
    FONT_DIR,
)

# Backwards-compatible ANALYZING_NOW - now dynamically computed from RECS_DIR
# Use get_analyzing_now() for dynamic path that respects config
ANALYZING_NOW = str(get_analyzing_now())


def get_font():
    """Get font configuration based on database language."""
    conf = get_config()
    font_dir = get_font_dir()

    if conf.get('DATABASE_LANG') == 'ar':
        ret = {'font.family': 'Noto Sans Arabic', 'path': str(font_dir / 'NotoSansArabic-Regular.ttf')}
    elif conf.get('DATABASE_LANG') in ['ja', 'zh_CN', 'zh_TW']:
        ret = {'font.family': 'Noto Sans JP', 'path': str(font_dir / 'NotoSansJP-Regular.ttf')}
    elif conf.get('DATABASE_LANG') == 'ko':
        ret = {'font.family': 'Noto Sans KR', 'path': str(font_dir / 'NotoSansKR-Regular.ttf')}
    elif conf.get('DATABASE_LANG') == 'th':
        ret = {'font.family': 'Noto Sans Thai', 'path': str(font_dir / 'NotoSansThai-Regular.ttf')}
    else:
        ret = {'font.family': 'Roboto Flex', 'path': str(font_dir / 'RobotoFlex-Regular.ttf')}
    return ret


def get_open_files_in_dir(dir_name):
    """Get list of open files in a directory using lsof."""
    result = subprocess.run(['lsof', '-w', '-Fn', '+D', f'{dir_name}'], check=False, capture_output=True)
    ret = result.stdout.decode('utf-8')
    err = result.stderr.decode('utf-8')
    if err:
        raise RuntimeError(f'{ret}:\n {err}')
    names = [line.lstrip('n') for line in ret.splitlines() if line.startswith('n')]
    return names


def get_wav_files():
    """Get list of WAV files available for processing."""
    conf = get_config()
    recs_dir = conf.get('RECS_DIR', os.path.expanduser('~/BirdSongs'))
    files = (glob.glob(os.path.join(recs_dir, '*/*/*.wav')) +
             glob.glob(os.path.join(recs_dir, 'StreamData/*.wav')))
    files.sort()
    files = [os.path.join(recs_dir, file) for file in files]
    rec_dir = os.path.join(recs_dir, 'StreamData')
    open_recs = get_open_files_in_dir(rec_dir)
    files = [file for file in files if file not in open_recs]
    return files


def get_language(language=None):
    """Load language labels from JSON file."""
    if language is None:
        conf = get_config()
        language = conf.get('DATABASE_LANG', 'en')
    model_path = get_model_path()
    file_name = model_path / 'l18n' / f'labels_{language}.json'
    with open(file_name) as f:
        ret = json.loads(f.read())
    return ret


def save_language(labels, language):
    """Save language labels to JSON file."""
    model_path = get_model_path()
    file_name = model_path / 'l18n' / f'labels_{language}.json'
    with open(file_name, 'w') as f:
        f.write(json.dumps(OrderedDict(sorted(labels.items())), indent=2, ensure_ascii=False))


def get_model_labels(model=None):
    """Get list of species labels from model file."""
    if model is None:
        conf = get_config()
        model = conf.get('MODEL', 'BirdNET_GLOBAL_6K_V2.4_Model_FP16')
    model_path = get_model_path()
    file_name = model_path / f'{model}_Labels.txt'
    with open(file_name) as f:
        labels = [line.strip() for line in f.readlines()]
    if labels and labels[0].count('_') == 1:
        labels = [re.sub(r'_.+$', '', label) for label in labels]
    return labels


def set_label_file():
    """Generate combined labels file with translations."""
    lang = get_language()
    labels = [f'{label}_{lang[label]}\n' for label in get_model_labels()]
    model_path = get_model_path()
    file_name = model_path / 'labels.txt'
    if os.path.islink(str(file_name)):
        os.remove(str(file_name))
    with open(file_name, 'w') as f:
        f.writelines(labels)
