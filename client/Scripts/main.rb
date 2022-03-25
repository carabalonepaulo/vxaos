class CustomNode < Node2D
  def _ready
    @texture = $system.resource_loader.load("res://001-Fighter01.png")
    @dest_rect = Rect.new(300, 100, 32, 48)
    @src_rect = Rect.new(0, 0, 32, 48)
    @font = DynamicFont.new
    @font.font_data = $system.resource_loader.load("res://Fonts/Roboto/Roboto-Regular.ttf")
  end

  def _draw
    draw_texture(@texture, Vector2.new(0, 0))
    draw_texture_rect_region(@texture, @dest_rect, @src_rect)
    draw_string(@font, Vector2.new(200, 100), "hello world", Color.new(255, 0, 0))
  end
end

class Main
  def initialize(system_api, file_system_api)
    $system = system_api
    $file_system = file_system_api
    $resource_loader = system_api.resource_loader

    begin
      node = CustomNode.new()
      # node = Node2D.new
      $system.root.add_child node.source
    rescue => e
      $system.raise_exception e.message, e.inspect, e.backtrace
    end
  end

  def init_rgss
    rgss = RGSS.new
    rgss.load_scripts
  end

  def notification(what)
  end

  def process(dt)
  end
end
