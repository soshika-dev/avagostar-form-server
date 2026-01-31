from datetime import datetime, timedelta, timezone

from flask import jsonify, request

from internal.http.responses import error_response, validate_required
from internal.repositories.user_repo import UserRepository
from internal.services.auth_service import (
    generate_reset_code,
    generate_token,
    hash_password,
    verify_password,
)


def register_auth_routes(app, cfg, get_db):
    @app.post("/api/v1/auth/login")
    def login():
        data = request.get_json(silent=True) or {}
        validation = validate_required(data, ["username", "password"])
        if validation:
            return validation
        db = get_db()
        repo = UserRepository(db)
        user = repo.get_by_username(data["username"])
        if not user or not verify_password(data["password"], user.password_hash):
            return error_response(401, "UNAUTHORIZED", "invalid credentials")
        token, expires_in = generate_token(user, cfg.jwt_secret, cfg.jwt_expiry_seconds())
        return jsonify(
            {
                "access_token": token,
                "token_type": "Bearer",
                "expires_in": expires_in,
                "user": {"id": user.id, "username": user.username, "role": user.role},
            }
        )

    @app.post("/api/v1/auth/forgot")
    def forgot_password():
        data = request.get_json(silent=True) or {}
        validation = validate_required(data, ["username"])
        if validation:
            return validation
        db = get_db()
        repo = UserRepository(db)
        user = repo.get_by_username(data["username"])
        if not user:
            return error_response(404, "NOT_FOUND", "user not found")
        code = generate_reset_code()
        user.reset_code_hash = hash_password(code)
        user.reset_code_expires_at = datetime.now(timezone.utc) + timedelta(minutes=10)
        user.updated_at = datetime.now(timezone.utc)
        repo.commit()
        resp = {"message": "reset code sent"}
        if cfg.enable_dev_reset_codes:
            resp["code"] = code
        return jsonify(resp)

    @app.post("/api/v1/auth/reset")
    def reset_password():
        data = request.get_json(silent=True) or {}
        validation = validate_required(data, ["username", "code", "new_password"])
        if validation:
            return validation
        if len(data["new_password"]) < cfg.password_min_len:
            return error_response(
                400,
                "VALIDATION_ERROR",
                f"password must be at least {cfg.password_min_len} characters",
            )
        db = get_db()
        repo = UserRepository(db)
        user = repo.get_by_username(data["username"])
        if not user:
            return error_response(404, "NOT_FOUND", "user not found")
        if not user.reset_code_hash or not user.reset_code_expires_at:
            return error_response(400, "VALIDATION_ERROR", "reset code not requested")
        if datetime.now(timezone.utc) > user.reset_code_expires_at:
            return error_response(400, "VALIDATION_ERROR", "reset code expired")
        if not verify_password(data["code"], user.reset_code_hash):
            return error_response(400, "VALIDATION_ERROR", "invalid reset code")
        user.password_hash = hash_password(data["new_password"])
        user.reset_code_hash = None
        user.reset_code_expires_at = None
        user.updated_at = datetime.now(timezone.utc)
        repo.commit()
        return jsonify({"message": "password updated"})
