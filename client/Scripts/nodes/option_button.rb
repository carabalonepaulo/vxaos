class OptionButton < Button
  def initialize(source = nil)
    init_source source
    init_inner_control_vars
  end

  def current_godot_object
    "OptionButton"
  end

  def selected
    @source.selected
  end

  def selected=(v)
    @source.selected = v
  end

  def align
    @source.align
  end

  def align=(v)
    @source.align = v
  end

  def toggle_mode
    @source.toggle_mode
  end

  def toggle_mode=(v)
    @source.toggle_mode = v
  end

  def action_mode
    @source.action_mode
  end

  def action_mode=(v)
    @source.action_mode = v
  end

  def add_icon_texture(texture, label, id)
    @source.add_icon_texture texture, label, id
  end

  def add_item(label, id)
    @source.add_item label, id
  end

  def add_separator
    @source.add_separator
  end

  def clear
    @source.clear
  end

  def get_item_count
    @source.get_item_count
  end

  def get_item_icon(idx)
    @source.get_item_icon idx
  end

  def get_item_id(idx)
    @source.get_item_id idx
  end

  def get_item_index(id)
    @source.get_item_index id
  end

  def get_item_metadata(idx)
    @source.get_item_metadata idx
  end

  def get_item_text(idx)
    @source.get_item_text idx
  end

  def get_popup
    # TODO: Popup.new @source.get_popup
  end

  def get_selected_id
    @source.get_selected_id
  end

  def get_selected_metadata
    @source.get_selected_metadata
  end

  def is_item_disabled?(idx)
    @source.is_item_disabled idx
  end

  def remove_item(idx)
    @source.remove_item idx
  end

  def select(idx)
    @source.select idx
  end

  def set_item_disabled(idx, disabled)
    @source.set_item_disabled idx, disabled
  end

  def set_item_icon(idx, texture)
    @source.set_item_icon idx, texture
  end

  def set_item_id(idx, id)
    @source.set_item_id idx, id
  end

  def set_item_metadata(idx, metadata)
    @source.set_item_metadata idx, metadata
  end

  def set_item_text(idx, text)
    @source.set_item_text idx, text
  end
end