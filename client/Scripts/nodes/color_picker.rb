class ColorPicker < BoxContainer
  def initialize(source=nil)
    init_source source
    init_inner_control_vars
  end

  def current_godot_object
    "ColorPicker"
  end

  def init_inner_control_vars
    super
    @color = Color.new @source.color
  end

  def color
    @color
  end

  def color=(v)
    @color = v
    @source.color = v.source
  end

  def edit_alpha
    @source.edit_alpha
  end

  def edit_alpha=(v)
    @source.edit_alpha = v
  end

  def hsv_mode
    @source.hsv_mode
  end

  def hsv_mode=(v)
    @source.hsv_mode = v
  end

  def raw_mode
    @source.raw_mode
  end

  def raw_mode=(v)
    @source.raw_mode = v
  end

  def deferred_mode
    @source.deferred_mode
  end

  def deferred_mode=(v)
    @source.deferred_mode = v
  end

  def presets_enabled
    @source.presets_enabled
  end

  def presets_enabled=(v)
    @source.presets_enabled = v
  end

  def presets_visible
    @source.presets_visible
  end

  def presets_visible=(v)
    @source.presets_visible = v
  end

  def add_preset(color)
    @source.add_preset color.source
  end

  def erase_preset(color)
    @source.erase_preset color.source
  end

  def get_presets
    @source.get_presets.map { |color| Color.new color }
  end
end