class GraphEdit < Control
  def initialize(source = nil)
    init_source source
    init_inner_control_vars
  end

  def current_godot_object
    "GraphEdit"
  end

  def right_disconnects
    @source.right_disconnects
  end

  def right_disconnects=(v)
    @source.right_disconnects = v
  end

  def scroll_offset
    @source.scroll_offset
  end

  def scroll_offset=(v)
    @source.scroll_offset = v
  end

  def snap_distance
    @source.snap_distance
  end

  def snap_distance=(v)
    @source.snap_distance = v
  end

  def use_snap
    @source.use_snap
  end

  def use_snap=(v)
    @source.use_snap = v
  end

  def zoom
    @source.zoom
  end

  def zoom=(v)
    @source.zoom = v
  end

  def zoom_min
    @source.zoom_min
  end

  def zoom_min=(v)
    @source.zoom_min = v
  end

  def zoom_max
    @source.zoom_max
  end

  def zoom_max=(v)
    @source.zoom_max
  end

  def zoom_step
    @source.zoom_step
  end

  def zoom_step=(v)
    @source.zoom_step = v
  end

  def show_zoom_label
    @source.show_zoom_label
  end

  def show_zoom_label=(v)
    @source.show_zoom_label = v
  end

  def minimap_enabled
    @source.minimap_enabled
  end

  def minimap_enabled=(v)
    @source.minimap_enabled
  end

  def minimap_size
    @source.minimap_size
  end

  def minimap_size=(v)
    @source.minimap_size = v
  end

  def minimap_opacity
    @source.minimap_opacity
  end

  def minimap_opacity=(v)
    @source.minimap_opacity = v
  end

  def add_valid_connection_type(from_type, to_type)
    @source.add_valid_connection_type from_type, to_type
  end

  def add_valid_left_disconnect_type(type)
    @source.add_valid_left_disconnect_type type
  end

  def add_valid_right_disconnect_type(type)
    @source.add_valid_right_disconnect_type type
  end

  def clear_connections
    @source.clear_connections
  end

  def connect_node(from, from_port, to, to_port)
    @source.connect_node from, from_port, to, to_port
  end

  def disconnect_node(from, from_port, to, to_port)
    @source.disconnect_node from, from_port, to, to_port
  end

  def get_connection_list
    @source.get_connection_list
  end

  def get_zoom_hbox
    HBoxContainer.new @source.get_zoom_hbox
  end

  def is_node_connected?(from, from_port, to, to_port)
    @source.is_node_connected from, from_port, to, to_port
  end

  def is_valid_connection_type?(from_type, to_type)
    @source.is_valid_connection_type from_type, to_type
  end

  def remove_valid_connection_type(from_type, to_type)
    @source.remove_valid_connection_type from_type, to_type
  end

  def remove_valid_left_disconnect_type(type)
    @source.remove_valid_left_disconnect_type type
  end

  def remove_valid_rigth_disconnect_type(type)
    @source.remove_valid_rigth_disconnect_type type
  end

  def set_connection_activity(from, from_port, to, to_port, amount)
    @source.set_connection_activity from, from_port, to, to_port, amount
  end

  def set_selected(node)
    @source.set_selected node
  end
end