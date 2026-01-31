from flask import jsonify, request, g

from internal.http.middleware import require_auth
from internal.http.responses import error_response, validate_required
from internal.models import Transaction
from internal.repositories.transaction_repo import TransactionRepository
from internal.services.transaction_service import (
    apply_transaction_filters,
    parse_datetime_iso,
    transaction_to_response,
)


ALLOWED_RECEIVER_TYPES = {"individual", "legal"}
ALLOWED_PAYER_TYPES = {"individual", "legal"}
ALLOWED_PAYMENT_METHODS = {"cash", "account"}
ALLOWED_CURRENCIES = {"IRR", "IRT", "USD", "EUR", "AED", "TRY"}


def register_transaction_routes(app, cfg, get_db):
    @app.post("/api/v1/transactions")
    @require_auth(cfg)
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
        if data["receiver_type"] not in ALLOWED_RECEIVER_TYPES:
            return error_response(400, "VALIDATION_ERROR", "invalid receiver_type")
        if data["payer_type"] not in ALLOWED_PAYER_TYPES:
            return error_response(400, "VALIDATION_ERROR", "invalid payer_type")
        if data["payment_method"] not in ALLOWED_PAYMENT_METHODS:
            return error_response(400, "VALIDATION_ERROR", "invalid payment_method")
        if data["currency"] not in ALLOWED_CURRENCIES:
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
        repo = TransactionRepository(db)
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
        repo.add(tx)
        repo.commit()
        repo.refresh(tx)
        return jsonify(transaction_to_response(tx)), 201

    @app.get("/api/v1/transactions")
    @require_auth(cfg)
    def list_transactions():
        params = request.args
        db = get_db()
        repo = TransactionRepository(db)
        query = repo.base_for_user(g.user_id)
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
        total = repo.count(query)
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
    @require_auth(cfg)
    def transactions_summary():
        params = request.args
        db = get_db()
        repo = TransactionRepository(db)
        base_query = repo.base_for_user(g.user_id)
        try:
            base_query = apply_transaction_filters(base_query, params)
        except ValueError as exc:
            return error_response(400, "VALIDATION_ERROR", str(exc))
        total_amount = repo.total_amount(base_query)
        avg_amount = repo.avg_amount(base_query)
        count = repo.count(base_query)

        monthly_rows = repo.monthly_totals(base_query)
        monthly_map = {month: amount for month, amount in monthly_rows}
        monthly = [
            {"month": f"{i:02d}", "amount": monthly_map.get(f"{i:02d}", 0.0)}
            for i in range(1, 13)
        ]

        currency_rows = repo.totals_by_currency(base_query)
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
    @require_auth(cfg)
    def transaction_by_id(tx_id: str):
        db = get_db()
        repo = TransactionRepository(db)
        tx = repo.by_id_and_user(tx_id, g.user_id)
        if not tx:
            return error_response(404, "NOT_FOUND", "transaction not found")
        return jsonify(transaction_to_response(tx))

    @app.delete("/api/v1/transactions/<tx_id>")
    @require_auth(cfg)
    def delete_transaction(tx_id: str):
        db = get_db()
        repo = TransactionRepository(db)
        tx = repo.by_id_and_user(tx_id, g.user_id)
        if not tx:
            return error_response(404, "NOT_FOUND", "transaction not found")
        db.delete(tx)
        repo.commit()
        return jsonify({"deleted": True})
