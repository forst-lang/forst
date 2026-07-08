package testutil

import (
	"os"
	"path/filepath"
	"testing"

	"forst/internal/testmod"
)

const siblingUsersFtSource = `package users

type User = {
  name: String,
}
`

const siblingUsersGoSource = `package users

type User struct {
	Name string ` + "`json:\"name\"`" + `
}
`

const siblingAccountsGoSource = `package accounts

import (
	"os"

	"interopdemo/users"
)

func CreateAccount(u users.User) error {
	return os.MkdirAll(u.Name, 0o755)
}
`

// SiblingGoInteropFixture holds paths for a generic mixed Forst/Go sibling interop module.
type SiblingGoInteropFixture struct {
	ModuleRoot        string
	UsersImportPath   string
	AccountsImportPath string
	UsersDir          string
	AccountsDir       string
	CallerSource      string
}

// WriteSiblingGoInteropModuleFixture creates a temp module with:
//   - users/user.ft (Forst User shape)
//   - users/types.go (Go User struct)
//   - accounts/api.go (Go func taking users.User)
//
// Returns fixture metadata for signup .ft tests that call accounts.CreateAccount.
func WriteSiblingGoInteropModuleFixture(tb testing.TB) SiblingGoInteropFixture {
	tb.Helper()
	const modName = "interopdemo"
	root := tb.TempDir()
	usersDir := filepath.Join(root, "users")
	accountsDir := filepath.Join(root, "accounts")
	for _, d := range []string{usersDir, accountsDir} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			tbFail(tb, err)
			return SiblingGoInteropFixture{}
		}
	}
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte(testmod.GoModContent(modName)), 0o644); err != nil {
		tbFail(tb, err)
		return SiblingGoInteropFixture{}
	}
	if err := os.WriteFile(filepath.Join(usersDir, "user.ft"), []byte(siblingUsersFtSource), 0o644); err != nil {
		tbFail(tb, err)
		return SiblingGoInteropFixture{}
	}
	if err := os.WriteFile(filepath.Join(usersDir, "types.go"), []byte(siblingUsersGoSource), 0o644); err != nil {
		tbFail(tb, err)
		return SiblingGoInteropFixture{}
	}
	if err := os.WriteFile(filepath.Join(accountsDir, "api.go"), []byte(siblingAccountsGoSource), 0o644); err != nil {
		tbFail(tb, err)
		return SiblingGoInteropFixture{}
	}
	usersImport := modName + "/users"
	accountsImport := modName + "/accounts"
	callerSrc := `package signup

import (
  "` + accountsImport + `"
  "` + usersImport + `"
)

func registerUser(): Error {
  user := users.User{ name: "ada" }
  return accounts.CreateAccount(user)
}
`
	return SiblingGoInteropFixture{
		ModuleRoot:         root,
		UsersImportPath:    usersImport,
		AccountsImportPath: accountsImport,
		UsersDir:           usersDir,
		AccountsDir:        accountsDir,
		CallerSource:       callerSrc,
	}
}
