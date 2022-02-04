class TextureRect < Control
  def initialize(source = nil)
    init_source source
    init_inner_control_vars
  end

  def texture
    @source.texture
  end

  def texture=(v)
    @source.texture = v
  end

  def expand
    @source.expand
  end

  def expand=(v)
    @source.expand = v
  end

  def stretch_mode
    @source.stretch_mode
  end

  def stretch_mode=(v)
    @source.stretch_mode = v
  end

  def flip_h
    @source.flip_h
  end

  def flip_h=(v)
    @source.flip_h = v
  end

  def flip_v
    @source.flip_v
  end

  def flip_v=(v)
    @source.flip_v = v
  end
end