import time
from collections import deque


class RateLimiter:
    def __init__(self, per_minute: int) -> None:
        self.per_minute = per_minute
        self.hits: dict[str, deque[float]] = {}

    def allow(self, key: str) -> bool:
        now = time.time()
        window_start = now - 60
        queue = self.hits.setdefault(key, deque())
        while queue and queue[0] < window_start:
            queue.popleft()
        if len(queue) >= self.per_minute:
            return False
        queue.append(now)
        return True
