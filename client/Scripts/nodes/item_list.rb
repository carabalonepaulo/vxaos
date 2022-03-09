class ItemList < Control
  def initialize(source = nil)
    init_source source
    init_inner_control_vars
  end

  def current_godot_object
    "ItemList"
  end

  def init_inner_control_vars
    super
    @fixed_icon_size = Vector2.new @source.fixed_icon_size
    @vscroll = VScrollBar.new @source.get_v_scroll
  end

  def select_mode
    @source.select_mode
  end

  def select_mode=(v)
    @source.select_mode = v
  end

  def allow_reselect
    @source.allow_reselect
  end

  def allow_reselect=(v)
    @source.allow_reselect = v
  end

  def allow_rmb_select
    @source.allow_rmb_select
  end

  def allow_rmb_select=(v)
    @source.allow_rmb_select = v
  end

  def max_text_lines
    @source.max_text_lines
  end

  def max_text_lines=(v)
    @source.max_text_lines = v
  end

  def auto_height
    @source.auto_height
  end

  def auto_height=(v)
    @source.auto_height = v
  end

  def max_columns
    @source.max_columns
  end

  def max_columns=(v)
    @source.max_columns = v
  end

  def same_column_width
    @source.same_column_width
  end

  def same_column_width=(v)
    @source.same_column_width = v
  end

  def fixed_column_width
    @source.fixed_column_width
  end

  def fixed_column_width=(v)
    @source.fixed_column_width = v
  end

  def icon_mode
    @source.icon_mode
  end

  def icon_mode=(v)
    @source.icon_mode = v
  end

  def icon_scale
    @source.icon_scale
  end

  def icon_scale=(v)
    @source.icon_scale = v
  end

  def fixed_icon_size
    @fixed_icon_size
  end

  def fixed_icon_size=(v)
    @fixed_icon_size = v
    @source.fixed_icon_size = v.source
  end

  def rect_clip_content
    @source.rect_clip_content
  end

  def rect_clip_content=(v)
    @source.rect_clip_content = v
  end

  def focus_mode
    @source.focus_mode
  end

  def focus_mode=(v)
    @source.focus_mode = v
  end

  def add_icon_item(icon, selectable = true)
    @source.add_icon_item icon, selectable
  end

  def add_item(text, texture = nil, selectable = true)
    @source.add_item text, texture, selectable
  end

  def clear
    @source.clear
  end

  def ensure_current_is_visible
    @source.ensure_current_is_visible
  end

  def get_item_at_position(position, exact = false)
    @source.get_item_at_position position.source, exact
  end

  def get_item_count
    @source.get_item_count
  end

  def get_item_custom_bg_color(idx)
    Color.new @source.get_item_custom_bg_color(idx)
  end

  def get_item_custom_fg_color(idx)
    Color.new @source.get_item_custom_fg_color(idx)
  end

  def get_item_icon(idx)
    @source.get_item_icon idx
  end

  def get_item_icon_modulate(idx)
    Color.new @source.get_item_icon_modulate(idx)
  end

  def get_item_icon_region(idx)
    @source.get_item_icon_region idx
  end

  def get_item_metadata(idx)
    @source.get_item_metadata idx
  end

  def get_item_text(idx)
    @source.get_item_text idx
  end

  def get_item_tooltip(idx)
    @source.get_item_tooltip idx
  end

  def get_selected_items
    @source.get_selected_items
  end

  def get_v_scroll
    @vscroll
  end

  def is_anything_selected?
    @source.is_anything_selected
  end

  def is_item_disabled?(idx)
    @source.is_item_disabled idx
  end

  def is_item_icon_transposed?(idx)
    @source.is_item_icon_transposed idx
  end

  def is_item_selectable?(idx)
    @source.is_item_selectable idx
  end

  def is_item_tooltip_enabled?(idx)
    @source.is_item_tooltip_enabled idx
  end

  def is_selected?(idx)
    @source.is_selected idx
  end

  def move_item(from_idx, to_idx)
    @source.move_item from_idx, to_idx
  end

  def remove_item(idx)
    @source.remove_item idx
  end

  def select(idx, single = true)
    @source.select idx, single
  end

  def set_item_custom_bg_color(idx, custom_bg_color)
    @source.set_item_custom_bg_color idx, custom_bg_color.source
  end

  def set_item_custom_fg_color(idx, custom_fg_color)
    @source.set_item_custom_fg_color idx, custom_fg_color.source
  end

  def set_item_disabled(idx, disabled)
    @source.set_item_disabled idx, disabled
  end

  def set_item_icon(idx, icon)
    @source.set_item_icon idx, icon
  end

  def set_item_icon_modulate(idx, color)
    @source.set_item_icon_modulate idx, color.source
  end

  def set_item_icon_region(idx, rect)
    @source.set_item_icon_region idx, rect.source
  end

  def set_item_icon_transposed(idx, transposed)
    @source.set_item_icon_transposed idx, transposed
  end

  def set_item_metadata(idx, metadata)
    @source.set_item_metadata idx, metadata
  end

  def set_item_selectable(idx, selectable)
    @source.set_item_selectable idx, selectable
  end

  def set_item_text(idx, text)
    @source.set_item_text idx, text
  end

  def set_item_tooltip(idx, tooltip)
    @source.set_item_tooltip idx, tooltip
  end

  def set_item_tooltip_enabled(idx, enabled)
    @source.set_item_tooltip_enabled idx, enabled
  end

  def sort_items_by_text
    @source.sort_items_by_text
  end

  def unselect(idx)
    @source.unselect idx
  end

  def unselect_all
    @source.unselect_all
  end
end