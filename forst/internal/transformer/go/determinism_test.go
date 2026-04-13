package transformergo

import "testing"

func TestTransformForstFileToGo_deterministicAcrossRuns_basicExample(t *testing.T) {
	t.Parallel()
	src := `package main

func sum(a Int, b Int): Int {
	return a + b
}

func main() {
	println(sum(1, 2))
}
`
	first := compileForstPipeline(t, src)
	for i := 0; i < 20; i++ {
		current := compileForstPipeline(t, src)
		if current != first {
			t.Fatalf("pipeline output changed between runs at iteration %d", i)
		}
	}
}

func TestTransformForstFileToGo_deterministicAcrossRuns_unionErrorExample(t *testing.T) {
	t.Parallel()
	src := `package main

error ParseError {
	message: String,
}

func parse(input String): Result(String, ParseError) {
	if input == "" {
		return ParseError{ message: "empty" }
	}
	return input
}

func main() {
	println("ok")
}
`
	first := compileForstPipeline(t, src)
	for i := 0; i < 20; i++ {
		current := compileForstPipeline(t, src)
		if current != first {
			t.Fatalf("pipeline output changed between runs at iteration %d", i)
		}
	}
}
