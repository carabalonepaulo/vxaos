class Node < Wrapper
  def initialize(source = nil)
    init_source source, 'Node'
    init_inner_control_vars
  end

  def init_source(source, name, godot_name = nil)
    @source = source.class.to_s == "Godot::#{godot_name || name}" ? source : $system.create_control(name, self)
  end

  def init_inner_control_vars
    @viewport = Viewport.new @source.get_viewport
    # TODO: @multiplayer_api = MultiplayerAPI.new @source.multiplayer
    # TODO: @custom_multiplayer = MultiplayerAPI.new @source.custom_multiplayer
  end

  def emit(signal, *args)
    @source.emitter.emit(signal, *args)
  end

  def connect(signal, &callback)
    @source.emitter.bind(signal, &callback)
  end

  def pause_mode
    @source.pause_mode
  end

  def pause_mode=(v)
    @source.pause_mode = v
  end

  def name
    @source.name
  end

  def name=(v)
    @source.name = v
  end

  def file_name
    @source.filename
  end

  def file_name=(v)
    @source.filename = v
  end

  def owner
    @owner
  end

  def owner=(v)
    @owner = v
    @owner.source = v.source
  end

  def multiplayer
    raise 'not implemented yet'
  end

  def multiplayer=(v)
    raise 'not implemented yet'
  end

  def custom_multiplayer
    raise 'not implemented yet'
  end

  def custom_multiplayer=(v)
    raise 'not implemented yet'
  end

  def process_priority
    @source.process_priority
  end

  def process_priority=(v)
    @source.process_priority = v
  end

  def _enter_tree
    @owner = Node.new @source.owner
    @path = NodePath.new @source.get_path
  end

  def _exit_tree
    @owner = nil
    @path = nil
  end

  def _get_configuration_warning
  end

  def _notification(what)
  end

  def _input(event)
  end

  def _physics_process(delta)
  end

  def _process(delta)
  end

  def _ready
  end

  def _unhandled_input(event)
  end

  def _unhandled_key_input(event)
  end

  def add_child(node)
    @source.add_child node.source
  end

  def add_child_below_node(node, child_node, legible_unique_name)
    @source.add_child_below_node node.source, child_node.source, legible_unique_name
  end

  def add_to_group(group, presistent)
    @source.add_to_group group, presistent
  end

  def can_process?
    @source.can_process
  end

  def duplicate(flags)
    Node.new(@source.duplicate(flags))
  end

  def find_node(mask, recursive = true, owned = true)
    Node.new(@source.find_node(mask, recursive, owned))
  end

  def find_parent(mask)
    Node.new(@source.find_parent(mask))
  end

  def get_child(id)
    Node.new(@source.get_child(id))
  end

  def get_child_count
    @source.get_child_count
  end

  def get_children
    @source.get_children.map { |child| Node.new child }
  end

  def get_groups
    @source.get_groups
  end

  def get_index
    @source.get_index
  end

  def get_network_master
    @source.get_network_master
  end

  def get_node(path)
    Node.new(@source.get_node(path.source))
  end

  def get_node_and_resource(path)
    raise 'not implemented yet'
  end

  def get_node_or_null(path)
    node = @source.get_node_or_null path.source
    node.nil? ? nil : Node.new(node)
  end

  def get_parent
    Node.new @source.get_parent
  end

  def get_path
    @path
  end

  def get_path_to(node)
    NodePath.new(@source.get_path_to(node.source))
  end

  def get_physics_process_delta_time
    @source.get_physics_process_delta_time
  end

  def get_position_in_parent
    @source.get_position_in_parent
  end

  def get_process_delta_time
    @source.get_process_delta_time
  end

  def get_scene_instance_load_placeholder
    @source.get_scene_instance_load_placeholder
  end

  def get_tree
    @scene_tree
  end

  def get_viewport
    @viewport
  end

  def has_node(path)
    @source.has_node path.source
  end

  def has_node_and_resource(path)
    @source.has_node_and_resource path.source
  end

  def is_parent_of?(node)
    @source.is_parent_of node.source
  end

  def is_displayed_folded?
    @source.is_displayed_folded
  end

  def is_greater_than?(node)
    @source.is_greater_than node.source
  end

  def is_in_group?(group)
    @source.is_in_group group
  end

  def is_inside_tree?
    @source.is_inside_tree
  end

  def is_network_master?
    @source.is_network_master
  end

  def is_physics_processing?
    @source.is_physics_processing
  end

  def is_physics_processing_internal?
    @source.is_physics_processing_internal
  end

  def is_processing?
    @source.is_processing
  end

  def is_processing_input?
    @source.is_processing_input
  end

  def is_processing_internal?
    @source.is_processing_internal
  end

  def is_processing_unhandled_input?
    @source.is_processing_unhandled_input
  end

  def is_processing_unhandled_key_input?
    @source.is_processing_unhandled_key_input
  end

  def move_child(child_node, to_position)
    @source.move_child child_node.source, to_position
  end

  def print_stray_nodes
    @source.print_stray_nodes
  end

  def print_tree
    @source.print_tree
  end

  def print_tree_pretty
    @source.print_tree_pretty
  end

  def propagate_call(method, args = [], parent_first = false)
    raise 'not implemented yet'
  end

  def propagate_notification(what)
    @source.propagate_notification what
  end

  def queue_free
    @source.queue_free
  end

  def raise
    @source.raise
  end

  def remove_and_skip
    @source.remove_and_skip
  end

  def remove_child(node)
    @source.remove_child node.source
  end

  def remove_from_group(group)
    @source.remove_from_group group
  end

  def replace_by(node, keep_data = false)
    @source.replace_by node.source, keep_data
  end

  def request_ready
    @source.request_ready
  end

  def rpc(method, *args)
    raise 'not implemented yet'
  end

  def rpc_config(method, mode)
    raise 'not implemented yet'
  end

  def rpc_id(peer_id, method, *args)
    raise 'not implemented yet'
  end

  def rpc_unreliable(method, *args)
    raise 'not implemented yet'
  end

  def rpc_unreliable_id(peer_id, method, *args)
    raise 'not implemented yet'
  end

  def rset(property, value)
    raise 'not implemented yet'
  end

  def rset_config(property, mode)
    raise 'not implemented yet'
  end

  def rset_id(peer_id, property, value)
    raise 'not implemented yet'
  end

  def rset_unreliable(property, value)
    raise 'not implemented yet'
  end

  def rset_unreliable_id(peer_id, property, value)
    raise 'not implemented yet'
  end

  def set_display_folded(fold)
    @source.set_display_folded fold
  end

  def set_network_master(id, recursive = true)
    @source.set_network_master id, recursive
  end

  def set_physics_process(enable)
    @source.set_physics_process enable
  end

  def set_physics_process_internal(enable)
    @source.set_physics_process_internal enable
  end

  def set_process(enable)
    @source.set_process enable
  end

  def set_process_input(enable)
    @source.set_process_input enable
  end

  def set_process_internal(enable)
    @source.set_process_internal enable
  end

  def set_process_unhandled_input(enable)
    @source.set_process_unhandled_input enable
  end

  def set_process_unhandled_key_input(enable)
    @source.set_process_unhandled_key_input enable
  end

  def set_scene_instance_load_placeholder(load_placeholder)
    @source.set_scene_instance_load_placeholder load_placeholder
  end

  def update_configuration_warning
    @source.update_configuration_warning
  end
end