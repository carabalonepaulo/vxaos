using System;
using Godot;

namespace API.Controls
{
    public class Node : Godot.Node, IRubyControl
    {
        public object Emitter;
        object _rubyOwner;

        public Node(object rubyOwner)
        {
            _rubyOwner = rubyOwner;
            var ctor = RubyEnvironment.Engine.Runtime.Globals.GetVariable("Emitter");
            Emitter = RubyEnvironment.Engine.Operations.CreateInstance(ctor);
        }

        public override void _EnterTree()
        {
            base._EnterTree();
            RubyEnvironment.Engine.Operations.InvokeMember(_rubyOwner, "_enter_tree");
        }

        public override void _ExitTree()
        {
            base._ExitTree();
            RubyEnvironment.Engine.Operations.InvokeMember(_rubyOwner, "_exit_tree");
        }

        public override void _Input(InputEvent @event)
        {
            base._Input(@event);
            RubyEnvironment.Engine.Operations.InvokeMember(_rubyOwner, "_input", @event);
        }

        public override void _PhysicsProcess(float delta)
        {
            base._PhysicsProcess(delta);
            RubyEnvironment.Engine.Operations.InvokeMember(_rubyOwner, "_physics_process", delta);
        }

        public override void _Process(float delta)
        {
            base._Process(delta);
            RubyEnvironment.Engine.Operations.InvokeMember(_rubyOwner, "_process", delta);
        }

        public override void _Ready()
        {
            base._Ready();
            RubyEnvironment.Engine.Operations.InvokeMember(_rubyOwner, "_ready");
        }

        public override void _UnhandledInput(InputEvent @event)
        {
            base._UnhandledInput(@event);
            RubyEnvironment.Engine.Operations.InvokeMember(_rubyOwner, "_unhandled_input", @event);
        }

        public override void _UnhandledKeyInput(InputEventKey @event)
        {
            base._UnhandledKeyInput(@event);
            RubyEnvironment.Engine.Operations.InvokeMember(_rubyOwner, "_unhandled_key_input", @event);
        }

        public override void _Notification(int what)
        {
            base._Notification(what);
            RubyEnvironment.Engine.Operations.InvokeMember(_rubyOwner, "_notification", what);
        }
    }
}