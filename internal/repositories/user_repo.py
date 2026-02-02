from internal.models import User


class UserRepository:
    def __init__(self, db):
        self.db = db

    def get_by_username(self, username: str) -> User | None:
        return self.db.query(User).filter(User.username == username).first()

    def get_by_id(self, user_id: str) -> User | None:
        return self.db.query(User).filter(User.id == user_id).first()

    def exists_username(self, username: str) -> bool:
        return self.db.query(User).filter(User.username == username).first() is not None

    def count(self) -> int:
        return self.db.query(User).count()

    def add(self, user: User) -> None:
        self.db.add(user)

    def commit(self) -> None:
        self.db.commit()
