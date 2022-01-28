module System
  def self.init(api)
    @@api = api
  end

  def self.root
    @@api.root
  end

  def self.create_node(name)
    @@api.create_node name
  end

  def self.create_godot_object(name, *args)
    @@api.create_godot_object name, *args
  end

  def self.print(*args)
    args.each { |arg| @@api.print arg.to_s }
  end

  def self.error(message)
    @@api.raise_exception message.to_s
  end

  def self.quit
    @@api.quit
  end
end