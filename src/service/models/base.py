"""Base class for ML model lifecycle management.

Provides a common interface for loading, unloading, and managing ML models
with thread-safe operations and memory tracking.
"""

import gc
import logging
import threading
import time
from abc import ABC, abstractmethod
from typing import Any, Optional

log = logging.getLogger(__name__)


class ModelManager(ABC):
    """Abstract base class for ML model lifecycle management.

    Provides thread-safe model loading/unloading with memory tracking.
    Subclasses must implement _load_model(), _unload_model(), and memory_usage().

    Example:
        class MyModelManager(ModelManager):
            def _load_model(self):
                return load_my_model()

            def _unload_model(self):
                pass  # Cleanup handled by gc

            def memory_usage(self) -> int:
                return 500 * 1024 * 1024  # 500MB estimate
    """

    def __init__(self, name: str = "model"):
        """Initialize the model manager.

        Args:
            name: Human-readable name for logging purposes.
        """
        self._name = name
        self._model: Optional[Any] = None
        self._lock = threading.Lock()
        self._load_time: Optional[float] = None
        self._inference_count: int = 0

    @property
    def name(self) -> str:
        """Get the model name."""
        return self._name

    @abstractmethod
    def _load_model(self) -> Any:
        """Load the model. Implement in subclass.

        Returns:
            The loaded model object.
        """
        pass

    @abstractmethod
    def _unload_model(self) -> None:
        """Unload the model. Implement in subclass.

        Called before setting _model to None. Use for cleanup.
        """
        pass

    @abstractmethod
    def memory_usage(self) -> int:
        """Return approximate memory usage in bytes.

        Returns:
            Memory usage in bytes, or 0 if model is not loaded.
        """
        pass

    def load(self) -> None:
        """Load the model if not already loaded.

        Thread-safe. Only one thread will load the model.
        """
        with self._lock:
            if self._model is None:
                log.info("Loading model: %s", self._name)
                start = time.time()
                self._model = self._load_model()
                self._load_time = time.time()
                elapsed = self._load_time - start
                log.info("Model %s loaded in %.2fs", self._name, elapsed)

    def unload(self) -> None:
        """Unload the model if loaded.

        Thread-safe. Triggers garbage collection after unloading.
        """
        with self._lock:
            if self._model is not None:
                log.info("Unloading model: %s", self._name)
                self._unload_model()
                self._model = None
                self._load_time = None
                gc.collect()
                log.info("Model %s unloaded", self._name)

    def is_loaded(self) -> bool:
        """Check if the model is currently loaded.

        Returns:
            True if model is loaded, False otherwise.
        """
        return self._model is not None

    def get_model(self) -> Any:
        """Get the model, loading it if necessary.

        Returns:
            The loaded model object.
        """
        if not self.is_loaded():
            self.load()
        return self._model

    def get_load_time(self) -> Optional[float]:
        """Get the timestamp when the model was loaded.

        Returns:
            Unix timestamp of load time, or None if not loaded.
        """
        return self._load_time

    def get_uptime(self) -> Optional[float]:
        """Get how long the model has been loaded in seconds.

        Returns:
            Seconds since model was loaded, or None if not loaded.
        """
        if self._load_time is None:
            return None
        return time.time() - self._load_time

    def increment_inference_count(self) -> None:
        """Increment the inference counter."""
        self._inference_count += 1

    def get_inference_count(self) -> int:
        """Get the total number of inferences performed.

        Returns:
            Number of inferences since model was loaded.
        """
        return self._inference_count

    def get_status(self) -> dict:
        """Get comprehensive status information about the model.

        Returns:
            Dictionary with model status information.
        """
        return {
            "name": self._name,
            "loaded": self.is_loaded(),
            "memory_bytes": self.memory_usage(),
            "load_time": self._load_time,
            "uptime_seconds": self.get_uptime(),
            "inference_count": self._inference_count,
        }
