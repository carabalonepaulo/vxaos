class CheckButton < Button
  def initialize(source = nil)
    init_source source
    init_inner_control_vars
  end

  def align
    @source.align
  end

  def align=(v)
    @source.align = v
  end

  def toggle_mode
    @source.toggle_mode
  end

  def toggle_mode=(v)
    @source.toggle_mode = v
  end
end