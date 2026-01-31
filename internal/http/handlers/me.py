from flask import jsonify, g

from internal.http.middleware import require_auth
from internal.http.responses import error_response
from internal.repositories.user_repo import UserRepository


def register_me_routes(app, cfg, get_db):
    @app.get("/api/v1/me")
    @require_auth(cfg)
    def me():
        db = get_db()
        repo = UserRepository(db)
        user = repo.get_by_id(g.user_id)
        if not user:
            return error_response(404, "NOT_FOUND", "user not found")
        return jsonify(
            {
                "id": user.id,
                "username": user.username,
                "role": user.role,
                "created_at": user.created_at,
            }
        )
