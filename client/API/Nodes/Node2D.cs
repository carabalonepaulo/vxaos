using Godot;
using System;

namespace API.Nodes
{
    public class Node2D : Godot.Node2D, IRubyControl
    {
        public object Emitter { get; set; }

        object _rubyOwner;

        public Node2D(object rubyOwner)
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
            // Node
            Connect("ready", this, nameof(OnReady));
            Connect("renamed", this, nameof(OnRenamed));
            Connect("tree_entered", this, nameof(OnTreeEntered));
            Connect("tree_exited", this, nameof(OnTreeExited));
            Connect("tree_exiting", this, nameof(OnTreeExiting));

            // CanvasItem
            Connect("draw", this, nameof(OnDraw));
            Connect("hide", this, nameof(OnHide));
            Connect("item_rect_changed", this, nameof(OnItemRectChanged));
            Connect("visibility_changed", this, nameof(OnVisibilityChanged));

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

        public override void _Draw()
        {
            base._Draw();
            RubyEnvironment.Engine.Operations.InvokeMember(_rubyOwner, "_draw");
        }

        public override void _Notification(int what)
        {
            base._Notification(what);
            RubyEnvironment.Engine.Operations.InvokeMember(_rubyOwner, "_notification", what);
        }

        #region Node
        void OnReady() => RubyEnvironment.Engine.Operations.InvokeMember(Emitter, "emit", "ready");
        void OnRenamed() => RubyEnvironment.Engine.Operations.InvokeMember(Emitter, "emit", "renamed");
        void OnTreeEntered() => RubyEnvironment.Engine.Operations.InvokeMember(Emitter, "emit", "tree_entered");
        void OnTreeExited() => RubyEnvironment.Engine.Operations.InvokeMember(Emitter, "emit", "tree_exited");
        void OnTreeExiting() => RubyEnvironment.Engine.Operations.InvokeMember(Emitter, "emit", "tree_exiting");
        #endregion

        #region CanvasItem
        void OnDraw() => RubyEnvironment.Engine.Operations.InvokeMember(Emitter, "emit", "draw");
        void OnHide() => RubyEnvironment.Engine.Operations.InvokeMember(Emitter, "emit", "hide");
        void OnItemRectChanged() => RubyEnvironment.Engine.Operations.InvokeMember(Emitter, "emit", "item_rect_changed");
        void OnVisibilityChanged() => RubyEnvironment.Engine.Operations.InvokeMember(Emitter, "emit", "visibility_changed");
        #endregion
    }
}