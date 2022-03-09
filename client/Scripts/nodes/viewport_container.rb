class ViewportContainer < Container
  def initialize(source = nil)
    init_source source
    init_inner_control_vars
  end

  def current_godot_object
    "ViewportContainer"
  end

  def stretch
    @source.stretch
  end

  def stretch=(v)
    @source.stretch = v
  end

  def stretch_shrink
    @source.stretch_shrink
  end

  def stretch_shrink=(v)
    @source.stretch_shrink = v
  end
end