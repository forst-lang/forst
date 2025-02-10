class ASTNode
end

class TypeDefinition < ASTNode
  property name : String
  property fields : Hash(String, String)

  def initialize(@name, @fields)
  end
end

class FunctionDefinition < ASTNode
  property name : String
  property param_name : String
  property param_type : String
  property return_type : String
  property body : String

  def initialize(@name, @param_name, @param_type, @return_type, @body)
  end
end

class Parser
  def initialize(@tokens : Array(Lexer::Token))
    @index = 0
  end

  def current_token
    @tokens[@index]?
  end

  def peek_token
    @tokens[@index + 1]?
  end

  def expect_token(type : Lexer::TokenType) : Lexer::Token?
    token = current_token
    return nil unless token && token.type == type
    advance
    token
  end

  def advance
    @index += 1
  end

  def parse
    ast_nodes = [] of ASTNode

    while token = current_token
      case token.type
      when Lexer::TokenType::Type
        if node = parse_type_definition
          ast_nodes << node
        end
      when Lexer::TokenType::Fn
        if node = parse_function_definition
          ast_nodes << node
        end
      else
        advance
      end
    end

    ast_nodes
  end

  def parse_type_definition : TypeDefinition?
    advance # Consume "type"

    name_token = current_token
    return nil unless name_token
    name = name_token.value
    advance # Consume identifier

    return nil unless expect_token(Lexer::TokenType::StructStart) # Consume "{"

    fields = {} of String => String
    while token = current_token
      break if token.type == Lexer::TokenType::StructEnd

      field_name = token.value
      advance # Consume field name

      return nil unless expect_token(Lexer::TokenType::Colon) # Consume ":"

      type_token = current_token
      return nil unless type_token
      field_type = type_token.value
      advance # Consume type

      fields[field_name] = field_type

      if comma_token = current_token
        advance if comma_token.type == Lexer::TokenType::Comma
      end
    end

    expect_token(Lexer::TokenType::StructEnd) # Consume "}"
    TypeDefinition.new(name, fields)
  end

  def parse_function_definition : FunctionDefinition?
    advance # Consume "fn"

    name_token = current_token
    return nil unless name_token
    name = name_token.value
    advance # Consume identifier

    return nil unless expect_token(Lexer::TokenType::LParen) # Consume "("

    param_token = current_token
    return nil unless param_token
    param_name = param_token.value
    advance # Consume parameter name

    return nil unless expect_token(Lexer::TokenType::Colon) # Consume ":"

    param_type_token = current_token
    return nil unless param_type_token
    param_type = param_type_token.value
    advance # Consume type

    return nil unless expect_token(Lexer::TokenType::RParen) # Consume ")"
    return nil unless expect_token(Lexer::TokenType::Arrow)  # Consume "->"

    return_type_token = current_token
    return nil unless return_type_token
    return_type = return_type_token.value
    advance # Consume return type

    return nil unless expect_token(Lexer::TokenType::StructStart) # Consume "{"

    body = ""
    while token = current_token
      break if token.type == Lexer::TokenType::StructEnd
      body += token.value + " "
      advance
    end

    expect_token(Lexer::TokenType::StructEnd) # Consume "}"
    FunctionDefinition.new(name, param_name, param_type, return_type, body.strip)
  end
end
