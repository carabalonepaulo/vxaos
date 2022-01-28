load_assembly 'IronRuby.Libraries', 'IronRuby.StandardLibrary.Zlib'

class Main
  def self.[](api_name)
    return @@apis[api_name]
  end

  def initialize(system_api, primitives_api, node_api, file_system_api)
    @@apis = {
      :system => system_api,
      :primitives => primitives_api,
      :node => node_api,
      :file_system => file_system_api
    }

    begin
      # create_test_ui
      load_rpg_maker_scripts
    rescue => e
      Main[:system].print e.inspect
    end
  end

  def load_rpg_maker_scripts
    scripts_file_path = './Data/Scripts.rvdata2'
    fs = Main[:file_system]
    sys = Main[:system]

    unless fs.exists scripts_file_path
      sys.error 'Scripts.rvdata2 not found.'
      return
    end

    buff = fs.read_all_bytes scripts_file_path
    Marshal.load(buff).each do |script|
      begin
        eval(Zlib::Inflate.inflate(script[2]).force_encoding('UTF-8'))
      rescue => e
        sys.error e.inspect
      end
    end
  end

  def create_test_ui
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

    txt_box = Main[:node].create_node 'LineEdit'
    txt_box.rect_min_size = Main[:primitives].vector2 150, 30
    btn1.emitter.bind('pressed') { p txt_box.text }

    hbox.add_child btn1
    hbox.add_child btn2
    hbox.add_child txt_box

    root = Main[:system].root
    root.add_child hbox
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
