package parser

// IsMainFunction checks if the current function is the main function
func (c *Context) IsMainFunction() bool {
	if c.Package == nil || !c.Package.IsMainPackage() {
		return false
	}

	return c.ScopeStack.current.FunctionName == "main"
}

func (c *Context) IsTypeGuard() bool {
	return c.ScopeStack.current.IsTypeGuard
}
