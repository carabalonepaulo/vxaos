class ReferenceRect < Control
  def initialize(source = nil)
    init_source source
    init_inner_control_vars
  end

  def init_inner_control_vars
    super
    @border_color = Color.new @source.border_color
  end

  def border_color
    @border_color
  end

  def border_color=(v)
    @border_color = v
    @source.border_color = v.source
  end

  def border_width
    @source.border_width
  end

  def border_width=(v)
    @source.border_width = v
  end

  def editor_only
    @source.editor_only
  end

  def editor_only=(v)
    @source.editor_only = v
  end
end