class CanvasItem < Node
  def initialize
    raise "Can't create instance of CanvasItem."
  end

  def init_inner_control_vars
    super
    @modulate = Color.new @source.modulate
    @self_modulate = Color.new @source.self_modulate
    # TODO: @material = Material.new @source.material
  end

  def visible
    @source.visible
  end

  def visible=(v)
    @source.visible = v
  end

  def modulate
    @modulate
  end

  def modulate=(v)
    @modulate = v
    @source.modulate = v.source
  end

  def self_modulate
    @self_modulate
  end

  def self_modulate=(v)
    @self_modulate = v
    @source.self_modulate = v.source
  end

  def show_behind_parent
    @source.show_behind_parent
  end

  def show_behind_parent=(v)
    @source.show_behind_parent = v
  end

  def show_on_top
    @source.show_on_top
  end

  def show_on_top=(v)
    @source.show_on_top = v
  end

  def light_mask
    @source.light_mask
  end

  def light_mask=(v)
    @source.light_mask = v
  end

  def material
    raise 'not implemented yet'
  end

  def material=(v)
    raise 'not implemented yet'
  end

  def use_parent_material
    @source.use_parent_material
  end

  def use_parent_material=(v)
    @source.use_parent_material = v
  end

  def _draw
  end

  def draw_arc(center, radius, start_angle, end_angle, point_count, color, width = 1, antialiased = false)
  end

  def draw_char(font, position, char, _next, modulate = Color::White)
  end

  def draw_circle(position, radius, color)
  end

  def draw_colored_polygon
    raise 'not implemented yet'
  end

  def draw_line(from, to, color, width = 1, antialiased = false)
  end

  def draw_mesh
    raise 'not implemented yet'
  end

  def draw_multiline
    raise 'not implemented yet'
  end

  def draw_multiline_colors
    raise 'not implemented yet'
  end

  def draw_multimesh
    raise 'not implemented yet'
  end

  def draw_polygon
    raise 'not implemented yet'
  end

  def draw_polyline_colors
    raise 'not implemented yet'
  end

  def draw_primitive
    raise 'not implemented yet'
  end

  def draw_rect(rect, color, filled = true, width = 1, antialiased = false)
    @source.draw_rect rect.source, color.source, filled, width, antialiased
  end

  def draw_set_transform(position, rotation, scale)
  end

  def draw_string(font, position, text, modulate = Color::White, clip_w = -1)
  end

  def draw_style_box(style_box, rect)
    raise 'not implemented yet'
  end

  def draw_texture(texture, position, modulate = Color::White, normal_map = nil)
  end

  def draw_texture_rect(texture, rect, tile, modulate = Color::White, transpose = false, normal_map = nil)
  end

  def draw_texture_rect_region(texture, rect, src_rect, modulate = Color::White, transpose = false, normal_map = nil, clip_uv = true)
  end
end