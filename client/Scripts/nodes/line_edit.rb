class LineEdit < Control
  def initialize(source = nil)
    init_source source
    init_inner_control_vars
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

  def max_length
    @source.max_length
  end

  def max_length=(v)
    @source.max_length = v
  end

  def editable
    @source.editable
  end

  def editable=(v)
    @source.editable = v
  end

  def secret
    @source.secret
  end

  def secret=(v)
    @source.secret = v
  end

  def secret_character
    @source.secret_character
  end

  def secret_character=(v)
    @source.secret_character = v
  end

  def expand_to_text_length
    @source.expand_to_text_length
  end

  def expand_to_text_length=(v)
    @source.expand_to_text_length = v
  end

  def context_menu_enabled
    @source.context_menu_enabled
  end

  def context_menu_enabled=(v)
    @source.context_menu_enabled = v
  end

  def virtual_keyboard_enabled
    @source.virtual_keyboard_enabled
  end

  def virtual_keyboard_enabled=(v)
    @source.virtual_keyboard_enabled = v
  end

  def clear_button_enabled
    @source.clear_button_enabled
  end

  def clear_button_enabled=(v)
    @source.clear_button_enabled = v
  end

  def shortcut_keys_enabled
    @source.shortcut_keys_enabled
  end

  def shortcut_keys_enabled=(v)
    @soruce.shortcut_keys_enabled = v
  end

  def selecting_enabled
    @source.selecting_enabled
  end

  def selecting_enabled=(v)
    @source.selecting_enabled = v
  end

  def right_icon
    @source.right_icon
  end

  def right_icon=(v)
    @source.right_icon = v
  end

  def placeholder_text
    @source.placeholder_text
  end

  def placeholder_text=(v)
    @source.placeholder_text = v
  end

  def placeholder_alpha
    @source.placeholder_alpha
  end

  def placeholder_alpha=(v)
    @source.placeholder_alpha = v
  end

  def caret_blink
    @source.caret_blink
  end

  def caret_blink=(v)
    @source.caret_blink = v
  end

  def caret_blink_speed
    @source.caret_blink_speed
  end

  def caret_blink_speed=(v)
    @source.caret_blink_speed = v
  end

  def caret_position
    @source.caret_position
  end

  def caret_position=(v)
    @source.caret_position = v
  end

  def focus_mode
    @source.focus_mode
  end

  def focus_mode=(v)
    @source.focus_mode = v
  end

  def append_at_cursor(text)
    @source.append_at_cursor text
  end

  def clear
    @source.clear
  end

  def delete_char_at_cursor
    @source.delete_char_at_cursor
  end

  def delete_text(from_column, to_column)
    @source.delete_text from_column, to_column
  end

  def deselect
    @source.deselect
  end

  def get_menu
    # TODO: PopupMenu.new @source.get_menu
  end

  def get_scroll_offset
    @source.get_scroll_offset
  end

  def menu_option(option)
    @source.menu_option option
  end

  def select(from, to = -1)
    @source.select from, to
  end

  def select_all
    @source.select_all
  end
end