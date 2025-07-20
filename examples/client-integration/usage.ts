// Example usage of the generated @forst/client
import {
  CreateUser,
  GetUserById,
  UpdateUserAge,
  User,
  CreateUserRequest,
} from "./generated/user";

async function main() {
  try {
    // Create a new user with full type safety
    const createRequest: CreateUserRequest = {
      name: "Alice Johnson",
      age: 28,
      email: "alice@example.com",
    };

    console.log("Creating user...");
    const createResult = await CreateUser(createRequest);
    console.log("User created:", createResult.user);
    console.log("Created at:", createResult.created_at);

    // Get user by ID
    console.log("\nFetching user...");
    const user = await GetUserById("123");
    console.log("User found:", user);

    // Update user age
    console.log("\nUpdating user age...");
    const updatedUser = await UpdateUserAge("123", 29);
    console.log("User updated:", updatedUser);

    // TypeScript provides full IntelliSense and type checking
    console.log("User name:", user.name); // TypeScript knows this is a string
    console.log("User age:", user.age); // TypeScript knows this is a number

    // This would cause a TypeScript error:
    // console.log(user.invalidField); // Property 'invalidField' does not exist
  } catch (error) {
    console.error("Error:", error);
  }
}

// Example with error handling
async function createUserWithValidation() {
  try {
    const invalidRequest = {
      name: "", // Invalid: empty name
      age: 200, // Invalid: age too high
      email: "invalid-email", // Invalid: no @ symbol
    };

    // This will throw an error due to validation
    await CreateUser(invalidRequest as any);
  } catch (error) {
    console.log("Validation error caught:", error.message);
  }
}

// Example with streaming (if supported)
async function processLargeDataset() {
  const largeDataset = Array.from({ length: 10000 }, (_, i) => ({
    id: `user-${i}`,
    name: `User ${i}`,
    age: 20 + (i % 50),
    email: `user${i}@example.com`,
  }));

  // Process in chunks with streaming
  for (const user of largeDataset) {
    try {
      const result = await CreateUser({
        name: user.name,
        age: user.age,
        email: user.email,
      });
      console.log(`Created user ${result.user.id}`);
    } catch (error) {
      console.error(`Failed to create user ${user.id}:`, error.message);
    }
  }
}

// Run examples
if (require.main === module) {
  main()
    .then(() => createUserWithValidation())
    .then(() => processLargeDataset())
    .catch(console.error);
}
