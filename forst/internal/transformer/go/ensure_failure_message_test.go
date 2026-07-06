package transformergo

import (
	"strings"
	"testing"

	"forst/internal/ast"
	goast "go/ast"
)

func TestEnsureConstraintWantHint(t *testing.T) {
	t.Parallel()
	log := setupTestLogger(nil)
	tr := setupTransformer(setupTypeChecker(log), log)

	tests := []struct {
		name string
		a    ast.AssertionNode
		want string
	}{
		{
			name: "bool true",
			a: ast.AssertionNode{
				BaseType:    typeIdentPtr(string(ast.TypeBool)),
				Constraints: []ast.ConstraintNode{{Name: string(TrueConstraint)}},
			},
			want: "true",
		},
		{
			name: "bool false",
			a: ast.AssertionNode{
				BaseType:    typeIdentPtr(string(ast.TypeBool)),
				Constraints: []ast.ConstraintNode{{Name: string(FalseConstraint)}},
			},
			want: "false",
		},
		{
			name: "int greater than",
			a: ast.AssertionNode{
				BaseType: typeIdentPtr(string(ast.TypeInt)),
				Constraints: []ast.ConstraintNode{{
					Name: string(GreaterThanConstraint),
					Args: []ast.ConstraintArgumentNode{{Value: intLiteralNodePtr(0)}},
				}},
			},
			want: "> 0",
		},
		{
			name: "type guard only",
			a: ast.AssertionNode{
				BaseType: typeIdentPtr("Strong"),
			},
			want: "Strong()",
		},
		{
			name: "result ok",
			a: ast.AssertionNode{
				Constraints: []ast.ConstraintNode{{Name: "Ok"}},
			},
			want: "Ok()",
		},
		{
			name: "string has prefix",
			a: ast.AssertionNode{
				BaseType: typeIdentPtr(string(ast.TypeString)),
				Constraints: []ast.ConstraintNode{{
					Name: string(HasPrefixConstraint),
					Args: []ast.ConstraintArgumentNode{{
						Value: strLiteralNodePtr("https://"),
					}},
				}},
			},
			want: `prefix "https://"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tr.ensureConstraintWantHint(&tt.a)
			if got != tt.want {
				t.Fatalf("got %q want %q", got, tt.want)
			}
		})
	}
}

func TestEnsureConstraintGotDiagnostic(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		setup      func(*Transformer)
		stmt       ast.EnsureNode
		wantFormat string
		wantIn     []string
		wantNotIn  []string
	}{
		{
			name: "bool false uses %t",
			stmt: ast.EnsureNode{
				Variable: ast.VariableNode{Ident: ast.Ident{ID: "flag"}},
				Assertion: ast.AssertionNode{
					BaseType:    typeIdentPtr(string(ast.TypeBool)),
					Constraints: []ast.ConstraintNode{{Name: string(FalseConstraint)}},
				},
			},
			wantFormat: "got %t, want %s",
			wantIn:     []string{"flag"},
		},
		{
			name: "string contains uses %q",
			stmt: ast.EnsureNode{
				Variable: ast.VariableNode{Ident: ast.Ident{ID: "s"}},
				Assertion: ast.AssertionNode{
					BaseType: typeIdentPtr(string(ast.TypeString)),
					Constraints: []ast.ConstraintNode{{
						Name: string(ContainsConstraint),
						Args: []ast.ConstraintArgumentNode{{
							Value: strLiteralNodePtr("needle"),
						}},
					}},
				},
			},
			wantFormat: "got %q, want %s",
			wantIn:     []string{"s"},
		},
		{
			name: "string not empty uses %q",
			stmt: ast.EnsureNode{
				Variable: ast.VariableNode{Ident: ast.Ident{ID: "s"}},
				Assertion: ast.AssertionNode{
					BaseType:    typeIdentPtr(string(ast.TypeString)),
					Constraints: []ast.ConstraintNode{{Name: string(NotEmptyConstraint)}},
				},
			},
			wantFormat: "got %q, want %s",
			wantIn:     []string{"s"},
		},
		{
			name: "string max uses len and %q",
			stmt: ast.EnsureNode{
				Variable: ast.VariableNode{Ident: ast.Ident{ID: "s"}},
				Assertion: ast.AssertionNode{
					BaseType: typeIdentPtr(string(ast.TypeString)),
					Constraints: []ast.ConstraintNode{{
						Name: string(MaxConstraint),
						Args: []ast.ConstraintArgumentNode{{Value: intLiteralNodePtr(10)}},
					}},
				},
			},
			wantFormat: "got len=%d (%q), want %s",
			wantIn:     []string{"len(", "s"},
		},
		{
			name: "multi constraint falls back to default %v",
			stmt: ast.EnsureNode{
				Variable: ast.VariableNode{Ident: ast.Ident{ID: "n"}},
				Assertion: ast.AssertionNode{
					BaseType: typeIdentPtr(string(ast.TypeInt)),
					Constraints: []ast.ConstraintNode{
						{Name: string(GreaterThanConstraint), Args: []ast.ConstraintArgumentNode{{Value: intLiteralNodePtr(0)}}},
						{Name: string(LessThanConstraint), Args: []ast.ConstraintArgumentNode{{Value: intLiteralNodePtr(100)}}},
					},
				},
			},
			wantFormat: "got %v, want %s",
			wantIn:     []string{"n"},
		},
		{
			name: "type guard only falls back to default %v",
			stmt: ast.EnsureNode{
				Variable: ast.VariableNode{Ident: ast.Ident{ID: "p"}},
				Assertion: ast.AssertionNode{
					BaseType: typeIdentPtr("Strong"),
				},
			},
			wantFormat: "got %v, want %s",
			wantIn:     []string{"p"},
		},
		{
			name: "pointer nil uses default %v",
			stmt: ast.EnsureNode{
				Variable: ast.VariableNode{Ident: ast.Ident{ID: "p"}},
				Assertion: ast.AssertionNode{
					BaseType:    typeIdentPtr(string(ast.TypeInt)),
					Constraints: []ast.ConstraintNode{{Name: string(NilConstraint)}},
				},
			},
			wantFormat: "got %v, want %s",
			wantIn:     []string{"p"},
		},
		{
			name: "result err uses err=%v",
			setup: func(tr *Transformer) {
				tr.resultLocalSplit = map[string]resultLocalSplit{
					"x": {errGoName: "xErr", successGoNames: []string{"x"}},
				}
			},
			stmt: ast.EnsureNode{
				Variable: ast.VariableNode{Ident: ast.Ident{ID: "x"}},
				Assertion: ast.AssertionNode{
					Constraints: []ast.ConstraintNode{{Name: "Err"}},
				},
			},
			wantFormat: "got err=%v, want %s",
			wantIn:     []string{"xErr"},
		},
		{
			name: "bool true uses %t",
			stmt: ast.EnsureNode{
				Variable: ast.VariableNode{Ident: ast.Ident{ID: "ok"}},
				Assertion: ast.AssertionNode{
					BaseType:    typeIdentPtr(string(ast.TypeBool)),
					Constraints: []ast.ConstraintNode{{Name: string(TrueConstraint)}},
				},
			},
			wantFormat: "got %t, want %s",
			wantIn:     []string{"ok"},
		},
		{
			name: "string has prefix uses %q",
			stmt: ast.EnsureNode{
				Variable: ast.VariableNode{Ident: ast.Ident{ID: "s"}},
				Assertion: ast.AssertionNode{
					BaseType: typeIdentPtr(string(ast.TypeString)),
					Constraints: []ast.ConstraintNode{{
						Name: string(HasPrefixConstraint),
						Args: []ast.ConstraintArgumentNode{{
							Value: strLiteralNodePtr("https://"),
						}},
					}},
				},
			},
			wantFormat: "got %q, want %s",
			wantIn:     []string{"s"},
		},
		{
			name: "string min uses len and %q",
			stmt: ast.EnsureNode{
				Variable: ast.VariableNode{Ident: ast.Ident{ID: "s"}},
				Assertion: ast.AssertionNode{
					BaseType: typeIdentPtr(string(ast.TypeString)),
					Constraints: []ast.ConstraintNode{{
						Name: string(MinConstraint),
						Args: []ast.ConstraintArgumentNode{{Value: intLiteralNodePtr(1)}},
					}},
				},
			},
			wantFormat: "got len=%d (%q), want %s",
			wantIn:     []string{"len(", "s"},
		},
		{
			name: "int greater than uses default %v",
			stmt: ast.EnsureNode{
				Variable: ast.VariableNode{Ident: ast.Ident{ID: "n"}},
				Assertion: ast.AssertionNode{
					BaseType: typeIdentPtr(string(ast.TypeInt)),
					Constraints: []ast.ConstraintNode{{
						Name: string(GreaterThanConstraint),
						Args: []ast.ConstraintArgumentNode{{Value: intLiteralNodePtr(0)}},
					}},
				},
			},
			wantFormat: "got %v, want %s",
			wantIn:     []string{"n"},
		},
		{
			name: "result ok uses err=%v",
			setup: func(tr *Transformer) {
				tr.resultLocalSplit = map[string]resultLocalSplit{
					"x": {errGoName: "xErr", successGoNames: []string{"x"}},
				}
			},
			stmt: ast.EnsureNode{
				Variable: ast.VariableNode{Ident: ast.Ident{ID: "x"}},
				Assertion: ast.AssertionNode{
					Constraints: []ast.ConstraintNode{{Name: "Ok"}},
				},
			},
			wantFormat: "got err=%v, want %s",
			wantIn:     []string{"xErr"},
			wantNotIn:  []string{"x,"},
		},
		{
			name: "result ok with arg uses err and val",
			setup: func(tr *Transformer) {
				tr.resultLocalSplit = map[string]resultLocalSplit{
					"x": {errGoName: "xErr", successGoNames: []string{"x"}},
				}
			},
			stmt: ast.EnsureNode{
				Variable: ast.VariableNode{Ident: ast.Ident{ID: "x"}},
				Assertion: ast.AssertionNode{
					Constraints: []ast.ConstraintNode{{
						Name: "Ok",
						Args: []ast.ConstraintArgumentNode{{Value: intLiteralNodePtr(42)}},
					}},
				},
			},
			wantFormat: "got err=%v, val=%v, want %s",
			wantIn:     []string{"xErr", "x"},
		},
		{
			name: "result ok without split falls back to subject %v",
			stmt: ast.EnsureNode{
				Variable: ast.VariableNode{Ident: ast.Ident{ID: "x"}},
				Assertion: ast.AssertionNode{
					Constraints: []ast.ConstraintNode{{Name: "Ok"}},
				},
			},
			wantFormat: "got %v, want %s",
			wantIn:     []string{"x"},
			wantNotIn:  []string{"xErr"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			log := setupTestLogger(nil)
			tr := setupTransformer(setupTypeChecker(log), log)
			if tt.setup != nil {
				tt.setup(tr)
			}
			subjectExpr, err := tr.transformExpression(tt.stmt.Variable)
			if err != nil {
				t.Fatal(err)
			}
			got := tr.ensureConstraintGotDiagnostic(tt.stmt, subjectExpr)
			if got.suffixFormat != tt.wantFormat {
				t.Fatalf("format got %q want %q", got.suffixFormat, tt.wantFormat)
			}
			var argStrs []string
			for _, arg := range got.gotArgs {
				argStrs = append(argStrs, goExprString(t, arg))
			}
			s := strings.Join(argStrs, ", ")
			for _, sub := range tt.wantIn {
				if !strings.Contains(s, sub) {
					t.Fatalf("missing %q in got args:\n%s", sub, s)
				}
			}
			for _, sub := range tt.wantNotIn {
				if strings.Contains(s, sub) {
					t.Fatalf("unexpected %q in got args:\n%s", sub, s)
				}
			}
		})
	}
}

func TestEnsureTestFatalfCall_staticFallbackOmitsGot(t *testing.T) {
	t.Parallel()
	log := setupTestLogger(nil)
	tr := setupTransformer(setupTypeChecker(log), log)
	// Multi-value Result split — transformExpression rejects single binding.
	tr.resultLocalSplit = map[string]resultLocalSplit{
		"x": {errGoName: "xErr", successGoNames: []string{"a", "b"}},
	}
	stmt := ast.EnsureNode{
		Variable: ast.VariableNode{Ident: ast.Ident{ID: "x"}},
		Assertion: ast.AssertionNode{
			Constraints: []ast.ConstraintNode{{Name: "Ok"}},
		},
	}
	call := tr.ensureTestFatalfCall(goast.NewIdent("t"), stmt)
	s := goExprString(t, call)
	for _, sub := range []string{
		`t.Fatalf(`,
		`"ensure %s is %s"`,
		`"x"`,
	} {
		if !strings.Contains(s, sub) {
			t.Fatalf("missing %q in:\n%s", sub, s)
		}
	}
	for _, sub := range []string{"got %", "got err="} {
		if strings.Contains(s, sub) {
			t.Fatalf("static fallback should not contain %q in:\n%s", sub, s)
		}
	}
}

func TestEnsureTestFatalfCall_emitsGotWant(t *testing.T) {
	t.Parallel()
	log := setupTestLogger(nil)
	tr := setupTransformer(setupTypeChecker(log), log)
	stmt := ast.EnsureNode{
		Variable: ast.VariableNode{Ident: ast.Ident{ID: "ok"}},
		Assertion: ast.AssertionNode{
			BaseType:    typeIdentPtr(string(ast.TypeBool)),
			Constraints: []ast.ConstraintNode{{Name: string(TrueConstraint)}},
		},
	}
	call := tr.ensureTestFatalfCall(goast.NewIdent("t"), stmt)
	s := goExprString(t, call)
	for _, sub := range []string{
		`t.Fatalf(`,
		`"ensure %s is %s: got %t, want %s"`,
		`"ok"`,
		`"Bool.True()"`,
		`"true"`,
		`ok`,
	} {
		if !strings.Contains(s, sub) {
			t.Fatalf("missing %q in:\n%s", sub, s)
		}
	}
}

func TestEnsureTestFatalfCall_fieldAccessSubject(t *testing.T) {
	t.Parallel()
	log := setupTestLogger(nil)
	tr := setupTransformer(setupTypeChecker(log), log)
	stmt := ast.EnsureNode{
		Variable: ast.VariableNode{Ident: ast.Ident{ID: "s.active"}},
		Assertion: ast.AssertionNode{
			BaseType:    typeIdentPtr(string(ast.TypeBool)),
			Constraints: []ast.ConstraintNode{{Name: string(FalseConstraint)}},
		},
	}
	call := tr.ensureTestFatalfCall(goast.NewIdent("t"), stmt)
	s := goExprString(t, call)
	for _, sub := range []string{
		`"s.active"`,
		`"false"`,
		`got %t`,
		`s.active`,
	} {
		if !strings.Contains(s, sub) {
			t.Fatalf("missing %q in:\n%s", sub, s)
		}
	}
}

func TestDefaultAssertionErrorExpr_includesSubjectAndWant(t *testing.T) {
	t.Parallel()
	log := setupTestLogger(nil)
	tr := setupTransformer(setupTypeChecker(log), log)
	stmt := ast.EnsureNode{
		Variable: ast.VariableNode{Ident: ast.Ident{ID: "n"}},
		Assertion: ast.AssertionNode{
			BaseType: typeIdentPtr(string(ast.TypeInt)),
			Constraints: []ast.ConstraintNode{{
				Name: string(GreaterThanConstraint),
				Args: []ast.ConstraintArgumentNode{{Value: intLiteralNodePtr(0)}},
			}},
		},
	}
	s := goExprString(t, tr.defaultAssertionErrorExpr(stmt))
	for _, sub := range []string{
		`errors.New`,
		`ensure n is Int.GreaterThan(0): want > 0`,
	} {
		if !strings.Contains(s, sub) {
			t.Fatalf("missing %q in:\n%s", sub, s)
		}
	}
}

func strLiteralNodePtr(v string) *ast.ValueNode {
	var n ast.ValueNode = ast.StringLiteralNode{Value: v}
	return &n
}
