from datetime import datetime, timezone

from internal.models import User
from internal.repositories.user_repo import UserRepository
from internal.services.auth_service import hash_password


def create_user(repo: UserRepository, username: str, role: str, password: str) -> User:
    user = User(
        username=username,
        role=role,
        password_hash=hash_password(password),
        created_at=datetime.now(timezone.utc),
        updated_at=datetime.now(timezone.utc),
    )
    repo.add(user)
    repo.commit()
    return user


def seed_users(repo: UserRepository) -> None:
    seeds = [
        {"username": "qamari", "password": "Qamari@123$", "role": "admin"},
        {"username": "sedaqati", "password": "Sedaqati@123$", "role": "user"},
    ]
    for seed in seeds:
        if repo.exists_username(seed["username"]):
            continue
        user = User(
            username=seed["username"],
            role=seed["role"],
            password_hash=hash_password(seed["password"]),
            created_at=datetime.now(timezone.utc),
            updated_at=datetime.now(timezone.utc),
        )
        repo.add(user)
    repo.commit()
