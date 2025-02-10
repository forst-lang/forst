class Lexer
  enum TokenType
    Type
    StructStart
    StructEnd
    Fn
    Identifier
    Colon
    Comma
    Arrow
    LBrace
    RBrace
    LParen
    RParen
    Return
    Op
    Number
    String
    Bool
    EOF
  end

  struct Token
    property type : TokenType
    property value : String

    def initialize(@type : TokenType, @value : String)
    end
  end

  def initialize(@input : String)
    @tokens = [] of Token
    tokenize
  end

  def tokenize
    patterns = {
      /\btype\b/               => TokenType::Type,
      /\bfn\b/                 => TokenType::Fn,
      /[a-zA-Z_][a-zA-Z0-9_]*/ => TokenType::Identifier,
      /\{/                     => TokenType::StructStart,
      /\}/                     => TokenType::StructEnd,
      /:/                      => TokenType::Colon,
      /,/                      => TokenType::Comma,
      /->/                     => TokenType::Arrow,
      /\(/                     => TokenType::LParen,
      /\)/                     => TokenType::RParen,
      /return/                 => TokenType::Return,
      /[=><!]+/                => TokenType::Op,
      /\d+/                    => TokenType::Number,
      /"[^"]*"/                => TokenType::String,
      /\btrue\b|\bfalse\b/     => TokenType::Bool,
    }

    until @input.empty?
      tokenized = false
      patterns.each do |regex, type|
        if match = @input.match(/^#{regex}/)
          @tokens << Token.new(type, match[0])
          @input = @input[match[0].size..-1]
          tokenized = true
          break
        end
      end

      unless tokenized
        @input = @input[1..-1] # Skip unknown character
      end
    end

    @tokens << Token.new(TokenType::EOF, "")
  end

  def next_token
    @tokens.shift?
  end
end
