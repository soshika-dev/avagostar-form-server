import os

from dotenv import load_dotenv


class Config:
    def __init__(self) -> None:
        load_dotenv()
        self.env = os.getenv("ENV", "dev")
        self.http_addr = os.getenv("HTTP_ADDR", ":8080")
        self.db_url = os.getenv(
            "DATABASE_URL",
            "postgresql+psycopg2://app:app@localhost:5432/avagostar",
        )
        self.jwt_secret = os.getenv("JWT_SECRET", "change-me")
        self.jwt_expires_in = os.getenv("JWT_EXPIRES_IN", "1h")
        self.allowed_origins = [
            origin.strip()
            for origin in os.getenv("CORS_ALLOWED_ORIGINS", "http://localhost:5173").split(",")
            if origin.strip()
        ]
        self.rate_limit_per_minute = int(os.getenv("RATE_LIMIT_PER_MIN", "30"))
        self.request_timeout = os.getenv("REQUEST_TIMEOUT", "5s")
        self.password_min_len = 8 if self.env == "prod" else 4
        self.enable_dev_reset_codes = self.env != "prod"

        if not self.jwt_secret:
            raise ValueError("JWT_SECRET is required")

    def jwt_expiry_seconds(self) -> int:
        raw = self.jwt_expires_in
        if raw.endswith("h"):
            return int(float(raw[:-1]) * 3600)
        if raw.endswith("m"):
            return int(float(raw[:-1]) * 60)
        if raw.endswith("s"):
            return int(float(raw[:-1]))
        return int(raw)
