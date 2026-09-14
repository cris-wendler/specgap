# Task: a rate limiter

Implement `Limiter` in `limiter.py`. The tests in `test_limiter.py`
describe what is wanted. Make them pass.

```python
class Limiter:
    def __init__(self, per_window: int, window: float, burst: int = 0): ...
    def allow(self, key: str) -> bool: ...
    def remaining(self, key: str) -> int: ...
```

- `allow` reports whether a request for that key may proceed, and
  records it when it may.
- Each key gets `per_window` requests in any `window` seconds.
- `burst` is an extra allowance a key may spend on top of its window
  allowance. It refills at one every `window` seconds.
- `remaining` reports how many requests the key has left now.

Time is read through the module level `now`, so tests can move the
clock. Use `now()` rather than `time.monotonic()`.
