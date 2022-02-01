using Godot;
using System;

namespace API.Nodes
{
    public class TextEdit : Godot.TextEdit, IRubyControl
    {
        public object Emitter { get; set; }

        public TextEdit()
        {
            var ctor = RubyEnvironment.Engine.Runtime.Globals.GetVariable("Emitter");
            Emitter = RubyEnvironment.Engine.Operations.CreateInstance(ctor);
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

            // Control
            Connect("focus_entered", this, nameof(OnFocusEntered));
            Connect("focus_exited", this, nameof(OnFocusExited));
            Connect("gui_input", this, nameof(OnGuiInput));
            Connect("minimum_size_changed", this, nameof(OnMinimumSizeChanged));
            Connect("modal_closed", this, nameof(OnModalClose));
            Connect("mouse_entered", this, nameof(OnMouseEntered));
            Connect("mouse_exited", this, nameof(OnMouseExited));
            Connect("resized", this, nameof(OnResized));
            Connect("size_flags_changed", this, nameof(OnSizeFlagsChanged));

            // TextEdit
            Connect("breakpoint_toggled", this, nameof(OnBreakpointToggled));
            Connect("cursor_changed", this, nameof(OnCursorChanged));
            Connect("info_clicked", this, nameof(OnInfoClicked));
            Connect("request_completion", this, nameof(OnRequestCompletion));
            Connect("symbol_lookup", this, nameof(OnSymbolLookup));
            Connect("text_changed", this, nameof(OnTextChanged));
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

        #region Control
        void OnFocusEntered() => RubyEnvironment.Engine.Operations.InvokeMember(Emitter, "emit", "focus_entered");
        void OnFocusExited() => RubyEnvironment.Engine.Operations.InvokeMember(Emitter, "emit", "focus_exited");
        void OnGuiInput(InputEvent e) => RubyEnvironment.Engine.Operations.InvokeMember(Emitter, "emit", "gui_input", e);
        void OnMinimumSizeChanged() => RubyEnvironment.Engine.Operations.InvokeMember(Emitter, "emit", "minimum_size_changed");
        void OnModalClose() => RubyEnvironment.Engine.Operations.InvokeMember(Emitter, "emit", "modal_close");
        void OnMouseEntered() => RubyEnvironment.Engine.Operations.InvokeMember(Emitter, "emit", "mouse_entered");
        void OnMouseExited() => RubyEnvironment.Engine.Operations.InvokeMember(Emitter, "emit", "mouse_exited");
        void OnResized() => RubyEnvironment.Engine.Operations.InvokeMember(Emitter, "emit", "resized");
        void OnSizeFlagsChanged() => RubyEnvironment.Engine.Operations.InvokeMember(Emitter, "emit", "size_flags_changed");
        #endregion

        #region TextEdit
        void OnBreakpointToggled(int row) => RubyEnvironment.Engine.Operations.InvokeMember(Emitter, "emit", "breakpoint_toggled", row);
        void OnCursorChanged() => RubyEnvironment.Engine.Operations.InvokeMember(Emitter, "emit", "cursor_changed");
        void OnInfoClicked(int row, string info) => RubyEnvironment.Engine.Operations.InvokeMember(
            Emitter,
            "emit",
            "info_clicked",
            row,
            info);
        void OnRequestCompletion() => RubyEnvironment.Engine.Operations.InvokeMember(Emitter, "emit", "request_completion");
        void OnSymbolLookup(string symbol, int row, int column) => RubyEnvironment.Engine.Operations.InvokeMember(
            Emitter,
            "emit",
            "symbol_lookup",
            symbol, row, column);
        void OnTextChanged() => RubyEnvironment.Engine.Operations.InvokeMember(Emitter, "emit", "text_changed");
        #endregion
    }
}