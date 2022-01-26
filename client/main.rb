class Emitter
  def initialize
    @callbacks = {}
  end

  def bind(signal, &callback)
    @callbacks[signal] ||= []
    @callbacks[signal].push callback
  end

  def emit(signal, *args)
    return if @callbacks[signal].nil?
    @callbacks[signal].each { |callback| callback.call(*args) }
  end
end

class Main
  def self.[](api_name)
    return @@apis[api_name]
  end

  def initialize(system_api, primitives_api, node_api)
    @@apis = {
      :system => system_api,
      :primitives => primitives_api,
      :node => node_api
    }

    begin
      hbox = Main[:node].create_node 'HorizontalBox'
      hbox.rect_min_size = Main[:primitives].vector2 640, 30

      btn1 = Main[:node].create_node 'Button'
      btn1.text = 'Click Me!'
      btn1.emitter.bind('pressed') { p 'button clicked' }
      btn1.emitter.bind('focus_entered') { p 'focus in' }
      btn1.emitter.bind('focus_exited') { p 'focus out' }

      btn2 = Main[:node].create_node 'Button'
      btn2.text = 'Toggle Me!'
      btn2.toggle_mode = true

      hbox.add_child btn1
      hbox.add_child btn2

      root = Main[:system].root
      root.add_child hbox
    rescue => e
      Main[:system].print e.inspect
    end
  end

  def p(message)
    Main[:system].print message.to_s
  end

  def on_pressed
    Main[:system].print 'from pressed signal'
  end

  def notification(what)
  end

  def process(dt)
  end
end
