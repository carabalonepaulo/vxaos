class Vector2
  attr_reader :source
  
  def initialize(x, y)
    @source = $system.create_godot_object 'Vector2', x, y
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
end

class Rect
  attr_reader :source
  
  def initialize(x, y, width, height)
    @source = $system.create_godot_object 'Rect2', x, y, width, height
  end
  
  def x=(v)
    @source.position = $system.create_godot_object 'Vector2', v, y
  end
  
  def x
    @source.position.x
  end
  
  def y=(v)
    @source.position = $system.create_godot_object 'Vector2', x, v
  end
  
  def y
    @source.position.y
  end
  
  def width=(v)
    @source.size = $system.create_godot_object 'Vector2', v, height
  end
  
  def width
    @source.size.x
  end
  
  def height=(v)
    @source.size = $system.create_godot_object 'Vector2', width, v
  end
  
  def height
    @source.size.y
  end
end
  
class Color
  attr_reader :source
  
  def initialize(r, g, b, a = 255)
    @source = $system.create_godot_object 'Color'
    @source.r8 = r
    @source.g8 = g
    @source.b8 = b
    @source.a8 = a
  end
  
  def red=(v)
    @source.r8 = v
  end
  
  def red
    @source.r8
  end
  
  def green=(v)
    @source.g8 = v
  end
  
  def green
    @source.g8
  end
  
  def blue=(v)
    @source.b8 = v
  end
  
  def blue
    @source.b8
  end
  
  def alpha=(v)
    @source.a8 = v
  end
  
  def alpha
    @source.a8
  end
end