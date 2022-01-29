class Rect
  attr_accessor :source
  
  def initialize(x = 0, y = 0, width = 0, height = 0)
    if x.class == 'Godot::Rect2'
      @source = x
    else
      @pos = Vector2.new x, y
      @size = Vector2.new width, height
      @source = $system.create_godot_object 'Rect2', @pos.source, @size.source
    end
  end
  
  def x=(v)
    @pos.x = v
    @source.position = @pos.source
  end
  
  def x
    @pos.x
  end
  
  def y=(v)
    @pos.y = v
    @source.position = @pos.source
  end
  
  def y
    @pos.y
  end
  
  def width=(v)
    @size.x = v
    @source.size = @size.source
  end
  
  def width
    @size.x
  end
  
  def height=(v)
    @size.y = v
    @source.size = @size.source
  end
  
  def height
    @size.y
  end

  def to_s
    "<Rect @x=#{x} @y=#{y} @width=#{width} @height=#{height}>"
  end
end