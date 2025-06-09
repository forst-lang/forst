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

test-lexer:
	cd forst && go test -v ./internal/lexer/...

test-transformer:
	cd forst && go test -v ./internal/transformer/...

# Run specific example
run-example:
	cd forst/cmd/forst && go run . run -trace -report-phases -- $(FILE)

# Run example compilations
run-example-shape-guard:
	cd forst/cmd/forst && go run . run -trace -report-phases -- ../../../examples/in/rfc/guard/shape_guard.ft

run-example-basic-guard:
	cd forst/cmd/forst && go run . run -trace -report-phases -- ../../../examples/in/rfc/guard/basic_guard.ft

run-example-basic:
	cd forst/cmd/forst && go run . run -trace -report-phases -- ../../../examples/in/basic.ft

run-example-basic-function:
	cd forst/cmd/forst && go run . run -trace -report-phases -- ../../../examples/in/basic_function.ft

run-example-ensure:
	cd forst/cmd/forst && go run . run -trace -report-phases -- ../../../examples/in/ensure.ft

run-example-pointers:
	cd forst/cmd/forst && go run . run -trace -report-phases -- ../../../examples/in/pointers.ft

test-examples:
	cd forst && go test ./cmd/forst/...