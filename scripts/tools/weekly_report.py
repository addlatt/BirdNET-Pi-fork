#!/usr/bin/env python3
"""
Weekly Report Generator for BirdNET-Pi

Generates a weekly summary of bird detections and sends it via Apprise
to configured notification channels (e.g., Discord).

Run via cron: 0 9 * * 6 (every Saturday at 9 AM)
"""
import argparse
import sys
from datetime import datetime, timedelta
from pathlib import Path

# Add src to path for imports
sys.path.insert(0, str(Path(__file__).parent.parent.parent / 'src'))

import apprise
from birdnet.config import get_settings, get_apprise_config_path


def get_week_boundaries() -> tuple[str, str, str, str]:
    """
    Calculate date boundaries for this week and prior week.

    Week runs Sunday to Saturday.
    Returns: (start_date, end_date, prior_start_date, prior_end_date)
    """
    today = datetime.now()

    # Find last Saturday (end of reporting week)
    days_since_saturday = (today.weekday() + 2) % 7
    if days_since_saturday == 0:
        days_since_saturday = 7  # If today is Saturday, report on last week

    end_date = today - timedelta(days=days_since_saturday)
    start_date = end_date - timedelta(days=6)

    prior_end_date = start_date - timedelta(days=1)
    prior_start_date = prior_end_date - timedelta(days=6)

    return (
        start_date.strftime('%Y-%m-%d'),
        end_date.strftime('%Y-%m-%d'),
        prior_start_date.strftime('%Y-%m-%d'),
        prior_end_date.strftime('%Y-%m-%d'),
    )


def get_db_connection():
    """Get a read-only SQLite connection to the birds database."""
    import sqlite3
    from birdnet.config import get_db_path

    db_path = get_db_path()
    conn = sqlite3.connect(f"file:{db_path}?mode=ro", uri=True)
    conn.row_factory = sqlite3.Row
    return conn


def safe_percentage(count: int, prior_count: int) -> int:
    """Calculate percentage change, handling division by zero."""
    if prior_count == 0:
        return 100 if count > 0 else 0
    return round(((count - prior_count) / prior_count) * 100)


def format_percentage(pct: int) -> str:
    """Format percentage with + or - prefix."""
    if pct > 0:
        return f"+{pct}%"
    elif pct < 0:
        return f"{pct}%"
    else:
        return "0%"


def generate_report(dry_run: bool = False) -> str:
    """Generate the weekly report text."""
    start_date, end_date, prior_start, prior_end = get_week_boundaries()

    conn = get_db_connection()
    cursor = conn.cursor()

    # Get species counts for this week
    cursor.execute('''
        SELECT Sci_Name, Com_Name, COUNT(*) as count
        FROM detections
        WHERE Date BETWEEN ? AND ?
        GROUP BY Sci_Name
        ORDER BY COUNT(*) DESC
    ''', (start_date, end_date))
    species_this_week = cursor.fetchall()

    # Build species data with comparisons
    species_data = []
    for row in species_this_week:
        sci_name = row['Sci_Name']
        com_name = row['Com_Name']
        count = row['count']

        # Get prior week count
        cursor.execute('''
            SELECT COUNT(*) as count FROM detections
            WHERE Sci_Name = ? AND Date BETWEEN ? AND ?
        ''', (sci_name, prior_start, prior_end))
        prior_count = cursor.fetchone()['count']

        # Check if first time seen (no detections outside this week)
        cursor.execute('''
            SELECT COUNT(*) as count FROM detections
            WHERE Sci_Name = ? AND Date NOT BETWEEN ? AND ?
        ''', (sci_name, start_date, end_date))
        is_first_seen = cursor.fetchone()['count'] == 0

        species_data.append({
            'com_name': com_name,
            'count': count,
            'pct_change': safe_percentage(count, prior_count),
            'is_first_seen': is_first_seen,
        })

    # Get total detections
    cursor.execute('''
        SELECT COUNT(*) as count FROM detections
        WHERE Date BETWEEN ? AND ?
    ''', (start_date, end_date))
    total_count = cursor.fetchone()['count']

    cursor.execute('''
        SELECT COUNT(*) as count FROM detections
        WHERE Date BETWEEN ? AND ?
    ''', (prior_start, prior_end))
    prior_total = cursor.fetchone()['count']

    # Get unique species count
    cursor.execute('''
        SELECT COUNT(DISTINCT Sci_Name) as count FROM detections
        WHERE Date BETWEEN ? AND ?
    ''', (start_date, end_date))
    species_count = cursor.fetchone()['count']

    cursor.execute('''
        SELECT COUNT(DISTINCT Sci_Name) as count FROM detections
        WHERE Date BETWEEN ? AND ?
    ''', (prior_start, prior_end))
    prior_species = cursor.fetchone()['count']

    conn.close()

    # Calculate week number
    week_num = datetime.strptime(end_date, '%Y-%m-%d').isocalendar()[1]

    # Build report
    lines = []
    lines.append(f"**BirdNET-Pi: Week {week_num} Report**")
    lines.append(f"_{start_date} to {end_date}_")
    lines.append("")

    total_pct = format_percentage(safe_percentage(total_count, prior_total))
    species_pct = format_percentage(safe_percentage(species_count, prior_species))

    lines.append(f"**Total Detections:** {total_count} ({total_pct})")
    lines.append(f"**Unique Species:** {species_count} ({species_pct})")
    lines.append("")

    # Top 10 species
    lines.append("**Top 10 Species:**")
    for i, sp in enumerate(species_data[:10], 1):
        pct = format_percentage(sp['pct_change'])
        lines.append(f"{i}. {sp['com_name']} - {sp['count']} ({pct})")

    # First-time species
    first_timers = [sp for sp in species_data if sp['is_first_seen']]
    if first_timers:
        lines.append("")
        lines.append("**New Species (First Time Detected):**")
        for sp in first_timers:
            lines.append(f"- {sp['com_name']} ({sp['count']})")

    lines.append("")
    lines.append(f"_Percentages relative to week {week_num - 1}_")

    return '\n'.join(lines)


def send_report(report: str) -> bool:
    """Send the report via Apprise."""
    settings = get_settings()

    # Check if weekly report is enabled (can be int or string)
    weekly_report_enabled = settings.get('APPRISE_WEEKLY_REPORT')
    if str(weekly_report_enabled) != '1':
        print(f"Weekly report is disabled (APPRISE_WEEKLY_REPORT = {weekly_report_enabled})")
        return False

    apprise_config_path = get_apprise_config_path()
    if not apprise_config_path.exists() or apprise_config_path.stat().st_size == 0:
        print(f"Apprise config not found or empty: {apprise_config_path}")
        return False

    # Initialize Apprise
    apobj = apprise.Apprise()
    config = apprise.AppriseConfig()
    config.add(str(apprise_config_path))
    apobj.add(config)

    # Send notification
    result = apobj.notify(
        body=report,
        title="BirdNET-Pi Weekly Report",
    )

    return result


def main():
    parser = argparse.ArgumentParser(description='Generate and send BirdNET-Pi weekly report')
    parser.add_argument('--dry-run', action='store_true',
                        help='Generate report but do not send')
    parser.add_argument('--print', dest='print_report', action='store_true',
                        help='Print report to stdout')
    args = parser.parse_args()

    try:
        report = generate_report()

        if args.print_report or args.dry_run:
            print(report)
            print()

        if not args.dry_run:
            success = send_report(report)
            if success:
                print("Weekly report sent successfully")
                return 0
            else:
                print("Failed to send weekly report")
                return 1
        else:
            print("Dry run - report not sent")
            return 0

    except Exception as e:
        print(f"Error generating weekly report: {e}")
        import traceback
        traceback.print_exc()
        return 1


if __name__ == '__main__':
    sys.exit(main())
