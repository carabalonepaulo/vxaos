class GraphNode < Container
  def initialize(source = nil)
    init_source source
    init_inner_control_vars
  end

  def current_godot_object
    "GraphNode"
  end

  def init_inner_control_vars
    super
    @offset = Vector2.new @source.offset
  end

  def title
    @source.title
  end

  def title=(v)
    @source.title = v
  end

  def offset
    @offset
  end

  def offset=(v)
    @offset = v
    @source.offset = v.source
  end

  def show_close
    @source.show_close
  end

  def show_close=(v)
    @source.show_close = v
  end

  def resizable
    @source.resizable
  end

  def resizable=(v)
    @source.resizable = v
  end

  def selected
    @source.selected
  end

  def selected=(v)
    @source.selected = v
  end

  def comment
    @source.comment
  end

  def comment=(v)
    @source.comment = v
  end

  def overlay
    @source.overlay
  end

  def overlay=(v)
    @source.overlay = v
  end

  def clear_all_slots
    @source.clear_all_slots
  end

  def clear_slot(idx)
    @source.clear_slot idx
  end

  def get_connection_input_color(idx)
    Color.new @source.get_connection_input_color(idx)
  end

  def get_connection_input_count
    @source.get_connection_input_count
  end

  def get_connection_input_position(idx)
    Vector2.new @source.get_connection_input_position(idx)
  end

  def get_connection_input_type(idx)
    @source.get_connection_input_type idx
  end

  def get_connection_output_color(idx)
    Color.new @source.get_connection_output_color(idx)
  end

  def get_connection_output_count
    @source.get_connection_output_count
  end

  def get_connection_output_position(idx)
    Vector2.new @source.get_connection_output_position(idx)
  end

  def get_connection_output_type(idx)
    @source.get_connection_output_type idx
  end

  def get_slot_color_left(idx)
    Color.new @source.get_slot_color_left(idx)
  end

  def get_slot_color_right(idx)
    Color.new @source.get_slot_color_right(idx)
  end

  def get_slot_type_left(idx)
    @source.get_slot_type_left idx
  end

  def get_slot_type_right(idx)
    @source.get_slot_type_right idx
  end

  def is_slot_enabled_left?(idx)
    @source.is_slot_enabled_left idx
  end

  def is_slot_enabled_right?(idx)
    @source.is_slot_enabled_right idx
  end

  def set_slot(idx, enable_left, type_left, color_left, enable_right, type_right, color_right, custom_left = nil, custom_right = nil)
    @source.set_slot idx, enable_left, type_left, color_left.source, enable_right, type_right, color_right.source, custom_left, custom_right
  end

  def set_slot_color_left(idx, color_left)
    @source.set_slot_color_left idx, color_left.source
  end

  def set_slot_color_right(idx, color_right)
    @source.set_slot_color_right idx, color_right.source
  end

  def set_slot_enabled_left(idx, enable_left)
    @source.set_slot_enabled_left idx, enable_left
  end

  def set_slot_enabled_right(idx, enable_right)
    @source.set_slot_enabled_right idx, enable_right
  end

  def set_slot_type_left(idx, type_left)
    @source.set_slot_type_left idx, type_left
  end

  def set_slot_type_right(idx, type_right)
    @source.set_slot_type_right idx, type_right
  end
end