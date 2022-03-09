class NinePatchRect < Control
  def initialize(source = nil)
    init_source source
    init_inner_control_vars
  end

  def current_godot_object
    "NinePatchRect"
  end

  def init_inner_control_vars
    super
    @region_rect = Rect.new @source.region_rect
  end

  def texture
    @source.texture
  end

  def texture=(v)
    @source.texture = v
  end

  def draw_center
    @source.draw_center
  end

  def draw_center=(v)
    @source.draw_center = v
  end

  def region_rect
    @region_rect
  end

  def region_rect=(v)
    @region_rect = v
    @source.region_rect = v.source
  end

  def patch_margin_left
    @source.patch_margin_left
  end

  def patch_margin_left=(v)
    @source.patch_margin_left = v
  end

  def patch_margin_top
    @source.patch_margin_top
  end

  def patch_margin_top=(v)
    @source.patch_margin_top = v
  end

  def patch_margin_right
    @source.patch_margin_right
  end

  def patch_margin_right=(v)
    @source.patch_margin_right = v
  end

  def patch_margin_bottom
    @source.patch_margin_bottom
  end

  def patch_margin_bottom=(v)
    @source.patch_margin_bottom = v
  end

  def axis_stretch_horizontal
    @source.axis_stretch_horizontal
  end

  def axis_stretch_horizontal=(v)
    @source.axis_stretch_horizontal = v
  end

  def axis_stretch_vertical
    @source.axis_stretch_vertical
  end

  def axis_stretch_vertical=(v)
    @source.axis_stretch_vertical = v
  end

  def get_patch_margin(margin)
    @source.get_patch_margin
  end

  def set_patch_margin(margin, value)
    @source.set_patch_margin margin, value
  end
end