.PHONY: test run-examples

# Run all tests
test:
	cd forst && go test -v ./internal/...

# Run specific test packages
test-typechecker:
	cd forst && go test -v ./internal/typechecker/...

test-parser:
	cd forst && go test -v ./internal/parser/...

test-ast:
	cd forst && go test -v ./internal/ast/...

# Run example compilations
run-example-shape-guard:
	cd forst/cmd/forst && go run . run -trace -- ../../../examples/in/rfc/guard/shape_guard.ft

run-example-basic-guard:
	cd forst/cmd/forst && go run . run -trace -- ../../../examples/in/rfc/guard/basic_guard.ft

run-example-basic:
	cd forst/cmd/forst && go run . run -trace -- ../../../examples/in/basic.ft

run-example-basic-function:
	cd forst/cmd/forst && go run . run -trace -- ../../../examples/in/basic_function.ft

run-example-ensure:
	cd forst/cmd/forst && go run . run -trace -- ../../../examples/in/ensure.ft

# Run all examples
run-examples: run-example-shape-guard run-example-basic-guard run-example-basic run-example-basic-function run-example-ensure

# Run specific example
run-example:
	cd forst/cmd/forst && go run . run -trace -- $(FILE)