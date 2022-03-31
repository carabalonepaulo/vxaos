class Font
  def get_string_size(string)
    @source.get_string_size string
  end
end

class DynamicFont < Font
  attr_accessor :source

  def initialize(source = nil)
    if source.class.to_s == 'Godot::DynamicFont'
      @source = source
    else
      @source = $system.create_godot_object 'DynamicFont'
    end
  end

  def size
    @source.size
  end

  def size=(v)
    @source.size = v
  end

  def outline_size
    @source.outline_size
  end

  def outline_size=(v)
    @source.outline_size = v
  end

  def outline_color
    @source.outline_color
  end

  def outline_color=(v)
    @source.outline_color = v.source
  end

  def use_mipmaps
    @source.use_mipmaps
  end

  def use_mipmaps=(v)
    @source.use_mipmaps = v
  end

  def use_filter
    @source.use_filter
  end

  def use_filter=(v)
    @source.use_filter = v
  end

  def extra_spacing_top
    @source.extra_spacing_top
  end

  def extra_spacing_top=(v)
    @source.extra_spacing_top = v
  end

  def extra_spacing_bottom
    @source.extra_spacing_bottom
  end

  def extra_spacing_bottom=(v)
    @source.extra_spacing_bottom = v
  end

  def extra_spacing_char
    @source.extra_spacing_char
  end

  def extra_spacing_char=(v)
    @source.extra_spacing_char = v
  end

  def extra_spacing_space
    @source.extra_spacing_space
  end

  def extra_spacing_space=(v)
    @soure.extra_spacing_space = v
  end

  def font_data
    @source.font_data
  end

  def font_data=(dynamic_font_data)
    @source.font_data = dynamic_font_data
  end

  def add_fallback(dynamic_font_data)
    @source.add_fallback dynamic_font_data
  end

  def get_available_chars
    @source.get_available_chars
  end

  def get_fallback(idx)
    @source.get_fallback idx
  end

  def get_fallback_count
    @source.get_fallback_count
  end

  def get_spacing(type)
    @source.get_spacing type
  end

  def remove_fallback(idx)
    @source.remove_fallback idx
  end

  def set_fallback(idx, dynamic_font_data)
    @source.set_fallback idx, dynamic_font_data
  end

  def set_spacing(type, value)
    @source.set_spacing type, value
  end
end
