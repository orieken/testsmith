"""Order service — application logic layer."""
from __future__ import annotations

from typing import Dict, Optional

from orders.models import LineItem, Order, OrderStatus


class OrderNotFoundError(Exception):
    """Raised when an order ID does not exist in the repository."""


class OrderService:
    """Manages the lifecycle of orders in memory (swap for a DB-backed repo)."""

    def __init__(self) -> None:
        self._store: Dict[str, Order] = {}

    def create_order(self, order_id: str, customer_id: str) -> Order:
        if order_id in self._store:
            raise ValueError(f"Order {order_id!r} already exists")
        order = Order(order_id=order_id, customer_id=customer_id)
        self._store[order_id] = order
        return order

    def get_order(self, order_id: str) -> Order:
        try:
            return self._store[order_id]
        except KeyError:
            raise OrderNotFoundError(order_id)

    def add_item(self, order_id: str, item: LineItem) -> Order:
        order = self.get_order(order_id)
        order.add_item(item)
        return order

    def confirm_order(self, order_id: str) -> Order:
        order = self.get_order(order_id)
        order.confirm()
        return order

    def cancel_order(self, order_id: str) -> Order:
        order = self.get_order(order_id)
        order.cancel()
        return order

    def list_by_status(self, status: OrderStatus) -> list[Order]:
        return [o for o in self._store.values() if o.status == status]
