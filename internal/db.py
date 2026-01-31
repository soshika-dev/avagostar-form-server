from sqlalchemy import create_engine
from sqlalchemy.orm import declarative_base, sessionmaker

Base = declarative_base()


def init_engine(database_url: str):
    return create_engine(database_url, pool_pre_ping=True)


def init_session(engine):
    return sessionmaker(bind=engine)


def init_db(engine) -> None:
    Base.metadata.create_all(engine)
