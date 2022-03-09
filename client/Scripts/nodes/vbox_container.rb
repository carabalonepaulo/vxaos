class VBoxContainer < BoxContainer
  def initialize(source = nil)
    init_source source
    init_inner_control_vars
  end

  def current_godot_object
    "VBoxContainer"
  end
end