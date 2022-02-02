class LinkButton < BaseButton
  def initialize(source = nil)
    init_source source
    init_inner_control_vars
  end

  def text
    @source.text
  end

  def text=(v)
    @source.text = v
  end

  def underline
    @source.underline
  end

  def underline=(v)
    @source.underline = v
  end

  def focus_mode
    @source.focus_mode
  end

  def focus_mode=(v)
    @source.focus_mode = v
  end

  def mouse_default_cursor_shape
    @source.mouse_default_cursor_shape
  end

  def mouse_default_cursor_shape=(v)
    @source.mouse_default_cursor_shape = v
  end
end