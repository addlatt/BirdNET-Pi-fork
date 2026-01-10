"""Database access layer for BirdNET-Pi detections (read-only)."""
import sqlite3
import time as timeim
from datetime import datetime
from typing import Any, Optional

from .helpers import DB_PATH

_DB: Optional[sqlite3.Connection] = None


def get_db() -> sqlite3.Connection:
    """Get or create a read-only database connection."""
    global _DB
    if _DB is None:
        con = sqlite3.connect(f"file:{DB_PATH}?mode=ro", uri=True)
        con.row_factory = sqlite3.Row
        _DB = con
    return _DB


def get_records(select_sql: str, params: Optional[tuple] = None) -> list[sqlite3.Row]:
    """Execute a SELECT query and return all matching rows."""
    con = get_db()
    try:
        if params:
            cur = con.execute(select_sql, params)
        else:
            cur = con.execute(select_sql)
        records = cur.fetchall()
    except sqlite3.Error as e:
        print(e)
        timeim.sleep(2)
        records = []
    return records


def get_record(select_sql: str, params: Optional[tuple] = None) -> Optional[dict[str, Any]]:
    """Execute a SELECT query and return the first row as a dict, or None."""
    records = get_records(select_sql, params)
    return dict(records[0]) if records else None


def get_latest() -> Optional[dict[str, Any]]:
    """Get the most recent detection."""
    select_sql = "SELECT * FROM detections ORDER BY Date DESC, Time DESC LIMIT 1"
    return get_record(select_sql)


def get_todays_count_for(sci_name: str) -> int:
    """Get the number of detections for a species today."""
    today = datetime.now().strftime("%Y-%m-%d")
    select_sql = "SELECT COUNT(*) FROM detections WHERE Date = DATE(?) AND Sci_Name = ?"
    records = get_records(select_sql, (today, sci_name))
    return records[0][0] if records else 0


def get_this_weeks_count_for(sci_name: str) -> int:
    """Get the number of detections for a species in the last 7 days."""
    today = datetime.now().strftime("%Y-%m-%d")
    select_sql = "SELECT COUNT(*) FROM detections WHERE Date >= DATE(?, '-7 day') AND Sci_Name = ?"
    records = get_records(select_sql, (today, sci_name))
    return records[0][0] if records else 0


def get_summary() -> dict[str, int]:
    """Get summary statistics for detections."""
    total_count = get_record("SELECT COUNT(*) as total_count FROM detections")
    todays_count = get_record("SELECT COUNT(*) as todays_count FROM detections WHERE Date == DATE('now', 'localtime')")
    hour_count = get_record("SELECT COUNT(*) as hour_count FROM detections "
                            "WHERE Date == Date('now', 'localtime') AND TIME >= TIME('now', 'localtime', '-1 hour')")
    todays_species_tally = get_record("SELECT COUNT(DISTINCT(Sci_Name)) as todays_species_tally FROM detections WHERE Date == Date('now','localtime')")
    species_tally = get_record("SELECT COUNT(DISTINCT(Sci_Name)) as species_tally FROM detections")

    summary = {**total_count, **todays_count, **hour_count, **todays_species_tally, **species_tally}
    return summary


def get_species_by(sort_by: Optional[str] = None, date: Optional[str] = None) -> list[sqlite3.Row]:
    """Get species grouped and sorted by the specified criteria."""
    where = "" if date is None else "WHERE Date = ?"
    params = (date,) if date is not None else None

    base_select = "SELECT Date, Time, File_Name, Com_Name, Sci_Name, COUNT(*) as Count, MAX(Confidence) as MaxConfidence FROM detections"

    if sort_by == "occurrences":
        select_sql = f"{base_select} {where} GROUP BY Sci_Name ORDER BY COUNT(*) DESC;"
    elif sort_by == "confidence":
        select_sql = f"{base_select} {where} GROUP BY Sci_Name ORDER BY MAX(Confidence) DESC;"
    elif sort_by == "date":
        select_sql = f"{base_select} {where} GROUP BY Sci_Name ORDER BY MIN(Date) DESC, Time DESC;"
    else:
        select_sql = f"{base_select} {where} GROUP BY Sci_Name ORDER BY Com_Name ASC;"

    records = get_records(select_sql, params)
    return records
