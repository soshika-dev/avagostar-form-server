from datetime import datetime, timedelta, timezone
import json
from urllib import error, request

from sqlalchemy import func

from internal.models import Transaction


USD_EXCHANGE_RATES = {
    "USD": 1.0,
    "EUR": 1.17,
    "AED": 0.2723,
    "TRY": 0.031,
    "IRR": 0.000012,
    "IRT": 0.00012,
}

BASE_NORMALIZATION_CURRENCY = "AED"
EXCHANGE_RATES_API_URL = "https://open.er-api.com/v6/latest/USD"


def set_usd_exchange_rates(custom_rates: dict[str, float] | None) -> None:
    if not custom_rates:
        return
    USD_EXCHANGE_RATES.update(custom_rates)


def fetch_usd_exchange_rates() -> dict[str, float]:
    try:
        with request.urlopen(EXCHANGE_RATES_API_URL, timeout=5) as response:
            payload = json.loads(response.read().decode("utf-8"))
    except (error.URLError, TimeoutError, json.JSONDecodeError):
        return {}

    rates = payload.get("rates")
    if not isinstance(rates, dict):
        return {}

    parsed = {}
    for currency, rate in rates.items():
        try:
            parsed[str(currency).upper()] = float(rate)
        except (TypeError, ValueError):
            continue
    return parsed


def parse_datetime_iso(value: str) -> datetime | None:
    try:
        parsed = datetime.fromisoformat(value.replace("Z", "+00:00"))
    except ValueError:
        return None
    if parsed.tzinfo is None:
        parsed = parsed.replace(tzinfo=timezone.utc)
    return parsed.astimezone(timezone.utc)


def amount_in_usd(amount: float, currency: str) -> float:
    rate = USD_EXCHANGE_RATES.get(currency)
    if not rate:
        return amount
    return amount * rate


def amount_in_aed(amount: float, currency: str) -> float:
    amount_usd = amount_in_usd(amount, currency)
    aed_rate = USD_EXCHANGE_RATES.get(BASE_NORMALIZATION_CURRENCY)
    if not aed_rate:
        return amount_usd
    return amount_usd / aed_rate


def transaction_to_response(tx: Transaction, normalize_currency: bool = False) -> dict:
    amount = tx.amount
    currency = tx.currency
    response = {
        "id": tx.id,
        "created_by_user_id": tx.created_by_user_id,
        "receiver_type": tx.receiver_type,
        "receiver_name": tx.receiver_name,
        "receiver_id": tx.receiver_id,
        "payer_type": tx.payer_type,
        "payer_name": tx.payer_name,
        "payer_id": tx.payer_id,
        "payment_method": tx.payment_method,
        "currency": currency,
        "amount": amount,
        "description": tx.description,
        "datetime_iso": tx.datetime_utc.astimezone(timezone.utc)
        .isoformat()
        .replace("+00:00", "Z"),
        "timezone": tx.timezone,
        "created_at": tx.created_at,
        "updated_at": tx.updated_at,
    }

    if normalize_currency:
        response["original_currency"] = currency
        response["original_amount"] = amount
        response["currency"] = BASE_NORMALIZATION_CURRENCY
        response["amount"] = amount_in_aed(amount, currency)

    return response


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
