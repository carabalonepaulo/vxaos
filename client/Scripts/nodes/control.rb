class Control < CanvasItem
  def initialize(source = nil)
    init_source source
    init_inner_control_vars
  end

  def init_inner_control_vars
    super
    @rect_position = Vector2.new @source.rect_position
    @rect_size = Vector2.new @source.rect_size
    @rect_min_size = Vector2.new @source.rect_min_size
    @rect_scale = Vector2.new @source.rect_scale
    @rect_pivot_offset = Vector2.new @source.rect_pivot_offset
  end

  def anchor_left
    @source.anchor_left
  end

  def anchor_left=(v)
    @source.anchor_left = v
  end

  def anchor_top
    @source.anchor_top
  end

  def anchor_top=(v)
    @source.anchor_top = v
  end

  def anchor_right
    @source.anchor_right
  end

  def anchor_right=(v)
    @source.anchor_right = v
  end

  def anchor_bottom
    @source.anchor_bottom
  end

  def anchor_bottom=(v)
    @source.anchor_bottom = v
  end

  def margin_left
    @source.margin_left
  end

  def margin_left=(v)
    @source.margin_left = v
  end

  def margin_top
    @source.margin_top
  end

  def margin_top=(v)
    @source.margin_top = v
  end

  def margin_right
    @source.margin_right
  end

  def margin_right=(v)
    @source.margin_right = v
  end

  def margin_bottom
    @source.margin_bottom
  end

  def margin_bottom=(v)
    @source.margin_bottom = v
  end

  def grow_horizontal
    @source.grow_horizontal
  end

  def grow_horizontal=(v)
    @source.grow_horizontal = v
  end

  def grow_vertical
    @source.grow_vertical
  end

  def grow_vertical=(v)
    @source.grow_vertical = v
  end

  def rect_position
    @rect_position
  end

  def rect_position=(v)
    @rect_position = v
    @source.rect_position = v.source
  end

  def rect_global_position
    @rect_global_position
  end

  def rect_global_position=(v)
    @rect_global_position = v
    @source.rect_global_position = v.source
  end

  def rect_size
    @rect_size
  end

  def rect_size=(v)
    @rect_size = v
    @source.rect_size = v.source
  end

  def rect_min_size
    @rect_min_size
  end

  def rect_min_size=(v)
    @rect_min_size = v
    @source.rect_min_size = v.source
  end

  def rect_rotation
    @source.rect_rotation
  end

  def rect_rotation=(v)
    @source.rect_rotation = v
  end

  def rect_scale
    @rect_scale
  end

  def rect_scale=(v)
    @rect_scale = v
    @source.rect_scale = v.source
  end

  def rect_pivot_offset
    @rect_pivot_offset
  end

  def rect_pivot_offset=(v)
    @rect_pivot_offset = v
    @source.rect_pivot_offset = v.source
  end

  def rect_clip_content
    @source.rect_clip_content
  end

  def rect_clip_content=(v)
    @source.rect_clip_content = v
  end

  def hint_tooltip
    @source.hint_tooltip
  end

  def hint_tooltip=(v)
    @source.hint_tooltip = v
  end

  def focus_neighbour_left
    raise 'not implemented yet'
  end

  def focus_neighbour_left=(v)
    raise 'not implemented yet'
  end

  def focus_neighbour_top
    raise 'not implemented yet'
  end

  def focus_neighbour_top=(v)
    raise 'not implemented yet'
  end

  def focus_neighbour_right
    raise 'not implemented yet'
  end

  def focus_neighbour_right=(v)
    raise 'not implemented yet'
  end

  def focus_neighbour_bottom
    raise 'not implemented yet'
  end

  def focus_neighbour_bottom=(v)
    raise 'not implemented yet'
  end

  def focus_next
    raise 'not implemented yet'
  end

  def focus_next=(v)
    raise 'not implemented yet'
  end

  def focus_previous
    raise 'not implemented yet'
  end

  def focus_previous=(v)
    raise 'not implemented yet'
  end

  def focus_mode
    @source.focus_mode
  end

  def focus_mode=(v)
    @source.focus_mode
  end

  def mouse_filter
    @source.mouse_filter
  end

  def mouse_filter=(v)
    @source.mouse_filter = v
  end

  def mouse_default_cursor_shape
    @source.mouse_default_cursor_shape
  end

  def mouse_default_cursor_shape=(v)
    @source.mouse_default_cursor_shape = v
  end

  def input_pass_on_modal_close_click
    @source.input_pass_on_modal_close_click
  end

  def input_pass_on_modal_close_click=(v)
    @source.input_pass_on_modal_close_click = v
  end

  def size_flags_horizontal
    @source.size_flags_horizontal
  end

  def size_flags_horizontal=(v)
    @source.size_flags_horizontal = v
  end

  def size_flags_vertical
    @source.size_flags_vertical
  end

  def size_flags_vertical=(v)
    @source.size_flags_vertical = v
  end

  def size_flags_stretch_ratio
    @source.size_flags_stretch_ratio
  end

  def size_flags_stretch_ratio=(v)
    @source.size_flags_stretch_ratio = v
  end

  def theme
    raise 'not implemented yet'
  end

  def theme=(v)
    raise 'not implemented yet'
  end

  def _clips_input
  end

  def _get_minimum_size
  end

  def _gui_input(event)
  end

  def _make_custom_tooltip(for_text)
  end

  def can_drop_data(position, data)
  end

  def drop_data(position, data)
  end

  def get_drag_data(position)
  end

  def has_point(point)
  end
end