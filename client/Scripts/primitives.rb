class Vector2
  attr_reader :source
  
  def initialize(x, y)
    @source = Main[:primitives].vector2 x, y
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
    @source = Main[:primitives].rect x, y, width, height
  end
  
  def x=(v)
    @source.position = Main[:primitives].vector2 v, y
  end
  
  def x
    @source.position.x
  end
  
  def y=(v)
    @source.position = Main[:primitives].vector2 x, v
  end
  
  def y
    @source.position.y
  end
  
  def width=(v)
    @source.size = Main[:primitives].vector2 v, height
  end
  
  def width
    @source.size.x
  end
  
  def height=(v)
    @source.size = Main[:primitives].vector2 width, v
  end
  
  def height
    @source.size.y
  end
end
  
class Color
  attr_reader :source
  
  def initialize(r, g, b, a = 255)
    @source = Main[:primitives].color r, g, b, a
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