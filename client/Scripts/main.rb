class Main
  def initialize(system_api, file_system_api)
    $system = system_api
    $file_system = file_system_api

    begin
      create_test_objects
      create_test_ui
    rescue => e
      $system.raise_exception e.inspect
    end
  end

  def create_test_objects
    vector = $system.create_godot_object 'Vector2', 10, 11
    $system.print vector.x
  end

  def create_test_ui
    hbox = $system.create_node 'HorizontalBox'
    hbox.rect_min_size = $system.create_godot_object 'Vector2', 640, 30

    btn1 = $system.create_node 'Button'
    btn1.text = 'Click Me!'
    btn1.emitter.bind('pressed') { $system.print 'button clicked' }
    btn1.emitter.bind('focus_entered') { $system.print 'focus in' }
    btn1.emitter.bind('focus_exited') { $system.print 'focus out' }

    btn2 = $system.create_node 'Button'
    btn2.text = 'Toggle Me!'
    btn2.toggle_mode = true

    btn3 = $system.create_node 'Button'
    btn3.text = 'From System'
    btn3.emitter.bind('pressed') { $system.quit }

    txt_box = $system.create_node 'LineEdit'
    txt_box.rect_min_size = $system.create_godot_object 'Vector2', 150, 30
    btn1.emitter.bind('pressed') { $system.print txt_box.text }

    hbox.add_child btn1
    hbox.add_child btn2
    hbox.add_child txt_box
    hbox.add_child btn3

    root = $system.root
    root.add_child hbox
  end

  def load_rpg_maker_scripts
  end

  def notification(what)
  end

  def process(dt)
  end
end
