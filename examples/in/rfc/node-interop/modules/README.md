# Node interop nested modules example

Fixture for **nested TypeScript module resolution** at runtime. Forst calls a single entry export (`legacy/api/checkout.ts` / `createOrder`), but that module imports a small graph of sibling modules:

```text
legacy/api/checkout.ts
  ├── legacy/services/payment.ts  → legacy/utils/money.ts, legacy/services/constants.ts
  └── legacy/services/pricing.ts  → legacy/services/tax.ts → constants.ts
                                    → legacy/utils/money.ts
```

Only `createOrder` is in the compile-time manifest. Nested `.ts` files load via normal Node `import` resolution when the entry module is invoked over RPC.

## Expected output

```text
ord_1:USD:100.00
USD:110.00
11000
```

(`100` USD + `10%` tax → `110.00`; payment id uses the shared `ord_` prefix from `constants.ts`.)

## Run

From repo root:

```bash
task example:node-interop-modules
```

Requires `@forst/node-runtime` built (`task build:node-runtime`).

## See also

- [sync](../sync/) — single-module sync RPC
- [async](../async/) — multiple top-level Forst imports
