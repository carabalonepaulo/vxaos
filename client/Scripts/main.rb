class CustomButton < Button
  def _ready
    print "from ready"
  end

  def _notification(what)
    # print what
  end

  def _process(dt)
    # print(dt)
  end
end

class Main
  def initialize(system_api, file_system_api)
    $system = system_api
    $file_system = file_system_api
    $resource_loader = system_api.resource_loader

    begin
      button = CustomButton.new
      button.text = "Click me!"
      button.connect("pressed") do
        print "hello from ruby"
      end

      $system.root.add_child button.source
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
