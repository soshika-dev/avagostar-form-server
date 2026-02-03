import os
from datetime import datetime, timedelta, timezone

import bcrypt
from internal.models import User
from internal.services import jwt_service


def hash_password(password: str) -> str:
    return bcrypt.hashpw(password.encode("utf-8"), bcrypt.gensalt()).decode("utf-8")


def verify_password(password: str, password_hash: str) -> bool:
    return bcrypt.checkpw(password.encode("utf-8"), password_hash.encode("utf-8"))


def generate_token(user: User, jwt_secret: str, expires_in: int) -> tuple[str, int]:
    issued_at = datetime.now(timezone.utc)
    expires_at = issued_at + timedelta(seconds=expires_in)
    issued_at_ts = int(issued_at.timestamp())
    expires_at_ts = int(expires_at.timestamp())
    payload = {
        "user_id": user.id,
        "username": user.username,
        "role": user.role,
        "iat": issued_at_ts,
        "exp": expires_at_ts,
        "sub": user.id,
    }
    token = jwt_service.encode_token(payload, jwt_secret, algorithm="HS256")
    return token, expires_in


def generate_reset_code() -> str:
    return "".join(str(int.from_bytes(os.urandom(1), "big") % 10) for _ in range(6))
