import uuid
from datetime import datetime, timezone

from sqlalchemy import Column, DateTime, Float, ForeignKey, String, Text
from sqlalchemy.orm import relationship

from internal.db import Base


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
