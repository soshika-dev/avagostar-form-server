from flask import Flask, g
from flask_cors import CORS

from internal.config import Config
from internal.db import init_db, init_engine, init_session
from internal.http.middleware import register_middleware
from internal.http.routes import register_routes
from internal.rate_limiter import RateLimiter
from internal.repositories.user_repo import UserRepository
from internal.services.user_service import seed_users


def create_app() -> Flask:
    cfg = Config()
    app = Flask(__name__)
    app.config["APP_CONFIG"] = cfg
    CORS(app, origins=cfg.allowed_origins)

    engine = init_engine(cfg.db_url)
    SessionLocal = init_session(engine)
    init_db(engine)

    rate_limiter = RateLimiter(cfg.rate_limit_per_minute)

    def get_db():
        if "db" not in g:
            g.db = SessionLocal()
        return g.db

    register_middleware(app, get_db, rate_limiter, cfg)
    register_routes(app, cfg, get_db)

    with app.app_context():
        db = get_db()
        repo = UserRepository(db)
        seed_users(repo)
        db.close()

    return app
