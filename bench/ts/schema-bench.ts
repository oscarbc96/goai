import { z } from "zod";
import { zodSchema } from "ai";

// Simple schema - equivalent to Go BenchmarkProduct.
const ProductSchema = z.object({
  name: z.string(),
  description: z.string(),
  price: z.number(),
  category: z.string(),
  tags: z.array(z.string()),
  in_stock: z.boolean(),
});

// Complex schema - equivalent to Go BenchmarkOrder.
const CustomerSchema = z.object({
  name: z.string(),
  email: z.string(),
  address: z.string(),
});

const OrderSchema = z.object({
  id: z.string(),
  customer: CustomerSchema,
  items: z.array(ProductSchema),
  total: z.number(),
  status: z.string(),
});

// Benchmark simple schema: zodSchema() + access .jsonSchema to trigger actual conversion.
// This is equivalent to Go's SchemaFrom[T]() which does reflection + JSON marshal.
const simpleRuns = 10000;
const simpleStart = Bun.nanoseconds();
for (let i = 0; i < simpleRuns; i++) {
  const s = zodSchema(ProductSchema);
  // Access getter to trigger actual Zod → JSON Schema conversion.
  const _ = s.jsonSchema;
}
const simpleElapsed = Bun.nanoseconds() - simpleStart;

console.log(
  JSON.stringify({
    benchmark: "schema_simple",
    ns_per_op: Math.round(simpleElapsed / simpleRuns),
    ops_per_sec: Math.round((simpleRuns * 1e9) / simpleElapsed),
  })
);

// Benchmark complex schema generation.
const complexRuns = 10000;
const complexStart = Bun.nanoseconds();
for (let i = 0; i < complexRuns; i++) {
  const s = zodSchema(OrderSchema);
  const _ = s.jsonSchema;
}
const complexElapsed = Bun.nanoseconds() - complexStart;

console.log(
  JSON.stringify({
    benchmark: "schema_complex",
    ns_per_op: Math.round(complexElapsed / complexRuns),
    ops_per_sec: Math.round((complexRuns * 1e9) / complexElapsed),
  })
);
