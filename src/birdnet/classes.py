"""Data classes for BirdNET-Pi detections and file parsing."""
import datetime
import os
import re
from typing import Optional

from tzlocal import get_localzone


class Detection:
    """Represents a single bird detection with timestamp and confidence."""

    def __init__(
        self,
        file_date: datetime.datetime,
        start_time: float | str,
        stop_time: float | str,
        scientific_name: str,
        common_name: str,
        confidence: float | str,
    ) -> None:
        self.start: float = float(start_time)
        self.stop: float = float(stop_time)
        self.datetime: datetime.datetime = file_date + datetime.timedelta(seconds=self.start)
        self.date: str = self.datetime.strftime("%Y-%m-%d")
        self.time: str = self.datetime.strftime("%H:%M:%S")
        self.iso8601: str = self.datetime.astimezone(get_localzone()).isoformat()
        self.week: int = self.datetime.isocalendar()[1]
        self.confidence: float = round(float(confidence), 4)
        self.confidence_pct: int = round(self.confidence * 100)
        self.species: str = scientific_name
        self.scientific_name: str = scientific_name
        self.common_name: str = common_name
        self.common_name_safe: str = self.common_name.replace("'", "").replace(" ", "_")
        self.file_name_extr: Optional[str] = None

    def __str__(self) -> str:
        return f'Detection({self.species}, {self.common_name}, {self.confidence}, {self.iso8601})'


class ParseFileName:
    """Parses recording filename to extract date, time, and stream ID."""

    def __init__(self, file_name: str) -> None:
        self.file_name: str = file_name
        name = os.path.splitext(os.path.basename(file_name))[0]
        date_created = re.search('^[0-9]+-[0-9]+-[0-9]+', name).group()
        time_created = re.search('[0-9]+:[0-9]+:[0-9]+$', name).group()
        self.file_date: datetime.datetime = datetime.datetime.strptime(
            f'{date_created}T{time_created}', "%Y-%m-%dT%H:%M:%S"
        )
        self.root: str = name

        ident_match = re.search("RTSP_[0-9]+-", file_name)
        self.RTSP_id: str = ident_match.group() if ident_match is not None else ""

    @property
    def iso8601(self) -> str:
        current_iso8601 = self.file_date.astimezone(get_localzone()).isoformat()
        return current_iso8601

    @property
    def week(self) -> int:
        week = self.file_date.isocalendar()[1]
        return week
