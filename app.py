import os
import time
import uuid
from collections import deque
from datetime import datetime, timedelta, timezone
from functools import wraps

import bcrypt
import jwt
from dotenv import load_dotenv
from flask import Flask, g, jsonify, request
from flask_cors import CORS
from sqlalchemy import (
    Column,
    DateTime,
    Float,
    ForeignKey,
    String,
    Text,
    create_engine,
    func,
)
from sqlalchemy.orm import declarative_base, relationship, sessionmaker

Base = declarative_base()


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


class User(Base):
    __tablename__ = "users"

    id = Column(String, primary_key=True, default=lambda: str(uuid.uuid4()))
    username = Column(String, unique=True, nullable=False)
    role = Column(String, default="user", nullable=False)
    password_hash = Column(String, nullable=False)
    reset_code_hash = Column(String)
    reset_code_expires_at = Column(DateTime(timezone=True))
    created_at = Column(DateTime(timezone=True), default=lambda: datetime.now(timezone.utc))
    updated_at = Column(DateTime(timezone=True), default=lambda: datetime.now(timezone.utc))

    transactions = relationship("Transaction", back_populates="creator")


class Transaction(Base):
    __tablename__ = "transactions"

    id = Column(String, primary_key=True, default=lambda: str(uuid.uuid4()))
    created_by_user_id = Column(String, ForeignKey("users.id"), nullable=False)
    receiver_type = Column(String, nullable=False)
    receiver_name = Column(String, nullable=False)
    receiver_id = Column(String)
    payer_type = Column(String, nullable=False)
    payer_name = Column(String, nullable=False)
    payer_id = Column(String)
    payment_method = Column(String, nullable=False)
    currency = Column(String, nullable=False)
    amount = Column(Float, nullable=False)
    description = Column(Text)
    datetime_utc = Column(DateTime(timezone=True), nullable=False)
    timezone = Column(String, nullable=False)
    created_at = Column(DateTime(timezone=True), default=lambda: datetime.now(timezone.utc))
    updated_at = Column(DateTime(timezone=True), default=lambda: datetime.now(timezone.utc))

    creator = relationship("User", back_populates="transactions")


class RateLimiter:
    def __init__(self, per_minute: int) -> None:
        self.per_minute = per_minute
        self.hits: dict[str, deque[float]] = {}

    def allow(self, key: str) -> bool:
        now = time.time()
        window_start = now - 60
        queue = self.hits.setdefault(key, deque())
        while queue and queue[0] < window_start:
            queue.popleft()
        if len(queue) >= self.per_minute:
            return False
        queue.append(now)
        return True


def create_app() -> Flask:
    cfg = Config()
    app = Flask(__name__)
    app.config["APP_CONFIG"] = cfg
    CORS(app, origins=cfg.allowed_origins)

    engine = create_engine(cfg.db_url, pool_pre_ping=True)
    SessionLocal = sessionmaker(bind=engine)
    Base.metadata.create_all(engine)

    rate_limiter = RateLimiter(cfg.rate_limit_per_minute)

    def get_db():
        if "db" not in g:
            g.db = SessionLocal()
        return g.db

    @app.before_request
    def before_request() -> None:
        g.request_id = request.headers.get("X-Request-ID", str(uuid.uuid4()))
        if request.path.startswith("/api/v1/auth"):
            key = request.headers.get("X-Forwarded-For", request.remote_addr or "unknown")
            if not rate_limiter.allow(key):
                return error_response(
                    429,
                    "RATE_LIMITED",
                    "too many requests",
                )

    @app.after_request
    def add_request_id(response):
        response.headers["X-Request-ID"] = g.get("request_id", "")
        return response

    @app.teardown_request
    def teardown_request(exception):
        db = g.pop("db", None)
        if db is not None:
            db.close()

    def hash_password(password: str) -> str:
        return bcrypt.hashpw(password.encode("utf-8"), bcrypt.gensalt()).decode("utf-8")

    def verify_password(password: str, password_hash: str) -> bool:
        return bcrypt.checkpw(password.encode("utf-8"), password_hash.encode("utf-8"))

    def generate_token(user: User) -> tuple[str, int]:
        issued_at = datetime.now(timezone.utc)
        expires_in = cfg.jwt_expiry_seconds()
        expires_at = issued_at + timedelta(seconds=expires_in)
        payload = {
            "user_id": user.id,
            "username": user.username,
            "role": user.role,
            "iat": issued_at,
            "exp": expires_at,
            "sub": user.id,
        }
        token = jwt.encode(payload, cfg.jwt_secret, algorithm="HS256")
        return token, expires_in

    def require_auth(fn):
        @wraps(fn)
        def wrapper(*args, **kwargs):
            auth_header = request.headers.get("Authorization", "")
            if not auth_header.startswith("Bearer "):
                return error_response(401, "UNAUTHORIZED", "missing token")
            token = auth_header.split(" ", 1)[1]
            try:
                payload = jwt.decode(token, cfg.jwt_secret, algorithms=["HS256"])
            except jwt.PyJWTError:
                return error_response(401, "UNAUTHORIZED", "invalid token")
            g.user_id = payload.get("user_id")
            g.username = payload.get("username")
            g.role = payload.get("role")
            return fn(*args, **kwargs)

        return wrapper

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

    def seed_users():
        db = get_db()
        seeds = [
            {"username": "admin", "password": "admin123", "role": "admin"},
            {"username": "user1", "password": "1111", "role": "user"},
            {"username": "user2", "password": "2222", "role": "user"},
        ]
        for seed in seeds:
            exists = db.query(User).filter(User.username == seed["username"]).first()
            if exists:
                continue
            user = User(
                username=seed["username"],
                role=seed["role"],
                password_hash=hash_password(seed["password"]),
            )
            db.add(user)
        db.commit()

    @app.get("/healthz")
    def health():
        return jsonify({"ok": True})

    @app.post("/api/v1/auth/login")
    def login():
        data = request.get_json(silent=True) or {}
        validation = validate_required(data, ["username", "password"])
        if validation:
            return validation
        db = get_db()
        user = db.query(User).filter(User.username == data["username"]).first()
        if not user or not verify_password(data["password"], user.password_hash):
            return error_response(401, "UNAUTHORIZED", "invalid credentials")
        token, expires_in = generate_token(user)
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
        user = db.query(User).filter(User.username == data["username"]).first()
        if not user:
            return error_response(404, "NOT_FOUND", "user not found")
        code = "".join(str(int.from_bytes(os.urandom(1), "big") % 10) for _ in range(6))
        user.reset_code_hash = hash_password(code)
        user.reset_code_expires_at = datetime.now(timezone.utc) + timedelta(minutes=10)
        user.updated_at = datetime.now(timezone.utc)
        db.commit()
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
        user = db.query(User).filter(User.username == data["username"]).first()
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
        db.commit()
        return jsonify({"message": "password updated"})

    @app.get("/api/v1/me")
    @require_auth
    def me():
        db = get_db()
        user = db.query(User).filter(User.id == g.user_id).first()
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

    @app.post("/api/v1/users")
    @require_auth
    def create_user():
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
        exists = db.query(User).filter(User.username == data["username"]).first()
        if exists:
            return error_response(409, "CONFLICT", "username already exists")
        user = User(
            username=data["username"],
            role=role,
            password_hash=hash_password(data["password"]),
        )
        db.add(user)
        db.commit()
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

    def parse_datetime_iso(value: str) -> datetime | None:
        try:
            parsed = datetime.fromisoformat(value.replace("Z", "+00:00"))
        except ValueError:
            return None
        if parsed.tzinfo is None:
            parsed = parsed.replace(tzinfo=timezone.utc)
        return parsed.astimezone(timezone.utc)

    def transaction_to_response(tx: Transaction) -> dict:
        return {
            "id": tx.id,
            "created_by_user_id": tx.created_by_user_id,
            "receiver_type": tx.receiver_type,
            "receiver_name": tx.receiver_name,
            "receiver_id": tx.receiver_id,
            "payer_type": tx.payer_type,
            "payer_name": tx.payer_name,
            "payer_id": tx.payer_id,
            "payment_method": tx.payment_method,
            "currency": tx.currency,
            "amount": tx.amount,
            "description": tx.description,
            "datetime_iso": tx.datetime_utc.astimezone(timezone.utc)
            .isoformat()
            .replace("+00:00", "Z"),
            "timezone": tx.timezone,
            "created_at": tx.created_at,
            "updated_at": tx.updated_at,
        }

    @app.post("/api/v1/transactions")
    @require_auth
    def create_transaction():
        data = request.get_json(silent=True) or {}
        required_fields = [
            "receiver_type",
            "receiver_name",
            "payer_type",
            "payer_name",
            "payment_method",
            "currency",
            "amount",
            "datetime_iso",
            "timezone",
        ]
        validation = validate_required(data, required_fields)
        if validation:
            return validation
        if data["receiver_type"] not in {"individual", "legal"}:
            return error_response(400, "VALIDATION_ERROR", "invalid receiver_type")
        if data["payer_type"] not in {"individual", "legal"}:
            return error_response(400, "VALIDATION_ERROR", "invalid payer_type")
        if data["payment_method"] not in {"cash", "account"}:
            return error_response(400, "VALIDATION_ERROR", "invalid payment_method")
        if data["currency"] not in {"IRR", "IRT", "USD", "EUR", "AED", "TRY"}:
            return error_response(400, "VALIDATION_ERROR", "invalid currency")
        try:
            amount = float(data["amount"])
        except (TypeError, ValueError):
            return error_response(400, "VALIDATION_ERROR", "amount must be numeric")
        if amount <= 0:
            return error_response(400, "VALIDATION_ERROR", "amount must be greater than 0")
        parsed_time = parse_datetime_iso(data["datetime_iso"])
        if not parsed_time:
            return error_response(400, "VALIDATION_ERROR", "datetime_iso must be RFC3339")
        db = get_db()
        tx = Transaction(
            created_by_user_id=g.user_id,
            receiver_type=data["receiver_type"],
            receiver_name=data["receiver_name"],
            receiver_id=data.get("receiver_id"),
            payer_type=data["payer_type"],
            payer_name=data["payer_name"],
            payer_id=data.get("payer_id"),
            payment_method=data["payment_method"],
            currency=data["currency"],
            amount=amount,
            description=data.get("description"),
            datetime_utc=parsed_time,
            timezone=data["timezone"],
        )
        db.add(tx)
        db.commit()
        db.refresh(tx)
        return jsonify(transaction_to_response(tx)), 201

    def apply_transaction_filters(query, params):
        search = params.get("search")
        if search:
            query = query.filter(
                Transaction.receiver_name.ilike(f"%{search}%")
                | Transaction.payer_name.ilike(f"%{search}%")
            )
        date_from = params.get("date_from")
        if date_from:
            try:
                parsed = datetime.strptime(date_from, "%Y-%m-%d")
                query = query.filter(Transaction.datetime_utc >= parsed)
            except ValueError:
                raise ValueError("invalid date_from")
        date_to = params.get("date_to")
        if date_to:
            try:
                parsed = datetime.strptime(date_to, "%Y-%m-%d") + timedelta(days=1)
                query = query.filter(Transaction.datetime_utc < parsed)
            except ValueError:
                raise ValueError("invalid date_to")
        currency = params.get("currency")
        if currency:
            query = query.filter(Transaction.currency == currency)
        min_amount = params.get("min_amount")
        if min_amount:
            try:
                amount = float(min_amount)
                query = query.filter(Transaction.amount >= amount)
            except ValueError:
                raise ValueError("invalid min_amount")
        month = params.get("month")
        if month:
            try:
                month_int = int(month)
                if month_int < 1 or month_int > 12:
                    raise ValueError
            except ValueError:
                raise ValueError("invalid month")
            query = query.filter(func.extract("month", Transaction.datetime_utc) == month_int)
        return query

    @app.get("/api/v1/transactions")
    @require_auth
    def list_transactions():
        params = request.args
        db = get_db()
        query = db.query(Transaction).filter(Transaction.created_by_user_id == g.user_id)
        try:
            query = apply_transaction_filters(query, params)
        except ValueError as exc:
            return error_response(400, "VALIDATION_ERROR", str(exc))
        sort_by = params.get("sort_by", "date")
        sort_dir = params.get("sort_dir", "desc").lower()
        sort_map = {
            "receiver": Transaction.receiver_name,
            "payer": Transaction.payer_name,
            "amount": Transaction.amount,
            "currency": Transaction.currency,
            "date": Transaction.datetime_utc,
        }
        sort_column = sort_map.get(sort_by, Transaction.datetime_utc)
        if sort_dir == "desc":
            query = query.order_by(sort_column.desc())
        else:
            query = query.order_by(sort_column.asc())
        page = int(params.get("page", 1))
        per_page = int(params.get("per_page", 10))
        total = query.count()
        items = query.limit(per_page).offset((page - 1) * per_page).all()
        data = [transaction_to_response(tx) for tx in items]
        total_pages = (total + per_page - 1) // per_page
        return jsonify(
            {
                "data": data,
                "meta": {
                    "page": page,
                    "per_page": per_page,
                    "total": total,
                    "total_pages": total_pages,
                },
            }
        )

    @app.get("/api/v1/transactions/summary")
    @require_auth
    def transactions_summary():
        params = request.args
        db = get_db()
        base_query = db.query(Transaction).filter(Transaction.created_by_user_id == g.user_id)
        try:
            base_query = apply_transaction_filters(base_query, params)
        except ValueError as exc:
            return error_response(400, "VALIDATION_ERROR", str(exc))
        total_amount = base_query.with_entities(func.coalesce(func.sum(Transaction.amount), 0.0)).scalar()
        avg_amount = base_query.with_entities(func.coalesce(func.avg(Transaction.amount), 0.0)).scalar()
        count = base_query.count()

        monthly_rows = (
            base_query.with_entities(
                func.to_char(func.date_trunc("month", Transaction.datetime_utc), "MM"),
                func.coalesce(func.sum(Transaction.amount), 0.0),
            )
            .group_by(1)
            .order_by(1)
            .all()
        )
        monthly_map = {month: amount for month, amount in monthly_rows}
        monthly = [
            {"month": f"{i:02d}", "amount": monthly_map.get(f"{i:02d}", 0.0)}
            for i in range(1, 13)
        ]

        currency_rows = (
            base_query.with_entities(Transaction.currency, func.coalesce(func.sum(Transaction.amount), 0.0))
            .group_by(Transaction.currency)
            .all()
        )
        by_currency = []
        for currency, amount in currency_rows:
            percent = (amount / total_amount * 100) if total_amount else 0.0
            by_currency.append({"currency": currency, "amount": amount, "percent": percent})

        return jsonify(
            {
                "kpis": {"total_amount": total_amount, "avg_amount": avg_amount, "count": count},
                "monthly": monthly,
                "by_currency": by_currency,
            }
        )

    @app.get("/api/v1/transactions/<tx_id>")
    @require_auth
    def transaction_by_id(tx_id: str):
        db = get_db()
        tx = (
            db.query(Transaction)
            .filter(Transaction.id == tx_id, Transaction.created_by_user_id == g.user_id)
            .first()
        )
        if not tx:
            return error_response(404, "NOT_FOUND", "transaction not found")
        return jsonify(transaction_to_response(tx))

    @app.delete("/api/v1/transactions/<tx_id>")
    @require_auth
    def delete_transaction(tx_id: str):
        db = get_db()
        tx = (
            db.query(Transaction)
            .filter(Transaction.id == tx_id, Transaction.created_by_user_id == g.user_id)
            .first()
        )
        if not tx:
            return error_response(404, "NOT_FOUND", "transaction not found")
        db.delete(tx)
        db.commit()
        return jsonify({"deleted": True})

    with app.app_context():
        seed_users()

    return app


if __name__ == "__main__":
    config = Config()
    host, port = "0.0.0.0", 8080
    if ":" in config.http_addr:
        _, port_str = config.http_addr.split(":", 1)
        if port_str:
            port = int(port_str)
    app = create_app()
    app.run(host=host, port=port, debug=config.env != "prod")
