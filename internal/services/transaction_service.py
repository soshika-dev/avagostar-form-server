from datetime import datetime, timedelta, timezone

from sqlalchemy import func

from internal.models import Transaction


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
        except ValueError as exc:
            raise ValueError("invalid date_from") from exc
    date_to = params.get("date_to")
    if date_to:
        try:
            parsed = datetime.strptime(date_to, "%Y-%m-%d") + timedelta(days=1)
            query = query.filter(Transaction.datetime_utc < parsed)
        except ValueError as exc:
            raise ValueError("invalid date_to") from exc
    currency = params.get("currency")
    if currency:
        query = query.filter(Transaction.currency == currency)
    min_amount = params.get("min_amount")
    if min_amount:
        try:
            amount = float(min_amount)
            query = query.filter(Transaction.amount >= amount)
        except ValueError as exc:
            raise ValueError("invalid min_amount") from exc
    month = params.get("month")
    if month:
        try:
            month_int = int(month)
            if month_int < 1 or month_int > 12:
                raise ValueError
        except ValueError as exc:
            raise ValueError("invalid month") from exc
        query = query.filter(func.extract("month", Transaction.datetime_utc) == month_int)
    return query
