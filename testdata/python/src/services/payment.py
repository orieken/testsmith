"""Payment processing service."""
import os
import logging

import stripe
import requests

from services.models import Transaction
from services.exceptions import PaymentError

logger = logging.getLogger(__name__)


class PaymentProcessor:
    """Handles payment charging and refunding via Stripe."""

    def __init__(self, api_key: str):
        self.api_key = api_key
        stripe.api_key = api_key

    def charge(self, amount: int, currency: str, token: str) -> dict:
        """Create a charge via Stripe. Amount in smallest currency unit."""
        try:
            charge = stripe.Charge.create(
                amount=amount,
                currency=currency,
                source=token,
            )
            logger.info("Charge created", extra={"charge_id": charge["id"]})
            return charge
        except stripe.error.CardError as e:
            raise PaymentError(str(e)) from e

    def refund(self, charge_id: str) -> dict:
        """Refund a previously created charge."""
        return stripe.Refund.create(charge=charge_id)


def calculate_total(items: list, tax_rate: float = 0.0) -> int:
    """Calculate the total amount in cents from a list of item dicts."""
    subtotal = sum(item["price"] * item["quantity"] for item in items)
    return int(subtotal * (1 + tax_rate))


def fetch_exchange_rate(base: str, target: str) -> float:
    """Fetch the current exchange rate from an external API."""
    response = requests.get(
        f"https://api.exchangerate.host/convert",
        params={"from": base, "to": target},
        timeout=5,
    )
    response.raise_for_status()
    return response.json()["result"]
