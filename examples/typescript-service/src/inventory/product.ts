/** Inventory product domain model. */
export interface Product {
  readonly sku: string;
  name: string;
  stockLevel: number;
  reservedLevel: number;
}

/** Error thrown when an operation would push stock below zero. */
export class InsufficientStockError extends Error {
  constructor(sku: string, requested: number, available: number) {
    super(`SKU ${sku}: requested ${requested}, available ${available}`);
    this.name = "InsufficientStockError";
  }
}

/** Returns the number of units actually available (stock minus reserved). */
export function availableStock(product: Product): number {
  return product.stockLevel - product.reservedLevel;
}

/** Reserves units for an in-flight order.
 *  Throws InsufficientStockError when available stock is too low.
 */
export function reserve(product: Product, units: number): void {
  if (units <= 0) throw new RangeError("units must be positive");
  const available = availableStock(product);
  if (units > available) {
    throw new InsufficientStockError(product.sku, units, available);
  }
  product.reservedLevel += units;
}

/** Releases previously reserved units (e.g. after order cancellation). */
export function release(product: Product, units: number): void {
  if (units <= 0) throw new RangeError("units must be positive");
  product.reservedLevel = Math.max(0, product.reservedLevel - units);
}

/** Restocks the product by adding units to stock level. */
export function restock(product: Product, units: number): void {
  if (units <= 0) throw new RangeError("units must be positive");
  product.stockLevel += units;
}
