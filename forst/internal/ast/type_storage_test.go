package ast

import "testing"

func TestTypeNode_StorageClass(t *testing.T) {
	t.Parallel()
	isBuiltin := func(id TypeIdent) bool { return id == TypeInt || id == TypeArray }

	if got := NewTypeParamType("T").StorageClass(isBuiltin); got != TypeStorageTypeParam {
		t.Fatalf("type param: got %v", got)
	}
	if got := NewHashBasedType("T_abc").StorageClass(isBuiltin); got != TypeStorageBuiltinOrStructural {
		t.Fatalf("hash: got %v", got)
	}
	if got := NewBuiltinType(TypeString).StorageClass(isBuiltin); got != TypeStorageBuiltinOrStructural {
		t.Fatalf("go builtin: got %v", got)
	}
	if got := (TypeNode{Ident: TypeInt}).StorageClass(isBuiltin); got != TypeStorageBuiltinOrStructural {
		t.Fatalf("builtin ident: got %v", got)
	}
	if got := NewUserDefinedType("AppContext").StorageClass(isBuiltin); got != TypeStorageNamedUserType {
		t.Fatalf("named user type: got %v", got)
	}
}
