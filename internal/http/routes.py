from internal.http.handlers.auth import register_auth_routes
from internal.http.handlers.health import register_health_routes
from internal.http.handlers.me import register_me_routes
from internal.http.handlers.transactions import register_transaction_routes
from internal.http.handlers.users import register_user_routes


def register_routes(app, cfg, get_db) -> None:
    register_health_routes(app)
    register_auth_routes(app, cfg, get_db)
    register_me_routes(app, cfg, get_db)
    register_user_routes(app, cfg, get_db)
    register_transaction_routes(app, cfg, get_db)
