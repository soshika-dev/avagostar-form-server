from sqlalchemy import func

from internal.models import Transaction


class TransactionRepository:
    def __init__(self, db):
        self.db = db

    def add(self, tx: Transaction) -> None:
        self.db.add(tx)

    def commit(self) -> None:
        self.db.commit()

    def refresh(self, tx: Transaction) -> None:
        self.db.refresh(tx)

    def by_id_and_user(self, tx_id: str, user_id: str) -> Transaction | None:
        return (
            self.db.query(Transaction)
            .filter(Transaction.id == tx_id, Transaction.created_by_user_id == user_id)
            .first()
        )

    def base_for_user(self, user_id: str):
        return self.db.query(Transaction).filter(Transaction.created_by_user_id == user_id)

    def total_amount(self, query):
        return query.with_entities(func.coalesce(func.sum(Transaction.amount), 0.0)).scalar()

    def avg_amount(self, query):
        return query.with_entities(func.coalesce(func.avg(Transaction.amount), 0.0)).scalar()

    def count(self, query):
        return query.count()

    def monthly_totals(self, query):
        return (
            query.with_entities(
                func.to_char(func.date_trunc("month", Transaction.datetime_utc), "MM"),
                func.coalesce(func.sum(Transaction.amount), 0.0),
            )
            .group_by(1)
            .order_by(1)
            .all()
        )

    def totals_by_currency(self, query):
        return (
            query.with_entities(Transaction.currency, func.coalesce(func.sum(Transaction.amount), 0.0))
            .group_by(Transaction.currency)
            .all()
        )
