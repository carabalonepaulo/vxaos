class Color < Wrapper
  White = Color.new(255, 255, 255, 255).freeze
  Black = Color.new(0, 0, 0, 255).freeze
  
  def initialize(r = 0, g = 0, b = 0, a = 255)
    if r.class == 'Godot::Color'
      @source = r
    else
      @source = $system.create_godot_object 'Color'
      @source.r8 = r
      @source.g8 = g
      @source.b8 = b
      @source.a8 = a
    end
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

  def to_s
    "<Color @red=#{red} @green=#{green} @blue=#{blue} @alpha=#{alpha}>"
  end
end