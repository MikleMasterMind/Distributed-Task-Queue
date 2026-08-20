import logging

import uvicorn

from app import app
from config import LOG_LEVEL


def main() -> None:
    logging.basicConfig(level=_parse_level(LOG_LEVEL))
    uvicorn.run(app, host="0.0.0.0", port=8000)


def _parse_level(name: str) -> int:
    level = getattr(logging, name.upper(), None)
    if isinstance(level, int):
        return level
    return logging.INFO