"""User management service."""
from dataclasses import dataclass
from typing import Optional

import bcrypt
import sqlalchemy as sa
from sqlalchemy.orm import Session


@dataclass
class UserDTO:
    id: int
    email: str
    is_active: bool


class UserService:
    """CRUD operations for user accounts."""

    def __init__(self, session: Session):
        self._session = session

    def get_by_id(self, user_id: int) -> Optional[UserDTO]:
        """Return a UserDTO for the given id, or None if not found."""
        row = self._session.execute(
            sa.text("SELECT id, email, is_active FROM users WHERE id = :id"),
            {"id": user_id},
        ).fetchone()
        if row is None:
            return None
        return UserDTO(id=row.id, email=row.email, is_active=row.is_active)

    def create(self, email: str, password: str) -> UserDTO:
        """Hash the password and persist a new user row."""
        hashed = bcrypt.hashpw(password.encode(), bcrypt.gensalt())
        result = self._session.execute(
            sa.text("INSERT INTO users (email, password_hash, is_active) VALUES (:email, :hash, true) RETURNING id"),
            {"email": email, "hash": hashed.decode()},
        )
        self._session.commit()
        return UserDTO(id=result.fetchone().id, email=email, is_active=True)

    def deactivate(self, user_id: int) -> bool:
        """Set is_active=false for the given user. Returns True if the row existed."""
        result = self._session.execute(
            sa.text("UPDATE users SET is_active = false WHERE id = :id"),
            {"id": user_id},
        )
        self._session.commit()
        return result.rowcount > 0


def hash_password(plain: str) -> str:
    """One-way hash a plaintext password."""
    return bcrypt.hashpw(plain.encode(), bcrypt.gensalt()).decode()
