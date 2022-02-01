class CenterContainer < Container
  def initialize(source = nil)
    init_source source
    init_inner_control_vars
  end

  def use_top_left
    @source.use_top_left
  end

  def use_top_left=(v)
    @source.use_top_left = v
  end
end