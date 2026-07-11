# async

Blocking sync Forst calling multiple TypeScript modules — async functions and async generators.

```ft
import node "./legacy/payment"
import node "./legacy/events"

func checkout(amount Float, currency String): String {
    result := payment.create(amount, currency)  // Result(T, Error)
    ensure result is Ok()
    return result.id
}
```

Requires `@forst/node-runtime` built (`task build:node-runtime`).

Run:

```bash
task example:node-interop-async
```
