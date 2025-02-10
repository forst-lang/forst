class User
  property id : UUID
  property name : String
  property phone_numbers : Array(PhoneNumber)

  def initialize(@id : UUID, @name : String, @phone_numbers : Array(PhoneNumber))
  end
end

struct CreateUserInput
  include JSON::Serializable

  property id : UUID
  property name : String
  property phone_numbers : Array(PhoneNumber)
end

def createUser!(input_json)
  # Parse input into struct
  input = CreateUserInput.from_json(input_json)

  # Validate input
  unless input.name.size.between?(3, 10)
    raise "Name must be between 3 and 10 characters"
  end

  unless input.phone_numbers.all? { |num| num.valid? }
    raise "Invalid phone number format"
  end

  # Create user if validation passes
  user = User.new(
    id: input.id,
    name: input.name,
    phone_numbers: input.phone_numbers
  )

  puts "Creating user with id: #{user.id}"
  user
end
