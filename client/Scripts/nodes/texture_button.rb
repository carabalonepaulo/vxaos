class TextureButton < BaseButton
  def initialize(source = nil)
    init_source source
    init_inner_control_vars
  end

  def texture_normal
    @source.texture_normal
  end

  def texture_normal=(v)
    @source.texture_normal = v
  end

  def texture_pressed
    @source.texture_pressed
  end

  def texture_pressed=(v)
    @source.texture_pressed = v
  end

  def texture_hover
    @source.texture_hover
  end

  def texture_hover=(v)
    @source.texture_hover = v
  end

  def texture_disabled
    @source.texture_disabled
  end

  def texture_disabled=(v)
    @source.texture_disabled = v
  end

  def texture_focused
    @source.texture_focused
  end

  def texture_focused=(v)
    @source.texture_focused = v
  end

  def texture_click_mask
    raise 'not implemented yet'
  end

  def texture_click_mask=(v)
    raise 'not implemented yet'
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
