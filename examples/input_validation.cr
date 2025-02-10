class User
  property id : Int32
  property name : String

  def initialize(@id : Int32, @name : String)
  end
end

def validate(user : User) : Bool
  user.id > 0 && user.name != ""
end
