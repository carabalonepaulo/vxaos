class ProgressBar < Range
  def initialize(source = nil)
    init_source source
    init_inner_control_vars
  end

  def percent_visible
    @source.percent_visible
  end

  def percent_visible=(v)
    @source.percent_visible = v
  end
end