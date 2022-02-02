class ColorRect < Control
  def initialize(source = nil)
    init_source source
    init_inner_control_vars
  end

  def init_inner_control_vars
    super
    @color = Color.new @source.color
  end

  def color
    @color
  end

  def color=(v)
    @color = v
    @source.color = v.source
  end
end