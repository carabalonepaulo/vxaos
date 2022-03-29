class Node2D < CanvasItem
  def initialize(source = nil)
    init_source source
    init_inner_control_vars
  end

  def current_godot_object
    "Node2D"
  end
  
  def init_inner_control_vars
    super
    @position = Vector2.new @source.position
  end

  def position
    @position
  end

  def position=(v)
    @position = v
    @source.position = v.source
  end
end
