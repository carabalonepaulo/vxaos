class Main
  def initialize(system_api, file_system_api)
    $system = system_api
    $file_system = file_system_api

    begin
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
