class NodePath
  attr_accessor :source

  def initialize(path)
    @source = path.class.to_s == 'Godot::NodePath' ? path : $system.create_godot_object('NodePath', path)
  end
end