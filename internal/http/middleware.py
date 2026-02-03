import uuid
from functools import wraps

from flask import g, request

from internal.http.responses import error_response
from internal.services import jwt_service


def register_middleware(app, get_db, rate_limiter, cfg) -> None:
    @app.before_request
    def before_request() -> None:
        g.request_id = request.headers.get("X-Request-ID", str(uuid.uuid4()))
        if request.path.startswith("/api/v1/auth"):
            key = request.headers.get("X-Forwarded-For", request.remote_addr or "unknown")
            if not rate_limiter.allow(key):
                return error_response(429, "RATE_LIMITED", "too many requests")

    @app.after_request
    def add_request_id(response):
        response.headers["X-Request-ID"] = g.get("request_id", "")
        return response

    @app.teardown_request
    def teardown_request(exception):
        db = g.pop("db", None)
        if db is not None:
            db.close()


def require_auth(cfg):
    def decorator(fn):
        @wraps(fn)
        def wrapper(*args, **kwargs):
            auth_header = request.headers.get("Authorization", "")
            if not auth_header.startswith("Bearer "):
                return error_response(401, "UNAUTHORIZED", "missing token")
            token = auth_header.split(" ", 1)[1]
            try:
                payload = jwt_service.decode_token(token, cfg.jwt_secret, algorithms=["HS256"])
            except jwt_service.JwtError:
                return error_response(401, "UNAUTHORIZED", "invalid token")
            g.user_id = payload.get("user_id")
            g.username = payload.get("username")
            g.role = payload.get("role")
            return fn(*args, **kwargs)

        return wrapper

    return decorator
