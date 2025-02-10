class CodeGenerator
  def initialize(@ast : Array(ASTNode))
  end

  def generate
    output = ""

    @ast.each do |node|
      case node
      when TypeDefinition
        output += generate_struct(node)
      when FunctionDefinition
        output += generate_function(node)
      end
    end

    output
  end

  def generate_struct(type : TypeDefinition)
    struct_def = "struct #{type.name}\n"
    type.fields.each do |name, type|
      struct_def += "  property #{name} : #{map_type(type)}\n"
    end
    struct_def += "end\n\n"
    struct_def
  end

  def generate_function(func : FunctionDefinition)
    "def #{func.name}(#{func.param_name} : #{func.param_type.capitalize}) : #{map_type(func.return_type)}\n  #{func.body}\nend\n\n"
  end

  def map_type(type : String)
    case type
    when "int"
      "Int32"
    when "string"
      "String"
    when "bool"
      "Bool"
    else
      type
    end
  end
end
