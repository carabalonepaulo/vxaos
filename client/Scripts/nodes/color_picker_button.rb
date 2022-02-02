class ColorPickerButton < Button
  def initialize(source=nil)
    init_source source
    init_inner_control_vars
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

  def toggle_mode
    @source.toggle_mode
  end

  def toggle_mode=(v)
    @source.toggle_mode = v
  end

  def get_picker
    ColorPicker.new @source.get_picker
  end

  def get_popup
    # TODO: PopupPanel.new @source.get_popup
  end
end