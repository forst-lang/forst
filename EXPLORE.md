## To Explore

- Compile down to Go or Zig (interop, compiler performance)
- Types have functions that define constraints and narrow them down (types with multiple inheritance / polymorphism if you will)
  - e.g. `String.Min(3)` is a type that is constrained to be a string with a minimum length of 3
- All types are capital case or anonymous struct types
- All nominal types are aliases of constraints on other types
  - e.g. `Array(String)` is an anonymous, non-nominal type
  - but `type StringArray = Array(String)` is a nominal type
- Traits from Rust instead of classes? Allow any objects that satisfy the trait to be used in place of the trait.
  - Disallow methods in parameter object types
- Pattern matching from Rust / match statements
- Don't fall through in switch statements.
- Make function signatures GraphQL compatible as well
- Avoid curly braces in method parameters by using named parameters. Only require named parameters if the method has more than 1 parameter. This avoids the need to add curly braces when you want to add another parameter. Instead, you just add another parameter and require the calling site to now name all parameters.
