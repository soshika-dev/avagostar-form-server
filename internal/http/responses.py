from flask import jsonify


def error_response(status: int, code: str, message: str, details=None):
    payload = {"error": {"code": code, "message": message}}
    if details is not None:
        payload["error"]["details"] = details
    return jsonify(payload), status


def validate_required(data, fields):
    missing = [field for field in fields if not data.get(field)]
    if missing:
        return error_response(400, "VALIDATION_ERROR", "invalid request", missing)
    return None
