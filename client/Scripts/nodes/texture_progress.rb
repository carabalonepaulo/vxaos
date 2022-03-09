class TextureProgress < SRange
  def initialize(source = nil)
    init_source source
    init_inner_control_vars
  end

  def current_godot_object
    "TextureProgress"
  end

  def init_inner_control_vars
    super
    @texture_progress_offset = Vector2.new @source.texture_progress_offset
    @radial_center_offset = Vector2.new @source.radial_center_offset
    @tint_under = Color.new @source.tint_under
    @tint_over = Color.new @source.tint_over
    @tint_progress = Color.new @source.tint_progress
  end

  def texture_under
    @source.texture_under
  end

  def texture_under=(v)
    @source.texture_under = v
  end

  def texture_over
    @source.texture_over
  end

  def texture_over=(v)
    @source.texture_over = v
  end

  def texture_progress
    @source.texture_progress
  end

  def texture_progress=(v)
    @source.texture_progress = v
  end

  def texture_progress_offset
    @texture_progress_offset
  end

  def texture_progress_offset=(v)
    @texture_progress_offset = v
    @source.texture_progress_offset = v.source
  end

  def fill_mode
    @source.fill_mode
  end

  def fill_mode=(v)
    @source.fill_mode = v
  end

  def tint_under
    @source.tint_under
  end

  def tint_under=(v)
    @tint_under = v
    @source.tint_under = v.source
  end

  def tint_over
    @source.tint_over
  end

  def tint_over=(v)
    @tint_over = v
    @source.tint_over = v.source
  end

  def tint_progress
    @tint_progress
  end

  def tint_progress=(v)
    @tint_progress = v
    @source.tint_progress = v.source
  end

  def radial_initial_angle
    @source.radial_initial_angle
  end

  def radial_initial_angle=(v)
    @source.radial_initial_angle = v
  end

  def radial_fill_degrees
    @source.radial_fill_degrees
  end

  def radial_fill_degrees=(v)
    @source.radial_fill_degrees = v
  end

  def radial_center_offset
    @radial_center_offset
  end

  def radial_center_offset=(v)
    @radial_center_offset = v
    @source.radial_center_offset = v.source
  end

  def nine_patch_stretch
    @source.nine_patch_stretch
  end

  def nine_patch_stretch=(v)
    @source.nine_patch_stretch = v
  end

  def stretch_margin_left
    @source.stretch_margin_left
  end

  def stretch_margin_left=(v)
    @source.stretch_margin_left = v
  end

  def stretch_margin_top
    @source.stretch_margin_top
  end

  def stretch_margin_top=(v)
    @source.stretch_margin_top = v
  end

  def stretch_margin_right
    @source.stretch_margin_right
  end

  def stretch_margin_right=(v)
    @source.stretch_margin_right = v
  end

  def stretch_margin_bottom
    @source.stretch_margin_bottom
  end

  def stretch_margin_bottom=(v)
    @source.stretch_margin_bottom = v
  end

  def mouse_filter
    @source.mouse_filter
  end

  def mouse_filter=(v)
    @source.mouse_filter = v
  end
end