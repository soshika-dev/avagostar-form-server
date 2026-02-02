from flask import jsonify, request

from internal.http.middleware import require_auth
from internal.http.responses import error_response, validate_required
from internal.repositories.user_repo import UserRepository
from internal.services.user_service import create_user


def register_user_routes(app, cfg, get_db):
    @app.post("/api/v1/users/bootstrap")
    def bootstrap_user_handler():
        data = request.get_json(silent=True) or {}
        validation = validate_required(data, ["username", "password"])
        if validation:
            return validation
        if len(data["password"]) < cfg.password_min_len:
            return error_response(
                400,
                "VALIDATION_ERROR",
                f"password must be at least {cfg.password_min_len} characters",
            )
        db = get_db()
        repo = UserRepository(db)
        if repo.count() > 0:
            return error_response(409, "CONFLICT", "users already exist")
        if repo.exists_username(data["username"]):
            return error_response(409, "CONFLICT", "username already exists")
        role = (data.get("role") or "admin").strip() or "admin"
        user = create_user(repo, data["username"], role, data["password"])
        return (
            jsonify(
                {
                    "id": user.id,
                    "username": user.username,
                    "role": user.role,
                    "created_at": user.created_at,
                }
            ),
            201,
        )

    @app.post("/api/v1/users")
    @require_auth(cfg)
    def create_user_handler():
        data = request.get_json(silent=True) or {}
        validation = validate_required(data, ["username", "password"])
        if validation:
            return validation
        role = (data.get("role") or "user").strip() or "user"
        if len(data["password"]) < cfg.password_min_len:
            return error_response(
                400,
                "VALIDATION_ERROR",
                f"password must be at least {cfg.password_min_len} characters",
            )
        db = get_db()
        repo = UserRepository(db)
        if repo.exists_username(data["username"]):
            return error_response(409, "CONFLICT", "username already exists")
        user = create_user(repo, data["username"], role, data["password"])
        return (
            jsonify(
                {
                    "id": user.id,
                    "username": user.username,
                    "role": user.role,
                    "created_at": user.created_at,
                }
            ),
            201,
        )
