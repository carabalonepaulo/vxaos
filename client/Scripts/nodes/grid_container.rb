class GridContainer < Container
  def initialize(source = nil)
    init_source source
    init_inner_control_vars
  end

  def columns
    @source.columns
  end

  def columns=(v)
    @source.columns = v
  end

  def mouse_filter
    @source.mouse_filter
  end

  def mouse_filter=(v)
    @source.mouse_filter = v
  end
end
