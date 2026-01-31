from flask import jsonify


def register_health_routes(app):
    @app.get("/healthz")
    def health():
        return jsonify({"ok": True})
