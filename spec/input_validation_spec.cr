require "spec"

describe "Input validation" do
  it "correctly compiles Crysma to Crystal" do
    input = "examples/input_validation.crysma"

    expected_output = <<-CRYSTAL
    class User
      property id : Int32
      property name : String
    
      def initialize(@id : Int32, @name : String)
      end
    end
    
    def validate(user : User) : Bool
      return user.id > 0 && user.name != ""
    end
    CRYSTAL

    actual_output = `crystal run src/main.cr -- "#{input}"`.strip
    expected_output = expected_output.strip

    actual_output.should eq(expected_output)
  end
end
