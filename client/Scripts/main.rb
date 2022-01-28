class Main
  def self.[](api_name)
    return @@apis[api_name]
  end

  def initialize(system_api, primitives_api)
    @@apis = {
      :system => system_api,
      :primitives => primitives_api
    }

    begin
      create_test_objects
      create_test_ui
    rescue => e
      Main[:system].print e.inspect
    end
  end

  def create_test_objects
    vector = Main[:system].create_godot_object 'Vector2', 10, 11
    p vector.x
  end

  def create_test_ui
    hbox = Main[:system].create_node 'HorizontalBox'
    hbox.rect_min_size = Main[:primitives].vector2 640, 30

    btn1 = Main[:system].create_node 'Button'
    btn1.text = 'Click Me!'
    btn1.emitter.bind('pressed') { p 'button clicked' }
    btn1.emitter.bind('focus_entered') { p 'focus in' }
    btn1.emitter.bind('focus_exited') { p 'focus out' }

    btn2 = Main[:system].create_node 'Button'
    btn2.text = 'Toggle Me!'
    btn2.toggle_mode = true

    btn3 = Main[:system].create_node 'Button'
    btn3.text = 'From System'
    btn3.emitter.bind('pressed') { Main[:system].quit }

    txt_box = Main[:system].create_node 'LineEdit'
    txt_box.rect_min_size = Main[:primitives].vector2 150, 30
    btn1.emitter.bind('pressed') { p txt_box.text }

    hbox.add_child btn1
    hbox.add_child btn2
    hbox.add_child txt_box
    hbox.add_child btn3

    root = Main[:system].root
    root.add_child hbox
  end

  def p(message)
    Main[:system].print message.to_s
  end

  def load_rpg_maker_scripts
  end

  def notification(what)
  end

  def process(dt)
  end
end
