class Main
  def initialize(system_api, file_system_api)
    System.init system_api
    FileSystem.init file_system_api

    begin
      create_test_objects
      create_test_ui
    rescue => e
      System.error e.inspect
    end
  end

  def create_test_objects
    vector = System.create_godot_object 'Vector2', 10, 11
    System.print vector.x
  end

  def create_test_ui
    hbox = System.create_node 'HorizontalBox'
    hbox.rect_min_size = System.create_godot_object 'Vector2', 640, 30

    btn1 = System.create_node 'Button'
    btn1.text = 'Click Me!'
    btn1.emitter.bind('pressed') { System.print 'button clicked' }
    btn1.emitter.bind('focus_entered') { System.print 'focus in' }
    btn1.emitter.bind('focus_exited') { System.print 'focus out' }

    btn2 = System.create_node 'Button'
    btn2.text = 'Toggle Me!'
    btn2.toggle_mode = true

    btn3 = System.create_node 'Button'
    btn3.text = 'From System'
    btn3.emitter.bind('pressed') { System.quit }

    txt_box = System.create_node 'LineEdit'
    txt_box.rect_min_size = System.create_godot_object 'Vector2', 150, 30
    btn1.emitter.bind('pressed') { System.print txt_box.text }

    hbox.add_child btn1
    hbox.add_child btn2
    hbox.add_child txt_box
    hbox.add_child btn3

    root = System.root
    root.add_child hbox
  end

  def load_rpg_maker_scripts
  end

  def notification(what)
  end

  def process(dt)
  end
end
