"""BirdNET model manager.

Wraps the existing birdnet.models module with the ModelManager interface
for lifecycle management and memory tracking.
"""

import logging
from typing import Any, Optional

from .base import ModelManager

log = logging.getLogger(__name__)

# Approximate memory usage by model type (in bytes)
MODEL_MEMORY_ESTIMATES = {
    "BirdNET_GLOBAL_6K_V2.4_Model_FP16": 500 * 1024 * 1024,  # ~500MB
    "BirdNET_6K_GLOBAL_MODEL": 400 * 1024 * 1024,  # ~400MB
    "Perch_v2": 300 * 1024 * 1024,  # ~300MB
    "BirdNET-Go_classifier_20250916": 500 * 1024 * 1024,  # ~500MB
}

DEFAULT_MEMORY_ESTIMATE = 500 * 1024 * 1024  # 500MB default


class BirdNETManager(ModelManager):
    """Model manager for BirdNET bird detection models.

    Wraps the existing birdnet.models.get_model() function with lifecycle
    management, thread safety, and memory tracking.

    Example:
        manager = BirdNETManager()
        manager.load()
        model = manager.get_model()
        predictions = model.predict(audio_chunk)
    """

    def __init__(self, model_name: Optional[str] = None):
        """Initialize the BirdNET model manager.

        Args:
            model_name: Optional model name override. If None, uses config.
        """
        super().__init__(name="BirdNET")
        self._model_name = model_name
        self._actual_model_name: Optional[str] = None

    def _load_model(self) -> Any:
        """Load the BirdNET model using the existing birdnet.models module.

        Returns:
            Loaded BirdNET model instance.
        """
        # Import here to avoid circular imports and defer heavy imports
        from birdnet.models import get_model
        from birdnet.config import get_settings

        # Determine which model to load
        if self._model_name is None:
            conf = get_settings()
            self._actual_model_name = conf.get("MODEL", "BirdNET_GLOBAL_6K_V2.4_Model_FP16")
        else:
            self._actual_model_name = self._model_name

        log.info("Loading BirdNET model: %s", self._actual_model_name)
        return get_model(self._actual_model_name)

    def _unload_model(self) -> None:
        """Unload the BirdNET model.

        The TFLite interpreter doesn't require explicit cleanup,
        but we clear our reference to allow garbage collection.
        """
        self._actual_model_name = None

    def memory_usage(self) -> int:
        """Return approximate memory usage in bytes.

        Returns:
            Estimated memory usage based on model type.
        """
        if not self.is_loaded():
            return 0

        return MODEL_MEMORY_ESTIMATES.get(
            self._actual_model_name or "", DEFAULT_MEMORY_ESTIMATE
        )

    def get_model_name(self) -> Optional[str]:
        """Get the name of the currently loaded model.

        Returns:
            Model name string, or None if not loaded.
        """
        return self._actual_model_name

    def predict(self, audio_chunk) -> list:
        """Run inference on an audio chunk.

        Args:
            audio_chunk: Audio data as numpy array.

        Returns:
            List of (species_name, confidence) tuples, sorted by confidence.
        """
        model = self.get_model()
        self.increment_inference_count()
        return model.predict(audio_chunk)

    def set_metadata(self, lat: float, lon: float, week: int) -> None:
        """Set location and week metadata for species filtering.

        Args:
            lat: Latitude coordinate.
            lon: Longitude coordinate.
            week: ISO week number (1-52).
        """
        model = self.get_model()
        model.set_meta_data(lat, lon, week)

    def get_species_list(self) -> list:
        """Get the filtered species list based on current metadata.

        Returns:
            List of species names filtered by location/week.
        """
        model = self.get_model()
        return model.get_species_list()

    def get_status(self) -> dict:
        """Get comprehensive status information.

        Returns:
            Dictionary with model status including BirdNET-specific info.
        """
        status = super().get_status()
        status["model_name"] = self._actual_model_name
        return status


# Singleton instance for use throughout the service
birdnet_manager = BirdNETManager()
