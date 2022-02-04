class Tabs < Control
  def initialize(source = nil)
    init_source source
    init_inner_control_vars
  end

  def current_tab
    @source.current_tab
  end

  def current_tab=(v)
    @source.current_tab = v
  end

  def tab_align
    @source.tab_align
  end

  def tab_align=(v)
    @source.tab_align = v
  end

  def tab_close_display_policy
    @source.tab_close_display_policy
  end

  def tab_close_display_policy=(v)
    @source.tab_close_display_policy = v
  end

  def scrolling_enabled
    @source.scrolling_enabled
  end

  def scrolling_enabled=(v)
    @source.scrolling_enabled = v
  end

  def drag_to_rearrange_enabled
    @source.drag_to_rearrange_enabled
  end

  def drag_to_rearrange_enabled=(v)
    @source.drag_to_rearrange_enabled = v
  end

  def add_tab(title = "", icon = nil)
    @source.add_tab title, icon
  end

  def ensure_tab_visible(idx)
    @source.ensure_tab_visible idx
  end

  def get_offset_buttons_visible
    @source.get_offset_buttons_visible
  end

  def get_previous_tab
    @source.get_previous_tab
  end

  def get_select_with_rmb
    @source.get_select_with_rmb
  end

  def get_tab_count
    @source.get_tab_count
  end

  def get_tab_disabled(tab_idx)
    @source.get_tab_disabled tab_idx
  end

  def get_tab_icon(tab_idx)
    @source.get_tab_icon tab_idx
  end

  def get_tab_offset
    @source.get_tab_offset
  end

  def get_tab_rect(tab_idx)
    Rect.new @source.get_tab_rect tab_idx
  end

  def get_tab_title(tab_idx)
    @source.get_tab_title tab_idx
  end

  def get_tabs_rearrange_group
    @source.get_tabs_rearrange_group
  end

  def move_tab(from, to)
    @source.move_tab from, to
  end

  def remove_tab(tab_idx)
    @source.remove_tab tab_idx
  end

  def set_select_with_rmb(enabled)
    @source.set_select_with_rmb enabled
  end

  def set_tab_disabled(tab_idx, disabled)
    @source.set_tab_disabled tab_idx, disabled
  end

  def set_tab_icon(tab_idx, icon)
    @source.set_tab_icon tab_idx, icon
  end

  def set_tab_title(tab_idx, title)
    @source.set_tab_title tab_idx, title
  end

  def set_tabs_rearrange_group(group_id)
    @source.set_tabs_rearrange_group group_id
  end
end