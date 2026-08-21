import Stripe from 'stripe';
import axios from 'axios';
import { EventEmitter } from 'events';
import * as path from 'path';

export interface PaymentConfig {
  apiKey: string;
  currency: string;
}

export type PaymentStatus = 'pending' | 'processing' | 'completed' | 'failed';

export class PaymentService {
  private stripe: Stripe;
  private config: PaymentConfig;

  constructor(config: PaymentConfig) {
    this.config = config;
    this.stripe = new Stripe(config.apiKey, { apiVersion: '2023-10-16' });
  }

  async charge(amount: number, token: string): Promise<PaymentStatus> {
    const charge = await this.stripe.charges.create({
      amount,
      currency: this.config.currency,
      source: token,
    });
    return charge.status === 'succeeded' ? 'completed' : 'failed';
  }

  async refund(chargeId: string): Promise<boolean> {
    const refund = await this.stripe.refunds.create({ charge: chargeId });
    return refund.status === 'succeeded';
  }
}

export async function fetchExchangeRate(from: string, to: string): Promise<number> {
  const response = await axios.get(`https://api.exchangerate.host/convert?from=${from}&to=${to}`);
  return response.data.result;
}

export function formatAmount(amount: number, currency: string): string {
  return new Intl.NumberFormat('en-US', { style: 'currency', currency }).format(amount / 100);
}
