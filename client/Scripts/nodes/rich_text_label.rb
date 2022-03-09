class RichTextLabel < Control
  def initialize(source = nil)
    init_source source
    init_inner_control_vars
  end

  def current_godot_object
    "RichTextLabel"
  end

  def bbcode_enabled
    @source.bbcode_enabled
  end

  def bbcode_enabled=(v)
    @source.bbcode_enabled = v
  end

  def bbcode_text
    @source.bbcode_text
  end

  def bbcode_text=(v)
    @source.bbcode_text = v
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

  def meta_underlined
    @source.meta_underlined
  end

  def meta_underlined=(v)
    @source.meta_underlined = v
  end

  def tab_size
    @source.tab_size
  end

  def tab_size=(v)
    @source.tab_size = v
  end

  def text
    @source.text
  end

  def text=(v)
    @source.text = v
  end

  def fit_content_height
    @source.fit_content_height
  end

  def fit_content_height=(v)
    @source.fit_content_height = v
  end

  def scroll_active
    @source.scroll_active
  end

  def scroll_active=(v)
    @source.scroll_active = v
  end

  def scroll_following
    @source.scroll_following
  end

  def scroll_following=(v)
    @source.scroll_following = v
  end

  def selection_enabled
    @source.selection_enabled
  end

  def selection_enabled=(v)
    @source.selection_enabled = v
  end

  def override_selected_font_color
    @source.override_selected_font_color
  end

  def override_selected_font_color=(v)
    @source.override_selected_font_color = v
  end

  def custom_effects
    raise 'not implemented yet'
  end

  def rect_clip_content
    @source.rect_clip_content
  end

  def rect_clip_content=(v)
    @source.rect_clip_content = v
  end

  def add_image(image, width = 0, height = 0)
    @source.add_image image, width, height
  end

  def add_text(text)
    @source.add_text text
  end

  def append_bbcode(bbcode)
    @source.append_bbcode bbcode
  end

  def clear
    @source.clear
  end

  def get_content_height
    @source.get_content_height
  end

  def get_line_count
    @source.get_line_count
  end

  def get_total_character_count
    @source.get_total_character_count
  end

  def get_v_scroll
    VScroll.new @source.get_v_scroll
  end

  def get_visible_line_count
    @source.get_visible_line_count
  end

  def install_effect(effect)
    raise 'not implemented yet'
  end

  def newline
    @source.newline
  end

  def parse_bbcode(bbcode)
    @source.parse_bbcode bbcode
  end

  def parse_expresions_for_values(expressions)
    @soruce.parse_expresions_for_values expressions
  end

  def pop
    @source.pop
  end

  def push_align(align)
    @source.push_align align
  end

  def push_bold
    @source.push_bold
  end

  def push_bold_italics
    @source.push_bold_italics
  end

  def push_cell
    @source.push_cell
  end

  def push_color(color)
    @source.push_color color.source
  end

  def push_font(font)
    @source.push_font font.source
  end

  def push_indent(level)
    @source.push_indent level
  end

  def push_italics
    @source.push_italics
  end

  def push_list(type)
    @source.push_list type
  end

  def push_meta(meta)
    @source.push_meta meta
  end

  def push_mono
    @source.push_mono
  end

  def push_normal
    @source.push_normal
  end

  def push_strikethrough
    @source.push_strikethrough
  end

  def push_table(columns)
    @source.push_table columns
  end

  def push_underline
    @source.push_underline
  end

  def remove_line(line)
    @source.remove_line line
  end

  def scroll_to_line(line)
    @source.scroll_to_line
  end

  def set_table_column_expand(column, expand, ratio)
    @source.set_table_column_expand column, expand, ratio
  end
end