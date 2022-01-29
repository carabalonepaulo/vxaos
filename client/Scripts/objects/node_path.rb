class NodePath
  attr_accessor :source

  def initialize(path)
    if path.class == 'Godot::NodePath'
      @source = path
    else
      @source = $system.create_godot_object 'NodePath', path
    end
  end
end