class HSplitContainer < SplitContainer
  def initialize(source = nil)
    init_source source
    init_inner_control_vars
  end

  def current_godot_object
    "VSplitContainer"
  end
end
