"""LLM router - Part 2 stub.

This router provides placeholder endpoints for the local LLM feature
that will be implemented in Part 2. Currently returns 501 Not Implemented.
"""

from fastapi import APIRouter, HTTPException
from pydantic import BaseModel, Field
from typing import Optional

router = APIRouter()


class LLMQueryRequest(BaseModel):
    """Request body for LLM queries."""

    question: str = Field(..., description="Question to ask the LLM")
    context: Optional[str] = Field(
        None,
        description="Optional context about birds or detections",
    )
    max_tokens: Optional[int] = Field(
        512,
        ge=1,
        le=2048,
        description="Maximum tokens in response",
    )
    temperature: Optional[float] = Field(
        0.7,
        ge=0.0,
        le=2.0,
        description="Sampling temperature",
    )


class LLMQueryResponse(BaseModel):
    """Response for LLM queries."""

    question: str
    answer: str
    model_name: str
    tokens_used: int


@router.post("/ask", response_model=LLMQueryResponse)
async def ask_llm(request: LLMQueryRequest):
    """Ask a question to the local LLM.

    Part 2 feature - not yet implemented.

    Args:
        request: LLM query request with question and optional context.

    Raises:
        HTTPException: 501 Not Implemented.
    """
    raise HTTPException(
        status_code=501,
        detail="LLM feature not implemented yet. Coming in Part 2.",
    )


@router.get("/status")
async def llm_status():
    """Get LLM model status.

    Returns:
        LLM feature status (currently disabled).
    """
    return {
        "enabled": False,
        "loaded": False,
        "model_name": "TinyLlama-1.1B",
        "status": "not_implemented",
        "memory_bytes": 0,
        "description": "Local LLM for bird information - Coming in Part 2",
        "supported_models": [
            "tinyllama-1.1b",
            "qwen2.5-0.5b",
            "phi-3-mini",
        ],
    }


@router.post("/load")
async def load_llm(model_name: Optional[str] = None):
    """Load the LLM model.

    Part 2 feature - not yet implemented.

    Args:
        model_name: Optional model name to load.

    Raises:
        HTTPException: 501 Not Implemented.
    """
    raise HTTPException(
        status_code=501,
        detail="LLM feature not implemented yet. Coming in Part 2.",
    )


@router.post("/unload")
async def unload_llm():
    """Unload the LLM model to free memory.

    Part 2 feature - not yet implemented.

    Raises:
        HTTPException: 501 Not Implemented.
    """
    raise HTTPException(
        status_code=501,
        detail="LLM feature not implemented yet. Coming in Part 2.",
    )


@router.get("/models")
async def list_llm_models():
    """List available LLM models.

    Returns:
        List of supported models for Part 2.
    """
    return {
        "available_models": [
            {
                "name": "tinyllama-1.1b",
                "description": "TinyLlama 1.1B - Compact general-purpose LLM",
                "memory_mb": 1024,
                "recommended": True,
            },
            {
                "name": "qwen2.5-0.5b",
                "description": "Qwen 2.5 0.5B - Very compact, fast responses",
                "memory_mb": 512,
                "recommended": False,
            },
            {
                "name": "phi-3-mini",
                "description": "Phi-3 Mini - Microsoft's efficient model",
                "memory_mb": 1500,
                "recommended": False,
            },
        ],
        "status": "not_implemented",
    }
