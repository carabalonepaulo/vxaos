class AspectRatioContainer < Container
  def initialize(source=nil)
    init_source source
    init_inner_control_vars
  end

  def current_godot_object
    "AspectRatioContainer"
  end

  def ratio
    @source.ratio
  end

  def ratio=(v)
    @source.ratio = v
  end

  def stretch_mode
    @source.stretch_mode
  end

  def stretch_mode=(v)
    @source.stretch_mode = v
  end

  def alignment_horizontal
    @source.alignment_horizontal
  end

  def alignment_horizontal=(v)
    @source.alignment_horizontal = v
  end

  def alignment_vertical
    @source.alignment_vertical
  end

  def alignment_vertical=(v)
    @source.alignment_vertical = v
  end
end