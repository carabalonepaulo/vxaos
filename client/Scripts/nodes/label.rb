class Label < Control
  def initialize(source = nil)
    init_source source
    init_inner_control_vars
  end

  def current_godot_object
    "Label"
  end

  def text
    @source.text
  end

  def text=(v)
    @source.text = v
  end

  def align
    @source.align
  end

  def align=(v)
    @source.align = v
  end

  def valign
    @source.valign
  end

  def valign=(v)
    @source.valign = v
  end

  def autowrap
    @source.autowrap
  end

  def autowrap=(v)
    @source.autowrap = v
  end

  def clip_text
    @source.clip_text
  end

  def clip_text=(v)
    @source.clip_text = v
  end

  def uppercase
    @source.uppercase
  end

  def uppercase=(v)
    @source.uppercase = v
  end

  def visible_characters
    @source.visible_characters
  end

  def visible_characters=(v)
    @source.visible_characters = v
  end

  def percent_visible
    @source.percent_visible
  end

  def percent_visible=(v)
    @source.percent_visible = v
  end

  def lines_skipped
    @source.lines_skipped
  end

  def lines_skipped=(v)
    @source.lines_skipped = v
  end

  def max_lines_visible
    @source.max_lines_visible
  end

  def max_lines_visible=(v)
    @source.max_lines_visible = v
  end

  def mouse_filter
    @source.mouse_filter
  end

  def mouse_filter=(v)
    @source.mouse_filter = v
  end

  def size_flags_vertical
    @source.size_flags_vertical
  end

  def size_flags_vertical=(v)
    @source.size_flags_vertical = v
  end

  def get_line_count
    @source.get_line_count
  end

  def get_line_height
    @source.get_line_height
  end

  def get_total_character_count
    @source.get_total_character_count
  end

  def get_visible_line_count
    @source.get_visible_line_count
  end
end