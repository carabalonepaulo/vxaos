class Vector2
  attr_accessor :source
  
  def initialize(x, y = nil)
    if x.class == 'Godot::Vector2'
      @source = x
    else
      @source = $system.create_godot_object 'Vector2', x, y
    end
  end
  
  def x=(v)
    @source.x = v
  end
  
  def x
    @source.x
  end
  
  def y=(v)
    @source.y = v
  end
  
  def y
    @source.y
  end
  
  def width=(v)
    @source.x = v
  end
  
  def width
    @source.x
  end
  
  def height=(v)
    @source.y = v
  end
  
  def height
    @source.y
  end

  def to_s
    "<Vector2 @x=#{x} @y=#{y}>"
  end
end

class EmptyVector2 < Vector2
  def initialize
  end
end