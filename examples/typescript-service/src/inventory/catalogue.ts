/** In-memory product catalogue. */
import { Product, reserve, release, restock, InsufficientStockError } from "./product.js";

export class ProductNotFoundError extends Error {
  constructor(sku: string) {
    super(`Product not found: ${sku}`);
    this.name = "ProductNotFoundError";
  }
}

export class Catalogue {
  private readonly products = new Map<string, Product>();

  /** Adds a new product. Throws if the SKU already exists. */
  addProduct(product: Product): void {
    if (this.products.has(product.sku)) {
      throw new Error(`Product ${product.sku} already exists`);
    }
    this.products.set(product.sku, { ...product });
  }

  /** Retrieves a product by SKU. Throws ProductNotFoundError when absent. */
  getProduct(sku: string): Product {
    const p = this.products.get(sku);
    if (!p) throw new ProductNotFoundError(sku);
    return p;
  }

  /** Reserves units of a product for an order. */
  reserveUnits(sku: string, units: number): void {
    reserve(this.getProduct(sku), units);
  }

  /** Releases previously reserved units. */
  releaseUnits(sku: string, units: number): void {
    release(this.getProduct(sku), units);
  }

  /** Restocks a product. */
  restockProduct(sku: string, units: number): void {
    restock(this.getProduct(sku), units);
  }

  /** Lists all products with available stock above zero. */
  listAvailable(): Product[] {
    return [...this.products.values()].filter((p) => p.stockLevel - p.reservedLevel > 0);
  }
}
