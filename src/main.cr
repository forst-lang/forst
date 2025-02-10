require "option_parser"
require "./lexer"
require "./parser"
require "./code_generator"

def main(source_code : String, output_file : String?)
  lexer = Lexer.new(source_code)
  parser = Parser.new(lexer.tokenize)
  ast = parser.parse
  generator = CodeGenerator.new(ast)
  generated_code = generator.generate

  if output_file
    File.write(output_file, generated_code)
  else
    puts generated_code
  end
end

input_file = ""
output_file = nil

OptionParser.parse do |parser|
  parser.banner = "Usage: crysma [input_file] [-o output_file]"

  parser.on("-h", "--help", "Show this help") do
    puts parser
    exit
  end

  parser.on("-o FILE", "--output=FILE", "Write output to FILE") do |file|
    output_file = file
  end

  parser.unknown_args do |args|
    if args.size != 1
      STDERR.puts "Error: Please provide exactly one input file"
      STDERR.puts parser
      exit(1)
    end
    input_file = args[0]
  end
end

source_code = File.read(input_file)
main(source_code, output_file)
