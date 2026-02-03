from __future__ import annotations

from typing import Any, Iterable

try:
    import jwt as pyjwt

    if not hasattr(pyjwt, "encode") or not hasattr(pyjwt, "decode"):
        raise ImportError("PyJWT not available")

    JwtError = pyjwt.PyJWTError

    def encode_token(payload: dict[str, Any], secret: str, algorithm: str = "HS256") -> str:
        return pyjwt.encode(payload, secret, algorithm=algorithm)

    def decode_token(token: str, secret: str, algorithms: Iterable[str]) -> dict[str, Any]:
        return pyjwt.decode(token, secret, algorithms=list(algorithms))

except Exception:  # pragma: no cover - fallback for environments without PyJWT
    import base64

    from jwt import JWT, jwk_from_dict

    class JwtError(Exception):
        pass

    _jwt = JWT()

    def _normalize_secret(secret: str) -> str:
        secret_bytes = secret.encode("utf-8")
        return base64.urlsafe_b64encode(secret_bytes).decode("utf-8").rstrip("=")

    def _jwk_from_secret(secret: str) -> dict[str, Any]:
        return jwk_from_dict({"k": _normalize_secret(secret), "kty": "oct"})

    def encode_token(payload: dict[str, Any], secret: str, algorithm: str = "HS256") -> str:
        return _jwt.encode(payload, _jwk_from_secret(secret), alg=algorithm)

    def decode_token(token: str, secret: str, algorithms: Iterable[str]) -> dict[str, Any]:
        try:
            return _jwt.decode(
                token,
                _jwk_from_secret(secret),
                do_verify=True,
                algorithms=list(algorithms),
            )
        except Exception as exc:
            raise JwtError(str(exc)) from exc
