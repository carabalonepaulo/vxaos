class TabContainer < Container
  def initialize(source = nil)
    init_source source
    init_inner_control_vars
  end

  def current_godot_object
    "TabContainer"
  end

  def tab_align
    @source.tab_align
  end

  def tab_align=(v)
    @source.tab_align = v
  end

  def current_tab
    @source.current_tab
  end

  def current_tab=(v)
    @source.current_tab = v
  end

  def tabs_visible
    @source.tabs_visible
  end

  def tabs_visible=(v)
    @source.tabs_visible = v
  end

  def all_tabs_in_front
    @source.all_tabs_in_front
  end

  def all_tabs_in_front=(v)
    @source.all_tabs_in_front = v
  end

  def drag_to_rearrange_enabled
    @source.drag_to_rearrange_enabled
  end

  def drag_to_rearrange_enabled=(v)
    @source.drag_to_rearrange_enabled = v
  end

  def use_hidden_tabs_for_min_size
    @source.use_hidden_tabs_for_min_size
  end

  def use_hidden_tabs_for_min_size=(v)
    @source.use_hidden_tabs_for_min_size = v
  end

  def get_current_tab_control
    Control.new @source.get_current_tab_control
  end

  def get_popup
    # TODO: Popup.new @source.get_popup
  end

  def get_previous_tab
    @source.get_previous_tab
  end

  def get_tab_control(tab_idx)
    Control.new @source.get_tab_control(tab_idx)
  end

  def get_tab_count
    @source.get_tab_count
  end

  def get_tab_disabled(tab_idx)
    @source.get_tab_disabled tab_idx
  end

  def get_tab_hidden(tab_idx)
    @source.get_tab_hidden tab_idx
  end

  def get_tab_icon(tab_idx)
    @source.get_tab_icon tab_idx
  end

  def get_tab_idx_at_point(point)
    @source.get_tab_idx_at_point point.source
  end

  def get_tab_title(tab_idx)
    @source.get_tab_title tab_idx
  end

  def get_tabs_rearrange_group
    @source.get_tabs_rearrange_group
  end

  def set_popup(popup)
    @source.set_popup popup.source
  end

  def set_tab_disabled(tab_idx, disabled)
    @source.set_tab_disabled tab_idx, disabled
  end

  def set_tab_hidden(tab_idx, hidden)
    @source.set_tab_hidden tab_idx, hidden
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