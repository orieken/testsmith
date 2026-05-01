export function formatPrice(amount: number, currency: string): string {
  return `${amount.toFixed(2)} ${currency}`;
}

export function slugify(text: string): string {
  return text.toLowerCase().replace(/\s+/g, "-");
}
