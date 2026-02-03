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
    from jwt import JWT, jwk_from_dict

    class JwtError(Exception):
        pass

    _jwt = JWT()

    def _jwk_from_secret(secret: str) -> dict[str, Any]:
        return jwk_from_dict({"k": secret, "kty": "oct"})

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
