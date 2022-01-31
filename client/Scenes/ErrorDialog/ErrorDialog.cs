using Godot;
using System;

public class ErrorDialog : CenterContainer
{
    Node _backtraceContainer;
    Button _closeButton;
    Label _message;
    Label _inspect;

    public override void _Ready()
    {
        _backtraceContainer = GetNode("Panel/MarginContainer/VBoxContainer/ScrollContainer/C1/R1/C1");
        _closeButton = GetNode<Button>("Panel/MarginContainer/VBoxContainer/R2/Close");
        _closeButton.Connect("pressed", this, nameof(OnClosePressed));
        _message = GetNode<Label>("Panel/MarginContainer/VBoxContainer/ScrollContainer/C1/Message");
        _inspect = GetNode<Label>("Panel/MarginContainer/VBoxContainer/ScrollContainer/C1/Inspect");
    }

    public void RaiseException(string message, string inspect, string[] lines)
    {
        foreach (string line in lines)
        {
            var label = new Label();
            label.Text = line;
            _backtraceContainer.AddChild(label);
        }
        _message.Text = message;
        _inspect.Text = inspect;
        _closeButton.GrabFocus();
        Visible = true;
    }

    void OnClosePressed()
    {
        GetTree().Quit();
    }
}
