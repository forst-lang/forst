import { createServer } from "node:http";
import { handle } from "./node_invoke";
import { createTRPCProxyClient, httpBatchLink } from "@trpc/client";
import { initTRPC } from "@trpc/server";
import { z } from "zod";

const port = 3000;

const t = initTRPC.create();

export const router = t.router;
export const publicProcedure = t.procedure;

const appRouter = router({
  createUser: publicProcedure
    .input(
      z.object({
        id: z.string().uuid(),
        name: z.string().min(3).max(10),
        phoneNumber: z
          .string()
          .min(3)
          .max(10)
          .refine(
            (val: string) => val.startsWith("+") || val.startsWith("0"),
            "Phone number must start with + or 0"
          ),
        bankAccount: z.object({
          iban: z.string().min(10).max(34),
        }),
      })
    )
    .mutation(async ({ input }) => {
      // Send the input to Forst for validation
      const response = await fetch(`http://localhost:${port}`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(input),
      });

      if (!response.ok) {
        const error = await response.json();
        throw new Error(error.error || "Validation failed");
      }

      const result = await response.json();
      return result.data;
    }),
});

const url = `http://localhost:${port}`;

const trpc = createTRPCProxyClient<typeof appRouter>({
  links: [
    httpBatchLink({
      url,
    }),
  ],
});

const server = createServer(handle);

server.listen(port, async () => {
  console.log(`Server running at ${url}`);

  try {
    // Test tRPC call
    const result = await trpc.createUser.mutate({
      id: "123e4567-e89b-12d3-a456-426614174000",
      name: "John Doe",
      phoneNumber: "+49123456789",
      bankAccount: {
        iban: "DE89370400440532013000",
      },
    });

    console.log("Response:", result);
  } catch (err) {
    console.error("Error:", err);
  } finally {
    server.close();
  }
});
