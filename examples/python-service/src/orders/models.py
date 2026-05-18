"""Order domain models."""
from __future__ import annotations

from dataclasses import dataclass, field
from enum import Enum
from typing import List


class OrderStatus(Enum):
    PENDING = "pending"
    CONFIRMED = "confirmed"
    SHIPPED = "shipped"
    CANCELLED = "cancelled"


@dataclass
class LineItem:
    sku: str
    quantity: int
    unit_price_cents: int

    @property
    def total_cents(self) -> int:
        return self.quantity * self.unit_price_cents


@dataclass
class Order:
    order_id: str
    customer_id: str
    items: List[LineItem] = field(default_factory=list)
    status: OrderStatus = OrderStatus.PENDING

    @property
    def total_cents(self) -> int:
        return sum(item.total_cents for item in self.items)

    def add_item(self, item: LineItem) -> None:
        if self.status != OrderStatus.PENDING:
            raise ValueError(f"Cannot modify order in status {self.status.value}")
        self.items.append(item)

    def confirm(self) -> None:
        if not self.items:
            raise ValueError("Cannot confirm an empty order")
        if self.status != OrderStatus.PENDING:
            raise ValueError(f"Order already in status {self.status.value}")
        self.status = OrderStatus.CONFIRMED

    def cancel(self) -> None:
        if self.status == OrderStatus.SHIPPED:
            raise ValueError("Cannot cancel a shipped order")
        self.status = OrderStatus.CANCELLED
