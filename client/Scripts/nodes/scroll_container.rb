class ScrollContainer < Container
  def initialize(source = nil)
    init_source source
    init_inner_control_vars
  end

  def follow_focus
    @source.follow_focus
  end

  def follow_focus=(v)
    @source.follow_focus = v
  end

  def scroll_horizontal_enabled
    @source.scroll_horizontal_enabled
  end
  
  def scroll_horizontal_enabled=(v)
    @source.scroll_horizontal_enabled = v
  end

  def scroll_horizontal
    @source.scroll_horizontal
  end

  def scroll_horizontal=(v)
    @source.scroll_horizontal = v
  end
    
  def scroll_vertical_enabled
    @source.scroll_vertical_enabled
  end

  def scroll_vertical_enabled=(v)
    @source.scroll_vertical_enabled = v
  end

  def scroll_vertical
    @source.scroll_vertical
  end

  def scroll_vertical=(v)
    @source.scroll_vertical = v
  end

  def scroll_deadzone
    @source.scroll_deadzone
  end

  def scroll_deadzone=(v)
    @source.scroll_deadzone = v
  end

  def rect_clip_content
    @source.rect_clip_content
  end

  def rect_clip_content=(v)
    @source.rect_clip_content = v
  end

  def ensure_control_visible(control)
    @source.ensure_control_visible control.source
  end

  def get_h_scrollbar
    # TODO: HScrollBar.new @source.get_h_scrollbar
  end

  def get_v_scrollbar
    # TODO: VScrollBar.new @source.get_v_scrollbar
  end
end