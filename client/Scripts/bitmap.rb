class Bitmap
  def initialize(w, h = nil)
    raise 'not implemented'
  end
  
  def width
    raise 'not implemented'
  end
  
  def height
    raise 'not implemented'
  end
  
  def rect
    raise 'not implemented'
  end
  
  def font=(v)
    raise 'not implemented'
  end
  
  def font
    raise 'not implemented'
  end
  
  def clear
    raise 'not implemented'
  end
  
  def clear_rect
    raise 'not implemented'
  end
  
  def hue_change(hue)
    raise 'not implemented'
  end
  
  def blur
    raise 'not implemented'
  end
  
  def radial_blur(angle, division)
    raise 'not implemented'
  end
  
  def draw_text(x, y, width, height, str, align = 0)
    raise 'not implemented'
  end
  
  def text_size(str)
    raise 'not implemented'
  end
  
  def fill_rect(x, y, width, height, color)
    raise 'not implemented'
  end
  
  def blt(x, y, src_bitmap, src_rect, opacity = 255)
    raise 'not implemented'
  end
  
  def stretch_blt(dest_rect, src_bitmap, src_rect, opacity = 255)
    raise 'not implemented'
  end
  
  def gradient_fill_rect(x, y, width, height, color1, color2, vertical = true)
    raise 'not implemented'
  end
  
  def get_pixel(x, y)
    raise 'not implemented'
  end
  
  def set_pixel(x, y, color)
    raise 'not implemented'
  end
  
  def disposed?
    raise 'not implemented'
  end
  
  def dispose
    raise 'not implemented'
  end
end